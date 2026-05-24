package claude

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

func (r *DiagnosticReport) addCapabilities(cfg *config.Config) {
	capabilities := buildClaudeDiagnosticCapabilities(cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"Claude model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildClaudeDiagnosticCapabilities(cfg *config.Config, report DiagnosticReport) DiagnosticCapabilities {
	return providerdiag.DiagnosticCapabilitiesFromSnapshot(claudeDiagnosticCapabilitySnapshot(cfg, report))
}

func claudeDiagnosticCapabilitySnapshot(cfg *config.Config, report DiagnosticReport) providerdiag.CapabilitySnapshot {
	policy := providerdiag.ClaudeCatalogPolicy(cfg, report.Model, report.CatalogModel)
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
		ImageInput:                     providerdiag.ModelGatedImageInputAvailability("claude", report.Model, report.CatalogModel, report.ImageInputSupported),
		WebSearch:                      providerdiag.ModelGatedWebSearchAvailability("claude", report.Model, report.CatalogModel, report.WebSearchSupported),
		Thinking:                       providerdiag.KnownCapabilityAvailability(claudeDiagnosticThinkingSupported(report)),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention:                      providerdiag.NewRetentionSnapshot(false, false, false),
		ServerCompaction: providerdiag.NewServerCompactionSnapshot(providerdiag.ServerCompactionSnapshotOptions{
			ResponsesAPI: false,
			Enabled:      report.ClaudeCompactionSupported,
		}),
		ContextWindowTokens: policy.ContextWindowTokens,
		ContextWindowKnown:  policy.ContextWindowKnown,
		MaxOutput:           policy.MaxOutput,
		Pricing:             policy.Pricing,
	}
}

func claudeDiagnosticThinkingSupported(report DiagnosticReport) bool {
	return report.Route == DiagnosticRouteClaudeMessages && report.ThinkingEnabled
}

func (r *DiagnosticReport) addRequiredCapabilities(cfg *config.Config, required []string) {
	diagnostic := providerdiag.NewRequiredCapabilityDiagnostic(
		claudeDiagnosticCapabilitySnapshot(cfg, *r),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "Claude",
			MissingTarget:                 "Claude model/configuration",
			UnknownAvailabilitySuggestion: "Use --catalog-model with a known Claude model before requiring catalog-gated capabilities",
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
