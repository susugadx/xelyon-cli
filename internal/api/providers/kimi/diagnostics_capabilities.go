package kimi

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

func (r *DiagnosticReport) addCapabilities(cfg *config.Config) {
	capabilities := buildKimiDiagnosticCapabilities(cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"Kimi model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildKimiDiagnosticCapabilities(cfg *config.Config, report DiagnosticReport) DiagnosticCapabilities {
	return providerdiag.DiagnosticCapabilitiesFromSnapshot(kimiDiagnosticCapabilitySnapshot(cfg, report))
}

func kimiDiagnosticCapabilitySnapshot(cfg *config.Config, report DiagnosticReport) providerdiag.CapabilitySnapshot {
	policy := providerdiag.KimiCatalogPolicy(cfg, report.Model, report.CatalogModel)
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Model,
		CatalogModel:                   report.CatalogModel,
		Route:                          report.Route,
		RouteReason:                    report.RouteReason,
		ResponsesAPI:                   false,
		ResponsesStreaming:             false,
		ResponsesStreamingAvailability: providerdiag.KnownCapabilityAvailability(false),
		ChatCompletions:                report.Route == DiagnosticRouteChatCompletions || report.Route == DiagnosticRouteChatCompletionsWebSearch,
		FunctionCalling:                report.FunctionCallingEnabled,
		ImageInput:                     providerdiag.KnownCapabilityAvailability(true),
		WebSearch:                      providerdiag.KnownCapabilityAvailability(true),
		Thinking:                       kimiDiagnosticThinkingAvailability(cfg, report),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention:                      providerdiag.NewRetentionSnapshot(false, false, false),
		ServerCompaction:               providerdiag.NewServerCompactionSnapshot(providerdiag.ServerCompactionSnapshotOptions{}),
		ContextWindowTokens:            policy.ContextWindowTokens,
		ContextWindowKnown:             policy.ContextWindowKnown,
		MaxOutput:                      policy.MaxOutput,
		Pricing:                        policy.Pricing,
	}
}

func kimiDiagnosticThinkingAvailability(cfg *config.Config, report DiagnosticReport) providerdiag.CapabilityAvailability {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	_, thinkingActive, _ := kimiThinkingConfigForResolved(kimiResolvedRequestOptions{
		ctx:               config.WithContext(context.Background(), cfg),
		providerConfigKey: "kimi",
		requestedModel:    report.Model,
		catalogModel:      report.CatalogModel,
	})
	return providerdiag.KnownCapabilityAvailability(thinkingActive)
}

func (r *DiagnosticReport) addRequiredCapabilities(cfg *config.Config, required []string) {
	diagnostic := providerdiag.NewRequiredCapabilityDiagnostic(
		kimiDiagnosticCapabilitySnapshot(cfg, *r),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "Kimi",
			MissingTarget:                 "Kimi model/configuration",
			UnknownAvailabilitySuggestion: "Use --catalog-model with a known Kimi model before requiring catalog-gated capabilities",
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
