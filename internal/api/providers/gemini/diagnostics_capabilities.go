package gemini

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

func (r *DiagnosticReport) addCapabilities(cfg *config.Config) {
	capabilities := buildGeminiDiagnosticCapabilities(cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"Gemini model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildGeminiDiagnosticCapabilities(cfg *config.Config, report DiagnosticReport) DiagnosticCapabilities {
	return providerdiag.DiagnosticCapabilitiesFromSnapshot(geminiDiagnosticCapabilitySnapshot(cfg, report))
}

func geminiDiagnosticCapabilitySnapshot(cfg *config.Config, report DiagnosticReport) providerdiag.CapabilitySnapshot {
	policy := providerdiag.GeminiCatalogPolicy(cfg, report.Model, report.CatalogModel)
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
		ImageInput:                     geminiDiagnosticModelCapabilityAvailability(report.ImageInputSupported, report),
		WebSearch:                      providerdiag.ModelGatedWebSearchAvailability("gemini", report.Model, report.CatalogModel, report.WebSearchSupported),
		Thinking:                       geminiDiagnosticThinkingAvailability(report),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention:                      providerdiag.NewRetentionSnapshot(false, false, false),
		ServerCompaction:               providerdiag.NewServerCompactionSnapshot(providerdiag.ServerCompactionSnapshotOptions{}),
		ContextWindowTokens:            policy.ContextWindowTokens,
		ContextWindowKnown:             policy.ContextWindowKnown,
		MaxOutput:                      policy.MaxOutput,
		Pricing:                        policy.Pricing,
	}
}

func geminiDiagnosticThinkingAvailability(report DiagnosticReport) providerdiag.CapabilityAvailability {
	model, ok := geminiDiagnosticModelCapabilityModel(report)
	if !ok {
		return providerdiag.UnknownCapabilityAvailability()
	}
	if !isGeminiDiagnosticThinkingModel(model) {
		return providerdiag.KnownCapabilityAvailability(false)
	}
	if isGemini3Model(model) {
		return providerdiag.KnownCapabilityAvailability(true)
	}
	return providerdiag.KnownCapabilityAvailability(report.ThinkingEnabled)
}

func geminiDiagnosticModelCapabilityAvailability(providerSupported bool, report DiagnosticReport) providerdiag.CapabilityAvailability {
	if !providerSupported {
		return providerdiag.KnownCapabilityAvailability(false)
	}
	if !geminiDiagnosticModelCapabilityMetadataKnown(report) {
		return providerdiag.UnknownCapabilityAvailability()
	}
	return providerdiag.KnownCapabilityAvailability(true)
}

func geminiDiagnosticModelCapabilityMetadataKnown(report DiagnosticReport) bool {
	_, ok := geminiDiagnosticModelCapabilityModel(report)
	return ok
}

func geminiDiagnosticModelCapabilityModel(report DiagnosticReport) (string, bool) {
	if providerdiag.IsProviderCatalogModelListed("gemini", report.Model) {
		return strings.TrimSpace(report.Model), true
	}
	if providerdiag.IsProviderCatalogModelListed("gemini", report.CatalogModel) {
		return strings.TrimSpace(report.CatalogModel), true
	}
	return "", false
}

func isGeminiDiagnosticThinkingModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return isGemini3Model(model) || strings.Contains(model, "gemini-2.5")
}

func (r *DiagnosticReport) addRequiredCapabilities(cfg *config.Config, required []string) {
	diagnostic := providerdiag.NewRequiredCapabilityDiagnostic(
		geminiDiagnosticCapabilitySnapshot(cfg, *r),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "Gemini",
			MissingTarget:                 "Gemini model/configuration",
			UnknownAvailabilitySuggestion: "Use --catalog-model with a known Gemini model before requiring catalog-gated capabilities",
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
