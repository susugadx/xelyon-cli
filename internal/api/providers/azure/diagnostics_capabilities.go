package azure

import (
	"context"
	"fmt"
	"strings"

	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
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
	policy := providerdiag.AzureCatalogPolicy(policyCfg, report.Deployment, report.CatalogModel)
	snapshot := buildDiagnosticCapabilitySnapshot(ctx, policyCfg, report, policy)
	return diagnosticCapabilitiesFromSnapshot(snapshot)
}

func buildDiagnosticCapabilitySnapshot(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	policy providerdiag.CatalogPolicy,
) providerdiag.CapabilitySnapshot {
	responsesAPI := report.Route != ""
	return providerdiag.CapabilitySnapshot{
		RequestModel:        report.Deployment,
		CatalogModel:        report.CatalogModel,
		Route:               report.Route,
		RouteReason:         report.RouteReason,
		ResponsesAPI:        responsesAPI,
		ResponsesStreaming:  report.Route == DiagnosticRouteResponsesStreaming,
		FunctionCalling:     report.FunctionCallingEnabled,
		ImageInput:          New("diagnostic-key").SupportsImages(),
		Retention:           providerdiag.NewRetentionSnapshot(responsesAPI, report.ResponsesStore, report.ResponsesPersistID),
		ServerCompaction:    diagnosticServerCompactionSnapshot(ctx, cfg, report, responsesAPI),
		ContextWindowTokens: policy.ContextWindowTokens,
		ContextWindowKnown:  policy.ContextWindowKnown,
		MaxOutput:           policy.MaxOutput,
		Pricing:             policy.Pricing,
	}
}

func diagnosticCapabilitiesFromSnapshot(snapshot providerdiag.CapabilitySnapshot) DiagnosticCapabilities {
	return DiagnosticCapabilities{
		Deployment:               snapshot.RequestModel,
		CatalogModel:             snapshot.CatalogModel,
		Route:                    snapshot.Route,
		RouteReason:              snapshot.RouteReason,
		ResponsesAPI:             snapshot.ResponsesAPI,
		ResponsesStreaming:       snapshot.ResponsesStreaming,
		FunctionCalling:          snapshot.FunctionCalling,
		ImageInput:               snapshot.ImageInput,
		Retention:                diagnosticRetentionCapabilityFromSnapshot(snapshot.Retention),
		ServerCompaction:         diagnosticServerCompactionCapabilityFromSnapshot(snapshot.ServerCompaction),
		ContextWindowTokens:      snapshot.ContextWindowTokens,
		ContextWindowKnown:       snapshot.ContextWindowKnown,
		MaxOutputTokens:          snapshot.MaxOutput.CapabilityTokens(),
		MaxOutputTokensKnown:     snapshot.MaxOutput.Available,
		MaxOutputTokensSource:    snapshot.MaxOutput.CapabilitySource(),
		MaxOutputRuntimeFallback: snapshot.MaxOutput.RuntimeFallback,
		Pricing:                  diagnosticPricingCapability(snapshot.Pricing),
	}
}

func diagnosticRetentionCapabilityFromSnapshot(snapshot providerdiag.RetentionSnapshot) DiagnosticRetentionCapability {
	return DiagnosticRetentionCapability{
		Supported:          snapshot.Supported,
		Store:              snapshot.Store,
		PreviousResponseID: snapshot.PreviousResponseID,
		PersistResponseID:  snapshot.PersistResponseID,
		SessionPersistence: snapshot.SessionPersistence,
		Detail:             snapshot.Detail,
	}
}

func diagnosticServerCompactionSnapshot(ctx context.Context, cfg *config.Config, report DiagnosticReport, responsesAPI bool) providerdiag.ServerCompactionSnapshot {
	if ctx == nil {
		ctx = context.Background()
	}
	var compactThreshold int
	var skipLocalAutoCompression bool
	if responsesAPI && cfg.ResponsesServerCompactionEnabled() {
		decision := openairesponses.ResolveServerCompactionDecision(
			config.WithContext(ctx, cfg),
			"azure",
			openairesponses.NewModelIdentity(report.Deployment, report.CatalogModel),
			diagnosticCapabilityPreviousResponseID,
		)
		compactThreshold = decision.CompactThreshold()
		skipLocalAutoCompression = decision.ShouldSkipLocalAutoCompression
	}
	return providerdiag.NewServerCompactionSnapshot(providerdiag.ServerCompactionSnapshotOptions{
		ResponsesAPI:             responsesAPI,
		Enabled:                  cfg.ResponsesServerCompactionEnabled(),
		LocalFallback:            cfg.ResponsesServerCompactionLocalFallbackEnabled(),
		CompactThreshold:         compactThreshold,
		SkipLocalAutoCompression: skipLocalAutoCompression,
		UnavailableDetail:        "route could not be resolved",
	})
}

func diagnosticServerCompactionCapabilityFromSnapshot(snapshot providerdiag.ServerCompactionSnapshot) DiagnosticServerCompactionCapability {
	return DiagnosticServerCompactionCapability{
		Enabled:                  snapshot.Enabled,
		RequestPayload:           snapshot.RequestPayload,
		CompactThreshold:         snapshot.CompactThreshold,
		LocalFallback:            snapshot.LocalFallback,
		SkipLocalAutoCompression: snapshot.SkipLocalAutoCompression,
		Detail:                   snapshot.Detail,
	}
}

func diagnosticPricingCapability(pricing cost.PricingInfo) DiagnosticPricingCapability {
	return DiagnosticPricingCapability{
		Available:           !pricing.PricingUnavailable,
		InputCostPerM:       pricing.InputCostPerM,
		CachedInputCostPerM: pricing.CachedInputCostPerM,
		OutputCostPerM:      pricing.OutputCostPerM,
		Detail:              providerdiag.PricingDetail(pricing),
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
