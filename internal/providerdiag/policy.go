package providerdiag

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// RouteDecision は doctor が選んだ route と、その理由を構造化して保持する。
type RouteDecision struct {
	Route   string
	Reasons []string
}

// ReasonString は既存の route_reason 文字列へ戻す。
func (d RouteDecision) ReasonString() string {
	reasons := make([]string, 0, len(d.Reasons))
	for _, reason := range d.Reasons {
		reason = strings.TrimSpace(reason)
		if reason != "" {
			reasons = append(reasons, reason)
		}
	}
	return strings.Join(reasons, "; ")
}

// ShouldStreamResponsesCatalogModel は OpenAI-family Responses API で
// streaming を使うべき catalog model か返す。
func ShouldStreamResponsesCatalogModel(model string) bool {
	return !isGPT55ProCatalogModel(model)
}

func isGPT55ProCatalogModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return model == "gpt-5.5-pro" || strings.HasPrefix(model, "gpt-5.5-pro-")
}

// ResponsesStreamingReason は既存 route_reason 用の streaming 理由を返す。
func ResponsesStreamingReason(catalogModel string, streaming bool) string {
	catalogModel = strings.TrimSpace(catalogModel)
	if catalogModel == "" {
		return "catalog_model is not resolved; Responses streaming defaults to enabled"
	}
	if streaming {
		return fmt.Sprintf("catalog_model=%s supports Responses streaming", catalogModel)
	}
	return fmt.Sprintf("catalog_model=%s disables Responses streaming", catalogModel)
}

// MaxOutputPolicy は doctor catalog policy 用に解決した max_output_tokens を表す。
type MaxOutputPolicy struct {
	Tokens          int
	Source          string
	Available       bool
	RuntimeFallback int
}

const (
	// MaxOutputSourceCatalog は catalog metadata 由来の max_output_tokens を表す。
	MaxOutputSourceCatalog = "catalog"
	// MaxOutputSourceModelOverrides は provider_models.<provider>.model_overrides 由来の max_output_tokens を表す。
	MaxOutputSourceModelOverrides = "model_overrides"
	// MaxOutputSourceProviderDefault は provider default 由来の max_output_tokens を表す。
	MaxOutputSourceProviderDefault = "provider_default"
	// MaxOutputSourceRuntimeFallback は metadata 不足時に runtime が使う fallback 値を表す。
	MaxOutputSourceRuntimeFallback = "runtime_fallback"
	// MaxOutputSourceMissing は max_output_tokens を確定できない状態を表す。
	MaxOutputSourceMissing = "missing"
)

// OpenAIMaxOutputPolicy は OpenAI doctor の既存 max output 解決規則を返す。
func OpenAIMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "openai",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

// AzureMaxOutputPolicy は Azure doctor の既存 max output 解決規則を返す。
func AzureMaxOutputPolicy(cfg *config.Config, deployment, catalogModel string) MaxOutputPolicy {
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:               "azure",
		RequestModel:           deployment,
		CatalogModel:           catalogModel,
		IncludeRuntimeFallback: true,
	})
}

// GroqMaxOutputPolicy は Groq doctor の既存 max output 解決規則を返す。
func GroqMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	if nonProviderCatalogModel("groq", catalogModel) {
		return maxOutputOverridePolicy(cfg, "groq", model)
	}
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "groq",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

// DeepSeekMaxOutputPolicy は DeepSeek doctor の max output 解決規則を返す。
func DeepSeekMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	if nonProviderCatalogModel("deepseek", catalogModel) {
		return maxOutputOverridePolicy(cfg, "deepseek", model)
	}
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "deepseek",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

// OpenRouterMaxOutputPolicy は OpenRouter doctor の max output 解決規則を返す。
func OpenRouterMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	if strings.TrimSpace(catalogModel) == "" || nonProviderCatalogModel("openrouter", catalogModel) {
		return maxOutputOverridePolicy(cfg, "openrouter", model)
	}
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "openrouter",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

