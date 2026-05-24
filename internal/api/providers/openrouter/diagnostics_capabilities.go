package openrouter

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

func (r *DiagnosticReport) addCapabilities(cfg *config.Config) {
	capabilities := buildOpenRouterDiagnosticCapabilities(cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"OpenRouter model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildOpenRouterDiagnosticCapabilities(cfg *config.Config, report DiagnosticReport) DiagnosticCapabilities {
	return providerdiag.DiagnosticCapabilitiesFromSnapshot(openRouterDiagnosticCapabilitySnapshot(cfg, report))
}

func openRouterDiagnosticCapabilitySnapshot(cfg *config.Config, report DiagnosticReport) providerdiag.CapabilitySnapshot {
	catalogUse := resolveOpenRouterDiagnosticCatalogModelUse(report.Model, report.CatalogModel)
	policy := providerdiag.OpenRouterCatalogPolicy(cfg, report.Model, catalogUse.PolicyCatalogModel)
	catalogModel := policy.CatalogModel
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Model,
		CatalogModel:                   catalogModel,
		Route:                          report.Route,
		RouteReason:                    report.RouteReason,
		ResponsesAPI:                   false,
		ResponsesStreaming:             false,
		ResponsesStreamingAvailability: providerdiag.KnownCapabilityAvailability(false),
		ChatCompletions:                report.Route == DiagnosticRouteChatCompletions,
		FunctionCalling:                report.FunctionCallingEnabled,
		ImageInput:                     providerdiag.ModelGatedImageInputAvailability("openrouter", report.Model, catalogModel, report.ImageInputSupported),
		WebSearch:                      providerdiag.KnownCapabilityAvailability(false),
		Thinking:                       providerdiag.KnownCapabilityAvailability(false),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention:                      providerdiag.NewRetentionSnapshot(false, false, false),
		ServerCompaction:               providerdiag.NewServerCompactionSnapshot(providerdiag.ServerCompactionSnapshotOptions{}),
		ContextWindowTokens:            policy.ContextWindowTokens,
		ContextWindowKnown:             policy.ContextWindowKnown,
		MaxOutput:                      policy.MaxOutput,
		Pricing:                        policy.Pricing,
	}
}

func (r *DiagnosticReport) addRequiredCapabilities(cfg *config.Config, required []string) {
	diagnostic := providerdiag.NewRequiredCapabilityDiagnostic(
		openRouterDiagnosticCapabilitySnapshot(cfg, *r),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "OpenRouter",
			MissingTarget:                 "OpenRouter model/configuration",
			UnknownAvailabilitySuggestion: "Use --catalog-model with a known OpenRouter model before requiring catalog-gated capabilities",
		},
	)
	if !diagnostic.Requested {
		return
	}
	status := DiagnosticStatusFail
	if diagnostic.Satisfied {
		status = DiagnosticStatusOK
	}
	r.addCheck(status, diagnostic.Name, diagnostic.Message, diagnostic.Detail, diagnostic.Suggestion)
}
