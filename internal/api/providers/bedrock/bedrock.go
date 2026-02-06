package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func init() {
	api.RegisterProvider("bedrock", func(_ string) (api.Provider, error) {
		// apiKey は使用しない（AWS 認証チェーン使用）
		return New()
	})
}

const (
	defaultRegion           = "us-east-1"
	defaultModel            = "global.anthropic.claude-opus-4-5-20251101-v1:0"
	bedrockAnthropicVersion = "bedrock-2023-05-31"
)

// Provider は AWS Bedrock (Anthropic Claude) のプロバイダー実装
type Provider struct {
	client        *bedrockruntime.Client
	region        string
	mcpTools      []api.OpenAIToolFunction // MCP ツール定義
	usageCallback api.UsageCallback        // トークン使用量コールバック
}

// New は新しい Bedrock Provider を作成
func New() (*Provider, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = defaultRegion
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("AWS config load failed: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	return &Provider{
		client: client,
		region: region,
	}, nil
}

// Name はプロバイダー名を返す
func (p *Provider) Name() string {
	return "Bedrock"
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("BEDROCK_FUNCTION_CALLING") != "0"
}

// BedrockRequest は Bedrock InvokeModel 用リクエスト
// Claude API とは異なり anthropic_version をボディに含み、model/stream フィールドは不要
type BedrockRequest struct {
	AnthropicVersion string                    `json:"anthropic_version"`
	MaxTokens        int                       `json:"max_tokens"`
	System           interface{}               `json:"system,omitempty"`
	Messages         []claude.AnthropicMessage `json:"messages"`
	Thinking         *claude.ThinkingConfig    `json:"thinking,omitempty"`
	Tools            []claude.ClaudeTool       `json:"tools,omitempty"`
}

// BedrockMultimodalRequest はマルチモーダル（画像付き）リクエスト
type BedrockMultimodalRequest struct {
	AnthropicVersion string                 `json:"anthropic_version"`
	MaxTokens        int                    `json:"max_tokens"`
	System           interface{}            `json:"system,omitempty"`
	Messages         []interface{}          `json:"messages"`
	Thinking         *claude.ThinkingConfig `json:"thinking,omitempty"`
	Tools            []claude.ClaudeTool    `json:"tools,omitempty"`
}

// buildSystemField はプロンプトキャッシュ対応のシステムフィールドを構築
func buildSystemField(systemPrompt string) interface{} {
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		return systemPrompt
	}
	if !cfg.PromptCache.Enabled {
		return systemPrompt
	}

	return []claude.SystemBlock{
		{
			Type: "text",
			Text: systemPrompt,
			CacheControl: &claude.CacheControl{
				Type: "ephemeral",
			},
		},
	}
}

// ChatWithTools は Provider interface の実装
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	model = api.GetDefaultModel(model, "bedrock", defaultModel)

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	messages := claude.ConvertToAnthropicMessages(history)

	cfg := config.GetGlobalConfig()

	reqBody := BedrockRequest{
		AnthropicVersion: bedrockAnthropicVersion,
		MaxTokens:        api.GetMaxOutputTokens("bedrock", 32768),
		System:           buildSystemField(systemPrompt),
		Messages:         messages,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Thinking = &claude.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: api.LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	// Tool Use: ツール定義を追加
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = claude.GetCombinedClaudeTools(p.mcpTools)
	}

	return p.invokeStream(ctx, model, reqBody)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常の ChatWithTools を使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	model = api.GetDefaultModel(model, "bedrock", defaultModel)

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	converted := claude.ConvertToAnthropicMessages(history)
	var messages []interface{}
	for _, msg := range converted {
		messages = append(messages, msg)
	}

	// 画像付きユーザーメッセージを追加
	multimodalMessage := claude.MultimodalMessage{
		Role: "user",
		Content: []claude.ContentPart{
			{
				Type: "image",
				Source: &claude.ImageSource{
					Type:      "base64",
					MediaType: image.MediaType,
					Data:      image.Base64,
				},
			},
			{
				Type: "text",
				Text: userMessage,
			},
		},
	}
	messages = append(messages, multimodalMessage)

	cfg := config.GetGlobalConfig()

	reqBody := BedrockMultimodalRequest{
		AnthropicVersion: bedrockAnthropicVersion,
		MaxTokens:        api.GetMaxOutputTokens("bedrock", 32768),
		System:           buildSystemField(systemPrompt),
		Messages:         messages,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Thinking = &claude.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: api.LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	// Tool Use: ツール定義を追加
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = claude.GetCombinedClaudeTools(p.mcpTools)
	}

	return p.invokeStream(ctx, model, reqBody)
}

// invokeStream は Bedrock InvokeModelWithResponseStream を呼び出す共通処理
func (p *Provider) invokeStream(ctx context.Context, model string, reqBody interface{}) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("request marshal failed: %w", err)
	}

	spinner := api.StartThinkingSpinner(false, "")

	output, err := p.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(model),
		ContentType: aws.String("application/json"),
		Body:        jsonBody,
	})
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("bedrock API error: %w", err)
	}

	return p.handleEventStream(ctx, output, spinner)
}

// SetMCPTools は MCP ツール定義を設定する
func (p *Provider) SetMCPTools(tools []api.OpenAIToolFunction) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}