// OllamaMaxOutputPolicy は Ollama doctor の max output 解決規則を返す。
func OllamaMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	if nonProviderCatalogModel("ollama", catalogModel) {
		catalogModel = ""
	}
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "ollama",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

// BedrockUntrustedCatalogMaxOutputPolicy は non-Bedrock catalog を使わず request model 側の Bedrock max output を返す。
func BedrockUntrustedCatalogMaxOutputPolicy(cfg *config.Config, model string) MaxOutputPolicy {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if override, ok := cfg.ModelOverrideForProvider("bedrock", model); ok && override.MaxOutputTokens > 0 {
		return MaxOutputPolicy{
			Tokens:    override.MaxOutputTokens,
			Source:    MaxOutputSourceModelOverrides,
			Available: true,
		}
	}
	if providerCfg, ok := cfg.GetProviderModelConfig("bedrock"); ok &&
		strings.TrimSpace(providerCfg.DefaultModel) == strings.TrimSpace(model) &&
		providerCfg.MaxOutputTokens > 0 {
		return MaxOutputPolicy{
			Tokens:    providerCfg.MaxOutputTokens,
			Source:    MaxOutputSourceProviderDefault,
			Available: true,
		}
	}
	if llmcatalog.IsKnownModelNameForProvider("bedrock", model) {
		if tokens, ok := llmcatalog.KnownMaxOutputTokens(model); ok {
			return MaxOutputPolicy{
				Tokens:    tokens,
				Source:    MaxOutputSourceCatalog,
				Available: true,
			}
		}
	}
	return MaxOutputPolicy{Source: MaxOutputSourceMissing}
}

// BedrockClaudeMaxOutputPolicy は Bedrock Claude Messages route の max output 解決規則を返す。
func BedrockClaudeMaxOutputPolicy(cfg *config.Config, model string) MaxOutputPolicy {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if override, ok := cfg.ModelOverrideForProvider("bedrock", model); ok && override.MaxOutputTokens > 0 {
		return MaxOutputPolicy{
			Tokens:    override.MaxOutputTokens,
			Source:    MaxOutputSourceModelOverrides,
			Available: true,
		}
	}
	if catalogModel := cfg.ModelCatalogName("bedrock", model); llmcatalog.IsKnownModelNameForProvider("bedrock", catalogModel) {
		if tokens, ok := llmcatalog.KnownMaxOutputTokens(catalogModel); ok {
			return MaxOutputPolicy{
				Tokens:    tokens,
				Source:    MaxOutputSourceCatalog,
				Available: true,
			}
		}
	}
	if llmcatalog.IsKnownModelNameForProvider("bedrock", model) {
		if tokens, ok := llmcatalog.KnownMaxOutputTokens(model); ok {
			return MaxOutputPolicy{
				Tokens:    tokens,
				Source:    MaxOutputSourceCatalog,
				Available: true,
			}
		}
	}
	if providerCfg, ok := cfg.GetProviderModelConfig("bedrock"); ok && providerCfg.MaxOutputTokens > 0 {
		return MaxOutputPolicy{
			Tokens:    providerCfg.MaxOutputTokens,
			Source:    MaxOutputSourceProviderDefault,
			Available: true,
		}
	}
	return MaxOutputPolicy{Source: MaxOutputSourceMissing}
}

// GeminiMaxOutputPolicy は Gemini doctor の max output 解決規則を返す。
func GeminiMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	if nonProviderCatalogModel("gemini", catalogModel) {
		return maxOutputOverridePolicy(cfg, "gemini", model)
	}
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "gemini",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

// KimiMaxOutputPolicy は Kimi doctor の max output 解決規則を返す。
func KimiMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	if nonProviderCatalogModel("kimi", catalogModel) {
		return maxOutputOverridePolicy(cfg, "kimi", model)
	}
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "kimi",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

// ClaudeMaxOutputPolicy は Claude doctor の max output 解決規則を返す。
func ClaudeMaxOutputPolicy(cfg *config.Config, model, catalogModel string) MaxOutputPolicy {
	if nonProviderCatalogModel("claude", catalogModel) {
		return maxOutputOverridePolicy(cfg, "claude", model)
	}
	return resolveMaxOutputPolicy(cfg, maxOutputPolicyOptions{
		Provider:             "claude",
		RequestModel:         model,
		CatalogModel:         catalogModel,
		ProviderDefaultKnown: true,
	})
}

