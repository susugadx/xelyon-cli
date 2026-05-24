package ollama

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

func (r *DiagnosticReport) addCapabilities(cfg *config.Config, localModelAvailable providerdiag.CapabilityAvailability) {
	capabilities := buildOllamaDiagnosticCapabilities(cfg, *r, localModelAvailable)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"Ollama model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildOllamaDiagnosticCapabilities(cfg *config.Config, report DiagnosticReport, localModelAvailable providerdiag.CapabilityAvailability) DiagnosticCapabilities {
	return providerdiag.DiagnosticCapabilitiesFromSnapshot(ollamaDiagnosticCapabilitySnapshot(cfg, report, localModelAvailable))
}

func ollamaDiagnosticCapabilitySnapshot(cfg *config.Config, report DiagnosticReport, localModelAvailable providerdiag.CapabilityAvailability) providerdiag.CapabilitySnapshot {
	policy := providerdiag.OllamaCatalogPolicy(cfg, report.Model, report.CatalogModel)
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Model,
		CatalogModel:                   report.CatalogModel,
		Route:                          report.Route,
		RouteReason:                    report.RouteReason,
		ResponsesAPI:                   false,
		ResponsesStreaming:             false,
		ResponsesStreamingAvailability: providerdiag.KnownCapabilityAvailability(false),
		ChatCompletions:                false,
		FunctionCalling:                report.FunctionCallingEnabled,
		ImageInput:                     providerdiag.KnownCapabilityAvailability(false),
		WebSearch:                      providerdiag.KnownCapabilityAvailability(false),
		Thinking:                       providerdiag.KnownCapabilityAvailability(false),
		LocalModelAvailable:            localModelAvailable,
		Retention:                      providerdiag.NewRetentionSnapshot(false, false, false),
		ServerCompaction:               providerdiag.NewServerCompactionSnapshot(providerdiag.ServerCompactionSnapshotOptions{}),
		ContextWindowTokens:            policy.ContextWindowTokens,
		ContextWindowKnown:             policy.ContextWindowKnown,
		MaxOutput:                      policy.MaxOutput,
		Pricing:                        policy.Pricing,
	}
}

func (r *DiagnosticReport) addRequiredCapabilities(cfg *config.Config, required []string, localModelAvailable providerdiag.CapabilityAvailability) {
	diagnostic := providerdiag.NewRequiredCapabilityDiagnostic(
		ollamaDiagnosticCapabilitySnapshot(cfg, *r, localModelAvailable),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "Ollama",
			MissingTarget:                 "Ollama model/configuration",
			UnknownAvailabilitySuggestion: "Start Ollama and ensure /api/tags returns the model before requiring local_model_available",
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
