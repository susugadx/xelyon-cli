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
			Source:    "model_overrides",
			Available: true,
		}
	}

	if tokens, ok := llmcatalog.KnownMaxOutputTokens(options.CatalogModel); ok {
		return MaxOutputPolicy{
			Tokens:    tokens,
			Source:    "catalog",
			Available: true,
		}
	}

	tokens := api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), options.Provider, options.RequestModel)
	if options.ProviderDefaultKnown && tokens > 0 {
		return MaxOutputPolicy{
			Tokens:    tokens,
			Source:    "provider_default",
			Available: true,
		}
	}

	result := MaxOutputPolicy{Source: "missing"}
	if options.IncludeRuntimeFallback {
		result.RuntimeFallback = tokens
	}
	return result
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
		return "runtime_fallback"
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
	return "missing"
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
	return fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, %s",
		p.CatalogModel,
		IntDetail(p.ContextWindowTokens, p.ContextWindowKnown),
		p.MaxOutput.PlainDetail("unknown"),
		PricingDetail(p.Pricing),
	)
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
	RequestModel        string
	CatalogModel        string
	Route               string
	RouteReason         string
	ResponsesAPI        bool
	ResponsesStreaming  bool
	ChatCompletions     bool
	FunctionCalling     bool
	ImageInput          bool
	Retention           RetentionSnapshot
	ServerCompaction    ServerCompactionSnapshot
	ContextWindowTokens int
	ContextWindowKnown  bool
	MaxOutput           MaxOutputPolicy
	Pricing             cost.PricingInfo
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