func nonProviderCatalogModel(provider, catalogModel string) bool {
	catalogModel = strings.TrimSpace(catalogModel)
	return catalogModel != "" && !IsProviderCatalogModelKnown(provider, catalogModel)
}

type maxOutputPolicyOptions struct {
	Provider               string
	RequestModel           string
	CatalogModel           string
	ProviderDefaultKnown   bool
	IncludeRuntimeFallback bool
}

func resolveMaxOutputPolicy(cfg *config.Config, options maxOutputPolicyOptions) MaxOutputPolicy {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if override, ok := cfg.ModelOverrideForProvider(options.Provider, options.RequestModel); ok && override.MaxOutputTokens > 0 {
		return MaxOutputPolicy{
			Tokens:    override.MaxOutputTokens,
			Source:    MaxOutputSourceModelOverrides,
			Available: true,
		}
	}

	if tokens, ok := llmcatalog.KnownMaxOutputTokens(options.CatalogModel); ok {
		return MaxOutputPolicy{
			Tokens:    tokens,
			Source:    MaxOutputSourceCatalog,
			Available: true,
		}
	}

	tokens := api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), options.Provider, options.RequestModel)
	if options.ProviderDefaultKnown && tokens > 0 {
		return MaxOutputPolicy{
			Tokens:    tokens,
			Source:    MaxOutputSourceProviderDefault,
			Available: true,
		}
	}

	result := MaxOutputPolicy{Source: MaxOutputSourceMissing}
	if options.IncludeRuntimeFallback {
		result.RuntimeFallback = tokens
	}
	return result
}

func maxOutputOverridePolicy(cfg *config.Config, provider, model string) MaxOutputPolicy {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if override, ok := cfg.ModelOverrideForProvider(provider, model); ok && override.MaxOutputTokens > 0 {
		return MaxOutputPolicy{
			Tokens:    override.MaxOutputTokens,
			Source:    MaxOutputSourceModelOverrides,
			Available: true,
		}
	}
	return MaxOutputPolicy{Source: MaxOutputSourceMissing}
}

// RuntimeMaxOutputTokens は doctor report の runtime request path と同じ max output 値を返す。
func RuntimeMaxOutputTokens(cfg *config.Config, provider, model string) int {
	return api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), provider, model)
}

// CapabilityTokens は provider-specific public capability DTO 用の値を返す。
func (p MaxOutputPolicy) CapabilityTokens() int {
	if p.Available {
		return p.Tokens
	}
	return p.RuntimeFallback
}

// CapabilitySource は provider-specific public capability DTO 用の source を返す。
func (p MaxOutputPolicy) CapabilitySource() string {
	if p.Available {
		return p.Source
	}
	if p.RuntimeFallback > 0 {
		return MaxOutputSourceRuntimeFallback
	}
	return p.Source
}

// PlainDetail は OpenAI catalog_policy が従来使っていた max output 表現を返す。
func (p MaxOutputPolicy) PlainDetail(missing string) string {
	if p.Available {
		return fmt.Sprintf("%d", p.Tokens)
	}
	if strings.TrimSpace(missing) == "" {
		return "unknown"
	}
	return missing
}

// SourceDetail は Azure catalog_policy が従来使っていた max output 表現を返す。
func (p MaxOutputPolicy) SourceDetail() string {
	if p.Available {
		return fmt.Sprintf("%d (%s)", p.Tokens, p.Source)
	}
	if p.RuntimeFallback > 0 {
		return fmt.Sprintf("missing (runtime_fallback=%d)", p.RuntimeFallback)
	}
	return MaxOutputSourceMissing
}

// CatalogPolicy は catalog由来の context / output / pricing / streaming policy を表す。
type CatalogPolicy struct {
	CatalogModel        string
	ContextWindowTokens int
	ContextWindowKnown  bool
	MaxOutput           MaxOutputPolicy
	Pricing             cost.PricingInfo
	ResponsesStreaming  bool
}

