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
	defaultModel            = "global.anthropic.claude-sonnet-4-6"
	bedrockAnthropicVersion = "bedrock-2023-05-31"
	bedrockEffortBetaHeader = "effort-2025-11-24"
)

// Provider は AWS Bedrock のプロバイダー実装。
type Provider struct {
	client            invokeModelWithResponseStreamClient
	converseClient    converseStreamClient
	region            string
	mcpTools          []api.ToolDefinition // MCP ツール定義
	usageCallback     api.UsageCallback
	runtimeConfig     *config.Config
	lastContentBlocks []api.AnthropicContentBlock
}

type invokeModelWithResponseStreamClient interface {
	InvokeModelWithResponseStream(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error)
}

type converseStreamClient interface {
	ConverseStream(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error)
}

// New は新しい Bedrock Provider を作成
func New() (*Provider, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{}
	if region := explicitAWSRegionFromEnv(); region != "" {
		loadOptions = append(loadOptions, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("AWS config load failed: %w", err)
	}
	if strings.TrimSpace(cfg.Region) == "" {
		cfg.Region = defaultRegion
	}

	client := bedrockruntime.NewFromConfig(cfg)

	return &Provider{
		client:         client,
		converseClient: client,
		region:         cfg.Region,
	}, nil
}

func explicitAWSRegionFromEnv() string {
	if region := strings.TrimSpace(os.Getenv("AWS_REGION")); region != "" {
		return region
	}
	return strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
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

func mergeBedrockOutputBetaHeaders(headers []string, outputConfig *claude.OutputConfig) []string {
	if outputConfig == nil || strings.TrimSpace(outputConfig.Effort) == "" {
		return headers
	}
	return appendUniqueString(headers, bedrockEffortBetaHeader)
}

func appendUniqueString(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
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

// BedrockClaudeMessagesRequest は Bedrock InvokeModel 用の Claude Messages リクエスト。
// Claude API とは異なり anthropic_version をボディに含み、model/stream フィールドは不要
type BedrockClaudeMessagesRequest struct {
	AnthropicVersion  string                    `json:"anthropic_version"`
	AnthropicBeta     []string                  `json:"anthropic_beta,omitempty"`
	CacheControl      *api.CacheControl         `json:"cache_control,omitempty"`
	MaxTokens         int                       `json:"max_tokens"`
	System            interface{}               `json:"system,omitempty"` // can be string or []api.SystemBlock
	Messages          []claude.AnthropicMessage `json:"messages"`
	Thinking          *claude.ThinkingConfig    `json:"thinking,omitempty"`
	OutputConfig      *claude.OutputConfig      `json:"output_config,omitempty"`
	Tools             []claude.ClaudeTool       `json:"tools,omitempty"`
	ContextManagement *claude.ContextManagement `json:"context_management,omitempty"`
}

// BedrockClaudeMultimodalRequest は Bedrock InvokeModel 用の Claude 画像付きリクエスト。
type BedrockClaudeMultimodalRequest struct {
	AnthropicVersion  string                    `json:"anthropic_version"`
	AnthropicBeta     []string                  `json:"anthropic_beta,omitempty"`
	CacheControl      *api.CacheControl         `json:"cache_control,omitempty"`
	MaxTokens         int                       `json:"max_tokens"`
	System            interface{}               `json:"system,omitempty"`
	Messages          []interface{}             `json:"messages"`
	Thinking          *claude.ThinkingConfig    `json:"thinking,omitempty"`
	OutputConfig      *claude.OutputConfig      `json:"output_config,omitempty"`
	Tools             []claude.ClaudeTool       `json:"tools,omitempty"`
	ContextManagement *claude.ContextManagement `json:"context_management,omitempty"`
}

func buildBedrockThinkingConfig(model, level string) (*claude.ThinkingConfig, *claude.OutputConfig) {
	if claude.IsAdaptiveThinkingModel(model) {
		thinking := &claude.ThinkingConfig{Type: "adaptive"}
		outputConfig := &claude.OutputConfig{Effort: claude.LevelToEffort(level, model)}
		return thinking, outputConfig
	}
	return &claude.ThinkingConfig{
		Type:         "enabled",
		BudgetTokens: api.LevelToBudgetTokens(level),
	}, nil
}

// ChatWithTools は Provider interface の実装
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.lastContentBlocks = nil
	req := p.resolveBedrockRequestContext(ctx, model)

	switch req.route {
	case bedrockRouteClaudeMessages:
		return p.chatWithClaudeMessages(ctx, systemPrompt, history, req)
	case bedrockRouteConverseStream:
		return p.chatWithConverseStream(ctx, systemPrompt, history, "", nil, req)
	default:
		return "", fmt.Errorf("unsupported bedrock route %q for model=%q catalog_model=%q", req.route, req.model, req.catalogModel)
	}
}

func (p *Provider) chatWithClaudeMessages(ctx context.Context, systemPrompt string, history []api.Message, req bedrockRequestContext) (string, error) {
	if err := ensureBedrockClaudeMessagesRoute(req); err != nil {
		return "", err
	}

	messages := claude.ConvertToAnthropicMessagesWithThinking(history, api.IsThinkingEnabled(ctx))

	// Anthropic Version（config → フォールバック定数）
	version := req.providerConfig.AnthropicVersion
	if version == "" {
		version = bedrockAnthropicVersion
	}

	reqBody := BedrockClaudeMessagesRequest{
		AnthropicVersion: version,
		AnthropicBeta:    req.providerConfig.AnthropicBeta,
		MaxTokens:        api.GetMaxOutputTokens(ctx, "bedrock", req.model),
		System:           api.BuildSystemFieldWithConfig(systemPrompt, req.cfg),
		Messages:         messages,
	}
	if req.cfg.PromptCache.Enabled {
		reqBody.CacheControl = api.NewCacheControlWithConfig(req.cfg)
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Thinking, reqBody.OutputConfig = buildBedrockThinkingConfig(req.catalogModel, req.cfg.Thinking.Level)
	}

	// Tool Use: ツール定義を追加
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = claude.GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	reqBody.ContextManagement, reqBody.AnthropicBeta = buildBedrockContextManagement(req.catalogModel, req.cfg.Compression, reqBody.AnthropicBeta)
	reqBody.AnthropicBeta = mergeBedrockOutputBetaHeaders(reqBody.AnthropicBeta, reqBody.OutputConfig)

	return p.invokeClaudeMessagesStream(ctx, req.model, reqBody)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	p.lastContentBlocks = nil

	// 画像がない場合は通常の ChatWithTools を使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	req := p.resolveBedrockRequestContext(ctx, model)
	switch req.route {
	case bedrockRouteClaudeMessages:
		return p.chatWithClaudeImage(ctx, systemPrompt, history, userMessage, image, req)
	case bedrockRouteConverseStream:
		return p.chatWithConverseStream(ctx, systemPrompt, history, userMessage, image, req)
	default:
		return "", fmt.Errorf("unsupported bedrock route %q for model=%q catalog_model=%q", req.route, req.model, req.catalogModel)
	}
}

func (p *Provider) chatWithClaudeImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, req bedrockRequestContext) (string, error) {
	if err := ensureBedrockClaudeMessagesRoute(req); err != nil {
		return "", err
	}

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	converted := claude.ConvertToAnthropicMessagesWithThinking(history, api.IsThinkingEnabled(ctx))

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

	// Anthropic Version（config → フォールバック定数）
	version := req.providerConfig.AnthropicVersion
	if version == "" {
		version = bedrockAnthropicVersion
	}

	reqBody := BedrockClaudeMultimodalRequest{
		AnthropicVersion: version,
		AnthropicBeta:    req.providerConfig.AnthropicBeta,
		MaxTokens:        api.GetMaxOutputTokens(ctx, "bedrock", req.model),
		System:           api.BuildSystemFieldWithConfig(systemPrompt, req.cfg),
		Messages:         messages,
	}
	if req.cfg.PromptCache.Enabled {
		reqBody.CacheControl = api.NewCacheControlWithConfig(req.cfg)
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Thinking, reqBody.OutputConfig = buildBedrockThinkingConfig(req.catalogModel, req.cfg.Thinking.Level)
	}

	// Tool Use: ツール定義を追加
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = claude.GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	reqBody.ContextManagement, reqBody.AnthropicBeta = buildBedrockContextManagement(req.catalogModel, req.cfg.Compression, reqBody.AnthropicBeta)
	reqBody.AnthropicBeta = mergeBedrockOutputBetaHeaders(reqBody.AnthropicBeta, reqBody.OutputConfig)

	return p.invokeClaudeMessagesStream(ctx, req.model, reqBody)
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
	return isBedrockCompactionSupported(cfg.ModelCatalogName("bedrock", model))
}

// invokeClaudeMessagesStream は Claude Messages payload を Bedrock InvokeModelWithResponseStream へ送る。
func (p *Provider) invokeClaudeMessagesStream(ctx context.Context, model string, reqBody interface{}) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("request marshal failed: %w", err)
	}

	spinner := api.StartThinkingSpinner(ctx, false, "")

	output, err := p.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(model),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        jsonBody,
	})
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("bedrock API error: %w", err)
	}

	return p.handleEventStream(ctx, output, spinner)
}

// LastAnthropicThinkingBlocks は最後の API 呼び出しで返された thinking blocks を返す。
func (p *Provider) LastAnthropicThinkingBlocks() []api.AnthropicThinkingBlock {
	if p == nil {
		return nil
	}
	return api.AnthropicThinkingBlocksFromContentBlocks(p.lastContentBlocks)
}

// LastAnthropicContentBlocks は最後の API 呼び出しで返された assistant content blocks を順序付きで返す。
func (p *Provider) LastAnthropicContentBlocks() []api.AnthropicContentBlock {
	if p == nil || len(p.lastContentBlocks) == 0 {
		return nil
	}
	return api.CloneAnthropicContentBlocks(p.lastContentBlocks)
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
