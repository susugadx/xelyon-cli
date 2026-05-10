package azure

import (
	"context"
	"fmt"
	"strings"

	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

const diagnosticCapabilityPreviousResponseID = "capability_previous_response_id"

// DiagnosticCapabilities は Azure OpenAI doctor が解決した deployment 能力を表す。
type DiagnosticCapabilities struct {
	Deployment               string                               `json:"deployment"`
	CatalogModel             string                               `json:"catalog_model"`
	Route                    string                               `json:"route"`
	RouteReason              string                               `json:"route_reason,omitempty"`
	ResponsesAPI             bool                                 `json:"responses_api"`
	ResponsesStreaming       bool                                 `json:"responses_streaming"`
	FunctionCalling          bool                                 `json:"function_calling"`
	ImageInput               bool                                 `json:"image_input"`
	Retention                DiagnosticRetentionCapability        `json:"retention"`
	ServerCompaction         DiagnosticServerCompactionCapability `json:"server_compaction"`
	ContextWindowTokens      int                                  `json:"context_window_tokens,omitempty"`
	ContextWindowKnown       bool                                 `json:"context_window_known"`
	MaxOutputTokens          int                                  `json:"max_output_tokens,omitempty"`
	MaxOutputTokensKnown     bool                                 `json:"max_output_tokens_known"`
	MaxOutputTokensSource    string                               `json:"max_output_tokens_source,omitempty"`
	MaxOutputRuntimeFallback int                                  `json:"max_output_runtime_fallback,omitempty"`
	Pricing                  DiagnosticPricingCapability          `json:"pricing"`
}

// DiagnosticRetentionCapability は Responses API の retention 設定由来の能力を表す。
type DiagnosticRetentionCapability struct {
	Supported          bool   `json:"supported"`
	Store              bool   `json:"store"`
	PreviousResponseID bool   `json:"previous_response_id"`
	PersistResponseID  bool   `json:"persist_response_id"`
	SessionPersistence bool   `json:"session_persistence"`
	Detail             string `json:"detail,omitempty"`
}

// DiagnosticServerCompactionCapability は server-side context compaction の能力を表す。
type DiagnosticServerCompactionCapability struct {
	Enabled                  bool   `json:"enabled"`
	RequestPayload           bool   `json:"request_payload"`
	CompactThreshold         int    `json:"compact_threshold,omitempty"`
	LocalFallback            bool   `json:"local_fallback"`
	SkipLocalAutoCompression bool   `json:"skip_local_auto_compression"`
	Detail                   string `json:"detail,omitempty"`
}

// DiagnosticPricingCapability は catalog/config から解決した pricing 能力を表す。
type DiagnosticPricingCapability struct {
	Available           bool    `json:"available"`
	InputCostPerM       float64 `json:"input_cost_per_m,omitempty"`
	CachedInputCostPerM float64 `json:"cached_input_cost_per_m,omitempty"`
	OutputCostPerM      float64 `json:"output_cost_per_m,omitempty"`
	Detail              string  `json:"detail,omitempty"`
}

func (r *DiagnosticReport) addCapabilities(ctx context.Context, cfg *config.Config) {
	capabilities := buildDiagnosticCapabilities(ctx, cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"Azure OpenAI deployment capabilities were resolved",
		diagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildDiagnosticCapabilities(ctx context.Context, cfg *config.Config, report DiagnosticReport) DiagnosticCapabilities {
	policyCfg := diagnosticCatalogPolicyConfig(cfg, report.Deployment, report.CatalogModel)
	contextWindow, contextOK := llmcatalog.KnownModelContextLimit(report.CatalogModel)
	maxOutput := diagnosticMaxOutputPolicy(policyCfg, report.Deployment, report.CatalogModel)
	pricing := cost.GetPricingInfoForConfig(policyCfg, "azure", report.Deployment)
	return DiagnosticCapabilities{
		Deployment:               report.Deployment,
		CatalogModel:             report.CatalogModel,
		Route:                    report.Route,
		RouteReason:              report.RouteReason,
		ResponsesAPI:             report.Route != "",
		ResponsesStreaming:       report.Route == DiagnosticRouteResponsesStreaming,
		FunctionCalling:          report.FunctionCallingEnabled,
		ImageInput:               New("diagnostic-key").SupportsImages(),
		Retention:                diagnosticRetentionCapability(report, report.Route != ""),
		ServerCompaction:         diagnosticServerCompactionCapability(ctx, policyCfg, report, report.Route != ""),
		ContextWindowTokens:      contextWindow,
		ContextWindowKnown:       contextOK,
		MaxOutputTokens:          diagnosticCapabilityMaxOutputTokens(maxOutput),
		MaxOutputTokensKnown:     maxOutput.Available,
		MaxOutputTokensSource:    diagnosticCapabilityMaxOutputSource(maxOutput),
		MaxOutputRuntimeFallback: maxOutput.RuntimeFallback,
		Pricing:                  diagnosticPricingCapability(pricing),
	}
}

func diagnosticRetentionCapability(report DiagnosticReport, responsesAPI bool) DiagnosticRetentionCapability {
	previousResponseID := responsesAPI && report.ResponsesStore
	sessionPersistence := previousResponseID && report.ResponsesPersistID
	detail := fmt.Sprintf(
		"responses_api=%t, responses.store=%t, previous_response_id=%t, session_persistence=%t",
		responsesAPI,
		report.ResponsesStore,
		previousResponseID,
		sessionPersistence,
	)
	return DiagnosticRetentionCapability{
		Supported:          responsesAPI,
		Store:              report.ResponsesStore,
		PreviousResponseID: previousResponseID,
		PersistResponseID:  report.ResponsesPersistID,
		SessionPersistence: sessionPersistence,
		Detail:             detail,
	}
}

func diagnosticServerCompactionCapability(ctx context.Context, cfg *config.Config, report DiagnosticReport, responsesAPI bool) DiagnosticServerCompactionCapability {
	capability := DiagnosticServerCompactionCapability{
		Enabled:       responsesAPI && cfg.ResponsesServerCompactionEnabled(),
		LocalFallback: cfg.ResponsesServerCompactionLocalFallbackEnabled(),
	}
	if !responsesAPI {
		capability.Detail = "route could not be resolved"
		return capability
	}
	if !cfg.ResponsesServerCompactionEnabled() {
		capability.Detail = "responses.server_compaction is disabled or responses.store=false"
		return capability
	}
	if ctx == nil {
		ctx = context.Background()
	}
	decision := openairesponses.ResolveServerCompactionDecision(
		config.WithContext(ctx, cfg),
		"azure",
		openairesponses.NewModelIdentity(report.Deployment, report.CatalogModel),
		diagnosticCapabilityPreviousResponseID,
	)
	capability.CompactThreshold = decision.CompactThreshold()
	capability.RequestPayload = capability.CompactThreshold > 0
	capability.SkipLocalAutoCompression = decision.ShouldSkipLocalAutoCompression
	switch {
	case capability.RequestPayload:
		capability.Detail = "context_management.compaction would be sent with previous_response_id"
	case capability.SkipLocalAutoCompression:
		capability.Detail = "compact_threshold could not be resolved and local fallback is disabled"
	default:
		capability.Detail = "compact_threshold could not be resolved; local fallback remains enabled"
	}
	return capability
}

func diagnosticCapabilityMaxOutputTokens(maxOutput diagnosticMaxOutputPolicyResult) int {
	if maxOutput.Available {
		return maxOutput.Tokens
	}
	return maxOutput.RuntimeFallback
}

func diagnosticCapabilityMaxOutputSource(maxOutput diagnosticMaxOutputPolicyResult) string {
	if maxOutput.Available {
		return maxOutput.Source
	}
	if maxOutput.RuntimeFallback > 0 {
		return "runtime_fallback"
	}
	return maxOutput.Source
}

func diagnosticPricingCapability(pricing cost.PricingInfo) DiagnosticPricingCapability {
	return DiagnosticPricingCapability{
		Available:           !pricing.PricingUnavailable,
		InputCostPerM:       pricing.InputCostPerM,
		CachedInputCostPerM: pricing.CachedInputCostPerM,
		OutputCostPerM:      pricing.OutputCostPerM,
		Detail:              diagnosticPricingDetail(pricing),
	}
}

func diagnosticCapabilitiesDetail(capabilities DiagnosticCapabilities) string {
	features := []string{
		fmt.Sprintf("responses_api=%t", capabilities.ResponsesAPI),
		fmt.Sprintf("responses_streaming=%t", capabilities.ResponsesStreaming),
		fmt.Sprintf("function_calling=%t", capabilities.FunctionCalling),
		fmt.Sprintf("image_input=%t", capabilities.ImageInput),
		fmt.Sprintf("previous_response_id=%t", capabilities.Retention.PreviousResponseID),
		fmt.Sprintf("server_compaction=%t", capabilities.ServerCompaction.RequestPayload),
	}
	if capabilities.ContextWindowKnown {
		features = append(features, fmt.Sprintf("context_window=%d", capabilities.ContextWindowTokens))
	} else {
		features = append(features, "context_window=unknown")
	}
	if capabilities.MaxOutputTokensKnown {
		features = append(features, fmt.Sprintf("max_output_tokens=%d (%s)", capabilities.MaxOutputTokens, capabilities.MaxOutputTokensSource))
	} else if capabilities.MaxOutputRuntimeFallback > 0 {
		features = append(features, fmt.Sprintf("max_output_tokens=missing (runtime_fallback=%d)", capabilities.MaxOutputRuntimeFallback))
	} else {
		features = append(features, "max_output_tokens=unknown")
	}
	features = append(features, capabilities.Pricing.Detail)
	return strings.Join(features, ", ")
}