// OpenAICatalogPolicy は OpenAI doctor 用の catalog policy snapshot を返す。
func OpenAICatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	return NewCatalogPolicy(
		catalogModel,
		OpenAIMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "openai", model),
		ShouldStreamResponsesCatalogModel(catalogModel),
	)
}

// AzureCatalogPolicy は Azure doctor 用の catalog policy snapshot を返す。
func AzureCatalogPolicy(cfg *config.Config, deployment, catalogModel string) CatalogPolicy {
	return NewCatalogPolicy(
		catalogModel,
		AzureMaxOutputPolicy(cfg, deployment, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "azure", deployment),
		ShouldStreamResponsesCatalogModel(catalogModel),
	)
}

// GroqCatalogPolicy は Groq doctor 用の catalog policy snapshot を返す。
func GroqCatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	if nonProviderCatalogModel("groq", catalogModel) {
		return unknownProviderCatalogPolicy(catalogModel, GroqMaxOutputPolicy(cfg, model, catalogModel))
	}
	return NewCatalogPolicy(
		catalogModel,
		GroqMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "groq", model),
		false,
	)
}

// DeepSeekCatalogPolicy は DeepSeek doctor 用の catalog policy snapshot を返す。
func DeepSeekCatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	if nonProviderCatalogModel("deepseek", catalogModel) {
		return unknownProviderCatalogPolicy(catalogModel, DeepSeekMaxOutputPolicy(cfg, model, catalogModel))
	}
	return NewCatalogPolicy(
		catalogModel,
		DeepSeekMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "deepseek", model),
		false,
	)
}

// OpenRouterCatalogPolicy は OpenRouter doctor 用の catalog policy snapshot を返す。
func OpenRouterCatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	if strings.TrimSpace(catalogModel) == "" || nonProviderCatalogModel("openrouter", catalogModel) {
		return unknownProviderCatalogPolicy(catalogModel, OpenRouterMaxOutputPolicy(cfg, model, catalogModel))
	}
	return NewCatalogPolicy(
		catalogModel,
		OpenRouterMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "openrouter", model),
		false,
	)
}

// OllamaCatalogPolicy は Ollama doctor 用の catalog policy snapshot を返す。
func OllamaCatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	metadataCatalogModel := catalogModel
	if nonProviderCatalogModel("ollama", catalogModel) {
		metadataCatalogModel = ""
	}
	policy := NewCatalogPolicy(
		metadataCatalogModel,
		OllamaMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "ollama", model),
		false,
	)
	policy.CatalogModel = strings.TrimSpace(catalogModel)
	return policy
}

// BedrockCatalogPolicy は Bedrock doctor 用の catalog policy snapshot を返す。
func BedrockCatalogPolicy(cfg *config.Config, model, catalogModel string, maxOutput MaxOutputPolicy) CatalogPolicy {
	if nonProviderCatalogModel("bedrock", catalogModel) {
		return unknownProviderCatalogPolicy(catalogModel, maxOutput)
	}
	return NewCatalogPolicy(
		catalogModel,
		maxOutput,
		cost.GetPricingInfoForConfig(cfg, "bedrock", model),
		false,
	)
}

// GeminiCatalogPolicy は Gemini doctor 用の catalog policy snapshot を返す。
func GeminiCatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	if nonProviderCatalogModel("gemini", catalogModel) {
		return unknownProviderCatalogPolicy(catalogModel, GeminiMaxOutputPolicy(cfg, model, catalogModel))
	}
	return NewCatalogPolicy(
		catalogModel,
		GeminiMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "gemini", model),
		false,
	)
}

// KimiCatalogPolicy は Kimi doctor 用の catalog policy snapshot を返す。
func KimiCatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	if nonProviderCatalogModel("kimi", catalogModel) {
		return unknownProviderCatalogPolicy(catalogModel, KimiMaxOutputPolicy(cfg, model, catalogModel))
	}
	return NewCatalogPolicy(
		catalogModel,
		KimiMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "kimi", model),
		false,
	)
}

