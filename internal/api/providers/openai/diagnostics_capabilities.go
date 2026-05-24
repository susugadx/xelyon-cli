package openai

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
	capabilities := buildOpenAIDiagnosticCapabilities(ctx, cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"OpenAI model capabilities were resolved",
		openAIDiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildOpenAIDiagnosticCapabilities(ctx context.Context, cfg *config.Config, report DiagnosticReport) DiagnosticCapabilities {
	snapshot := openAIDiagnosticCapabilitySnapshot(ctx, cfg, report)
	return openAIDiagnosticCapabilitiesFromSnapshot(snapshot)
}

func openAIDiagnosticCapabilitySnapshot(ctx context.Context, cfg *config.Config, report DiagnosticReport) providerdiag.CapabilitySnapshot {
	responsesAPI := report.Route != "" && report.Route != DiagnosticRouteChatCompletions
	policy := providerdiag.OpenAICatalogPolicy(cfg, report.Model, report.CatalogModel)
	return buildOpenAIDiagnosticCapabilitySnapshot(ctx, cfg, report, responsesAPI, policy)
}

func buildOpenAIDiagnosticCapabilitySnapshot(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	responsesAPI bool,
	policy providerdiag.CatalogPolicy,
) providerdiag.CapabilitySnapshot {
	responsesStreaming := report.Route == DiagnosticRouteResponsesStreaming
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Model,
		CatalogModel:                   report.CatalogModel,
		Route:                          report.Route,
		RouteReason:                    report.RouteReason,
		ResponsesAPI:                   responsesAPI,
		ResponsesStreaming:             responsesStreaming,
		ResponsesStreamingAvailability: providerdiag.ResponsesStreamingCapabilityAvailability(responsesStreaming, policy),
		ChatCompletions:                report.Route == DiagnosticRouteChatCompletions,
		FunctionCalling:                report.FunctionCallingEnabled,
		ImageInput:                     providerdiag.ModelGatedImageInputAvailability("openai", report.Model, report.CatalogModel, New("diagnostic-key").SupportsImages()),
		WebSearch:                      openAIDiagnosticWebSearchAvailability(cfg, report.Model),
		Thinking:                       openAIDiagnosticThinkingAvailability(ctx, cfg, report, responsesAPI),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention:                      providerdiag.NewRetentionSnapshot(responsesAPI, report.ResponsesStore, report.ResponsesPersistResponseID),
		ServerCompaction:               openAIDiagnosticServerCompactionSnapshot(ctx, cfg, report, responsesAPI),
		ContextWindowTokens:            policy.ContextWindowTokens,
		ContextWindowKnown:             policy.ContextWindowKnown,
		MaxOutput:                      policy.MaxOutput,
		Pricing:                        policy.Pricing,
	}
}

func openAIDiagnosticWebSearchAvailability(cfg *config.Config, model string) providerdiag.CapabilityAvailability {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return providerdiag.KnownCapabilityAvailability(cfg.IsProviderResponsesAPIModel("openai", model))
}

func openAIDiagnosticThinkingAvailability(ctx context.Context, cfg *config.Config, report DiagnosticReport, responsesAPI bool) providerdiag.CapabilityAvailability {
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
	reasoning := responsesReasoningConfig(requestCtx, openairesponses.NewModelIdentity(report.Model, report.CatalogModel))
	return providerdiag.KnownCapabilityAvailability(reasoning != nil)
}

func openAIDiagnosticCapabilitiesFromSnapshot(snapshot providerdiag.CapabilitySnapshot) DiagnosticCapabilities {
	capabilities := providerdiag.DiagnosticCapabilitiesFromSnapshot(snapshot)
	capabilities.MaxOutputTokens = snapshot.MaxOutput.Tokens
	capabilities.MaxOutputTokensSource = snapshot.MaxOutput.Source
	capabilities.MaxOutputRuntimeFallback = 0
	return capabilities
}

func openAIDiagnosticServerCompactionSnapshot(ctx context.Context, cfg *config.Config, report DiagnosticReport, responsesAPI bool) providerdiag.ServerCompactionSnapshot {
	if ctx == nil {
		ctx = context.Background()
	}
	var compactThreshold int
	var skipLocalAutoCompression bool
	if responsesAPI && cfg.ResponsesServerCompactionEnabled() {
		decision := openairesponses.ResolveServerCompactionDecision(
			config.WithContext(ctx, cfg),
			"openai",
			openairesponses.NewModelIdentity(report.Model, report.CatalogModel),
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
		UnavailableDetail:        "not available on the Chat Completions route",
	})
}

func openAIDiagnosticCapabilitiesDetail(capabilities DiagnosticCapabilities) string {
	return providerdiag.DiagnosticCapabilitiesDetail(capabilities)
}
