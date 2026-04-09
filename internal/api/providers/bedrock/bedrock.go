package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	defaultModel            = "global.anthropic.claude-sonnet-4-6-v1"
	bedrockAnthropicVersion = "bedrock-2023-05-31"
)

// Provider は AWS Bedrock (Anthropic Claude) のプロバイダー実装
type Provider struct {
	client        *bedrockruntime.Client
	region        string
	mcpTools      []api.ToolDefinition // MCP ツール定義
	usageCallback api.UsageCallback
	runtimeConfig *config.Config
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

// isBedrockCompactionSupported はモデルが Compaction に対応しているか判定
func isBedrockCompactionSupported(model string) bool {
	return strings.Contains(model, "opus-4-6") || strings.Contains(model, "opus-4-5") ||
		strings.Contains(model, "sonnet-4-6")
}

func buildBedrockContextManagement(model string, compression config.CompressionConfig, betaHeaders []string) (*claude.ContextManagement, []string) {
	contextManagement := claude.BuildContextManagement(compression, isBedrockCompactionSupported(model))
	if contextManagement == nil {
		return nil, betaHeaders
	}

	return contextManagement, claude.MergeAnthropicBetaHeaders(betaHeaders, contextManagement)
}

// SupportsClaudeCompaction は Claude Compaction 対応状況を返す
func (p *Provider) SupportsClaudeCompaction() bool {
	return p.supportsClaudeCompactionWithConfig(p.effectiveConfig(), "")
}

// SupportsClaudeCompactionWithContext は request context とモデルを使って Claude Compaction 対応可否を返す。
func (p *Provider) SupportsClaudeCompactionWithContext(ctx context.Context, model string) bool {
	cfg := p.effectiveConfig()
	if ctxCfg, ok := config.LookupContext(ctx); ok {
		cfg = ctxCfg
	}
	return p.supportsClaudeCompactionWithConfig(cfg, model)
}

// SetRuntimeConfig は provider が参照する runtime 設定を差し替える。
func (p *Provider) SetRuntimeConfig(cfg *config.Config) {
	p.runtimeConfig = cfg
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
	AnthropicVersion  string                    `json:"anthropic_version"`
	AnthropicBeta     []string                  `json:"anthropic_beta,omitempty"`
	CacheControl      *api.CacheControl         `json:"cache_control,omitempty"`
	MaxTokens         int                       `json:"max_tokens"`
	System            interface{}               `json:"system,omitempty"` // can be string or []api.SystemBlock
	Messages          []claude.AnthropicMessage `json:"messages"`
	Thinking          *claude.ThinkingConfig    `json:"thinking,omitempty"`
	Tools             []claude.ClaudeTool       `json:"tools,omitempty"`
	ContextManagement *claude.ContextManagement `json:"context_management,omitempty"`
}

// BedrockMultimodalRequest はマルチモーダル（画像付き）リクエスト
type BedrockMultimodalRequest struct {
	AnthropicVersion  string                    `json:"anthropic_version"`
	AnthropicBeta     []string                  `json:"anthropic_beta,omitempty"`
	CacheControl      *api.CacheControl         `json:"cache_control,omitempty"`
	MaxTokens         int                       `json:"max_tokens"`
	System            interface{}               `json:"system,omitempty"`
	Messages          []interface{}             `json:"messages"`
	Thinking          *claude.ThinkingConfig    `json:"thinking,omitempty"`
	Tools             []claude.ClaudeTool       `json:"tools,omitempty"`
	ContextManagement *claude.ContextManagement `json:"context_management,omitempty"`
}

// ChatWithTools は Provider interface の実装
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	model = api.GetDefaultModelWithContext(ctx, model, "bedrock", defaultModel)

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	messages := claude.ConvertToAnthropicMessages(history)

	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	pCfg, _ := cfg.GetProviderModelConfig("bedrock")

	// Anthropic Version（config → フォールバック定数）
	version := pCfg.AnthropicVersion
	if version == "" {
		version = bedrockAnthropicVersion
	}

	reqBody := BedrockRequest{
		AnthropicVersion: version,
		AnthropicBeta:    pCfg.AnthropicBeta,
		MaxTokens:        api.GetMaxOutputTokens(ctx, "bedrock", model),
		System:           api.BuildSystemFieldWithConfig(systemPrompt, cfg),
		Messages:         messages,
	}
	if cfg.PromptCache.Enabled {
		reqBody.CacheControl = api.NewCacheControlWithConfig(cfg)
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Thinking = &claude.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: api.LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	// Tool Use: ツール定義を追加
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = claude.GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	reqBody.ContextManagement, reqBody.AnthropicBeta = buildBedrockContextManagement(model, cfg.Compression, reqBody.AnthropicBeta)

	return p.invokeStream(ctx, model, reqBody)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常の ChatWithTools を使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	model = api.GetDefaultModelWithContext(ctx, model, "bedrock", defaultModel)

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	converted := claude.ConvertToAnthropicMessages(history)

	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

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

	pCfg, _ := cfg.GetProviderModelConfig("bedrock")

	// Anthropic Version（config → フォールバック定数）
	version := pCfg.AnthropicVersion
	if version == "" {
		version = bedrockAnthropicVersion
	}

	reqBody := BedrockMultimodalRequest{
		AnthropicVersion: version,
		AnthropicBeta:    pCfg.AnthropicBeta,
		MaxTokens:        api.GetMaxOutputTokens(ctx, "bedrock", model),
		System:           api.BuildSystemFieldWithConfig(systemPrompt, cfg),
		Messages:         messages,
	}
	if cfg.PromptCache.Enabled {
		reqBody.CacheControl = api.NewCacheControlWithConfig(cfg)
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Thinking = &claude.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: api.LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	// Tool Use: ツール定義を追加
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = claude.GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	reqBody.ContextManagement, reqBody.AnthropicBeta = buildBedrockContextManagement(model, cfg.Compression, reqBody.AnthropicBeta)

	return p.invokeStream(ctx, model, reqBody)
}

func (p *Provider) effectiveConfig() *config.Config {
	if p != nil && p.runtimeConfig != nil {
		return p.runtimeConfig
	}
	return config.DefaultConfig()
}

func (p *Provider) supportsClaudeCompactionWithConfig(cfg *config.Config, model string) bool {
	if cfg == nil || !cfg.Compression.ClaudeCompaction {
		return false
	}
	if model == "" {
		model = cfg.GetEffectiveModelForProvider("bedrock")
	}
	if model == "" {
		model = defaultModel
	}
	return isBedrockCompactionSupported(model)
}

// invokeStream は Bedrock InvokeModelWithResponseStream を呼び出す共通処理
func (p *Provider) invokeStream(ctx context.Context, model string, reqBody interface{}) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("request marshal failed: %w", err)
	}

	spinner := api.StartThinkingSpinner(ctx, false, "")

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
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetMCPEnabled はMCPが有効かどうかを設定する（レガシー、互換性のため）
func (p *Provider) SetMCPEnabled(enabled bool) {
	// BedrockプロバイダーではMCP有効/無効の切り替えは不要
	// 常にFunction Calling経由でMCPツールを使用可能
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}