// ClaudeCatalogPolicy は Claude doctor 用の catalog policy snapshot を返す。
func ClaudeCatalogPolicy(cfg *config.Config, model, catalogModel string) CatalogPolicy {
	if nonProviderCatalogModel("claude", catalogModel) {
		return unknownProviderCatalogPolicy(catalogModel, ClaudeMaxOutputPolicy(cfg, model, catalogModel))
	}
	return NewCatalogPolicy(
		catalogModel,
		ClaudeMaxOutputPolicy(cfg, model, catalogModel),
		cost.GetPricingInfoForConfig(cfg, "claude", model),
		false,
	)
}

func unknownProviderCatalogPolicy(catalogModel string, maxOutput MaxOutputPolicy) CatalogPolicy {
	return CatalogPolicy{
		CatalogModel: strings.TrimSpace(catalogModel),
		MaxOutput:    maxOutput,
		Pricing:      cost.PricingInfo{PricingUnavailable: true},
	}
}

// NewCatalogPolicy は providerごとの解決結果から共通 snapshot を作る。
func NewCatalogPolicy(catalogModel string, maxOutput MaxOutputPolicy, pricing cost.PricingInfo, responsesStreaming bool) CatalogPolicy {
	contextWindow, contextOK := llmcatalog.KnownModelContextLimit(catalogModel)
	return CatalogPolicy{
		CatalogModel:        strings.TrimSpace(catalogModel),
		ContextWindowTokens: contextWindow,
		ContextWindowKnown:  contextOK,
		MaxOutput:           maxOutput,
		Pricing:             pricing,
		ResponsesStreaming:  responsesStreaming,
	}
}

// OpenAIDetail は OpenAI doctor の既存 catalog_policy detail を返す。
func (p CatalogPolicy) OpenAIDetail() string {
	return p.plainDetail()
}

// AzureDetail は Azure doctor の既存 catalog_policy detail を返す。
func (p CatalogPolicy) AzureDetail() string {
	return fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, responses_streaming=%t, %s",
		p.CatalogModel,
		IntDetail(p.ContextWindowTokens, p.ContextWindowKnown),
		p.MaxOutput.SourceDetail(),
		p.ResponsesStreaming,
		PricingDetail(p.Pricing),
	)
}

// GroqDetail は Groq doctor の catalog_policy detail を返す。
func (p CatalogPolicy) GroqDetail() string {
	return p.plainDetail()
}

// DeepSeekDetail は DeepSeek doctor の catalog_policy detail を返す。
func (p CatalogPolicy) DeepSeekDetail() string {
	return p.plainDetail()
}

// OpenRouterDetail は OpenRouter doctor の catalog_policy detail を返す。
func (p CatalogPolicy) OpenRouterDetail() string {
	return p.plainDetail()
}

// OllamaDetail は Ollama doctor の catalog_policy detail を返す。
func (p CatalogPolicy) OllamaDetail() string {
	return p.plainDetail()
}

// GeminiDetail は Gemini doctor の catalog_policy detail を返す。
func (p CatalogPolicy) GeminiDetail() string {
	return p.plainDetail()
}

// KimiDetail は Kimi doctor の catalog_policy detail を返す。
func (p CatalogPolicy) KimiDetail() string {
	return p.plainDetail()
}

// ClaudeDetail は Claude doctor の catalog_policy detail を返す。
func (p CatalogPolicy) ClaudeDetail() string {
	return p.plainDetail()
}

func (p CatalogPolicy) plainDetail() string {
	return fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, %s",
		p.CatalogModel,
		IntDetail(p.ContextWindowTokens, p.ContextWindowKnown),
		p.MaxOutput.PlainDetail("unknown"),
		PricingDetail(p.Pricing),
	)
}

// IntDetail は known/unknown 付き整数 detail を返す。
func IntDetail(value int, ok bool) string {
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%d", value)
}

