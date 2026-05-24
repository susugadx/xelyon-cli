package groq

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

func (r *DiagnosticReport) addCapabilities(cfg *config.Config) {
	capabilities := buildGroqDiagnosticCapabilities(cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"Groq model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildGroqDiagnosticCapabilities(cfg *config.Config, report DiagnosticReport) DiagnosticCapabilities {
	return providerdiag.DiagnosticCapabilitiesFromSnapshot(groqDiagnosticCapabilitySnapshot(cfg, report))
}

func groqDiagnosticCapabilitySnapshot(cfg *config.Config, report DiagnosticReport) providerdiag.CapabilitySnapshot {
	policy := providerdiag.GroqCatalogPolicy(cfg, report.Model, report.CatalogModel)
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Model,
		CatalogModel:                   report.CatalogModel,
		Route:                          report.Route,
		RouteReason:                    report.RouteReason,
		ResponsesAPI:                   false,
		ResponsesStreaming:             false,
		ResponsesStreamingAvailability: providerdiag.KnownCapabilityAvailability(false),
		ChatCompletions:                report.Route == DiagnosticRouteChatCompletions,
		FunctionCalling:                report.FunctionCallingEnabled,
		ImageInput:                     providerdiag.KnownCapabilityAvailability(false),
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
		groqDiagnosticCapabilitySnapshot(cfg, *r),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "Groq",
			MissingTarget:                 "Groq model/configuration",
			UnknownAvailabilitySuggestion: "Use --catalog-model with a known Groq model before requiring catalog-gated capabilities",
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
