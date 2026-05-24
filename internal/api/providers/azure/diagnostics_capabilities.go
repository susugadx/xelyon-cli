package azure

import (
	"context"

	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const diagnosticCapabilityPreviousResponseID = "capability_previous_response_id"

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

type DiagnosticRetentionCapability = providerdiag.DiagnosticRetentionCapability

type DiagnosticServerCompactionCapability = providerdiag.DiagnosticServerCompactionCapability

type DiagnosticPricingCapability = providerdiag.DiagnosticPricingCapability

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
	snapshot := diagnosticCapabilitySnapshot(ctx, policyCfg, report)
	return diagnosticCapabilitiesFromSnapshot(snapshot)
}

func diagnosticCapabilitySnapshot(ctx context.Context, cfg *config.Config, report DiagnosticReport) providerdiag.CapabilitySnapshot {
	policy := providerdiag.AzureCatalogPolicy(cfg, report.Deployment, report.CatalogModel)
	return buildDiagnosticCapabilitySnapshot(ctx, cfg, report, policy)
}

func buildDiagnosticCapabilitySnapshot(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	policy providerdiag.CatalogPolicy,
) providerdiag.CapabilitySnapshot {
	responsesAPI := report.Route != ""
	responsesStreaming := report.Route == DiagnosticRouteResponsesStreaming
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Deployment,
		CatalogModel:                   report.CatalogModel,
		Route:                          report.Route,
		RouteReason:                    report.RouteReason,
		ResponsesAPI:                   responsesAPI,
		ResponsesStreaming:             responsesStreaming,
		ResponsesStreamingAvailability: providerdiag.ResponsesStreamingCapabilityAvailability(responsesStreaming, policy),
		FunctionCalling:                report.FunctionCallingEnabled,
		ImageInput:                     providerdiag.ModelGatedImageInputAvailability("azure", report.Deployment, report.CatalogModel, New("diagnostic-key").SupportsImages()),
		WebSearch:                      providerdiag.KnownCapabilityAvailability(false),
		Thinking:                       diagnosticThinkingAvailability(ctx, cfg, report, responsesAPI),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention:                      providerdiag.NewRetentionSnapshot(responsesAPI, report.ResponsesStore, report.ResponsesPersistID),
		ServerCompaction:               diagnosticServerCompactionSnapshot(ctx, cfg, report, responsesAPI),
		ContextWindowTokens:            policy.ContextWindowTokens,
		ContextWindowKnown:             policy.ContextWindowKnown,
		MaxOutput:                      policy.MaxOutput,
		Pricing:                        policy.Pricing,
	}
}

func diagnosticThinkingAvailability(ctx context.Context, cfg *config.Config, report DiagnosticReport, responsesAPI bool) providerdiag.CapabilityAvailability {
	if !responsesAPI {
		return providerdiag.KnownCapabilityAvailability(false)
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx := config.WithContext(ctx, cfg)
	reasoning := azureResponsesReasoningConfig(requestCtx, newModelIdentity(report.Deployment, report.CatalogModel))
	return providerdiag.KnownCapabilityAvailability(reasoning != nil)
}

func diagnosticCapabilitiesFromSnapshot(snapshot providerdiag.CapabilitySnapshot) DiagnosticCapabilities {
	capabilities := providerdiag.DiagnosticCapabilitiesFromSnapshot(snapshot)
	capabilities.Deployment = capabilities.Model
	capabilities.Model = ""
	return capabilities
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

func diagnosticCapabilitiesDetail(capabilities DiagnosticCapabilities) string {
	return providerdiag.DiagnosticCapabilitiesDetail(capabilities)
}