// PricingDetail は doctor 出力用の pricing detail を返す。
func PricingDetail(pricing cost.PricingInfo) string {
	if pricing.PricingUnavailable {
		return "pricing=unavailable"
	}
	return fmt.Sprintf(
		"pricing=input $%.2f/M cached $%.3f/M output $%.2f/M",
		pricing.InputCostPerM,
		pricing.CachedInputCostPerM,
		pricing.OutputCostPerM,
	)
}

// CapabilitySnapshot は provider-specific public capabilities DTO へ投影する内部 snapshot。
type CapabilitySnapshot struct {
	RequestModel                   string
	CatalogModel                   string
	Route                          string
	RouteReason                    string
	ResponsesAPI                   bool
	ResponsesStreaming             bool
	ResponsesStreamingAvailability CapabilityAvailability
	ChatCompletions                bool
	FunctionCalling                bool
	ImageInput                     CapabilityAvailability
	WebSearch                      CapabilityAvailability
	Thinking                       CapabilityAvailability
	LocalModelAvailable            CapabilityAvailability
	Retention                      RetentionSnapshot
	ServerCompaction               ServerCompactionSnapshot
	ContextWindowTokens            int
	ContextWindowKnown             bool
	MaxOutput                      MaxOutputPolicy
	Pricing                        cost.PricingInfo
}

// RetentionSnapshot は Responses retention の内部 snapshot。
type RetentionSnapshot struct {
	Supported          bool
	Store              bool
	PreviousResponseID bool
	PersistResponseID  bool
	SessionPersistence bool
	Detail             string
}

// NewRetentionSnapshot は既存 doctor capability の retention detail を構築する。
func NewRetentionSnapshot(responsesAPI, store, persistResponseID bool) RetentionSnapshot {
	previousResponseID := responsesAPI && store
	sessionPersistence := previousResponseID && persistResponseID
	detail := fmt.Sprintf(
		"responses_api=%t, responses.store=%t, previous_response_id=%t, session_persistence=%t",
		responsesAPI,
		store,
		previousResponseID,
		sessionPersistence,
	)
	return RetentionSnapshot{
		Supported:          responsesAPI,
		Store:              store,
		PreviousResponseID: previousResponseID,
		PersistResponseID:  persistResponseID,
		SessionPersistence: sessionPersistence,
		Detail:             detail,
	}
}

// ServerCompactionSnapshot は server-side context compaction の内部 snapshot。
type ServerCompactionSnapshot struct {
	Enabled                  bool
	RequestPayload           bool
	CompactThreshold         int
	LocalFallback            bool
	SkipLocalAutoCompression bool
	Detail                   string
}

// NewServerCompactionSnapshot は request path の decision から capability 表示用 snapshot を作る。
func NewServerCompactionSnapshot(options ServerCompactionSnapshotOptions) ServerCompactionSnapshot {
	snapshot := ServerCompactionSnapshot{
		Enabled:       options.ResponsesAPI && options.Enabled,
		LocalFallback: options.LocalFallback,
	}
	if !options.ResponsesAPI {
		snapshot.Detail = options.UnavailableDetail
		return snapshot
	}
	if !options.Enabled {
		snapshot.Detail = "responses.server_compaction is disabled or responses.store=false"
		return snapshot
	}

	snapshot.CompactThreshold = options.CompactThreshold
	snapshot.RequestPayload = snapshot.CompactThreshold > 0
	snapshot.SkipLocalAutoCompression = options.SkipLocalAutoCompression
	switch {
	case snapshot.RequestPayload:
		snapshot.Detail = "context_management.compaction would be sent with previous_response_id"
	case snapshot.SkipLocalAutoCompression:
		snapshot.Detail = "compact_threshold could not be resolved and local fallback is disabled"
	default:
		snapshot.Detail = "compact_threshold could not be resolved; local fallback remains enabled"
	}
	return snapshot
}

// ServerCompactionSnapshotOptions は server compaction snapshot の入力。
type ServerCompactionSnapshotOptions struct {
	ResponsesAPI             bool
	Enabled                  bool
	LocalFallback            bool
	CompactThreshold         int
	SkipLocalAutoCompression bool
	UnavailableDetail        string
}
