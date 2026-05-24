package bedrock

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

func (r *DiagnosticReport) addCapabilities(cfg *config.Config, route bedrockRoute) {
	capabilities := buildBedrockDiagnosticCapabilities(cfg, *r, route)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"Bedrock model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildBedrockDiagnosticCapabilities(cfg *config.Config, report DiagnosticReport, route bedrockRoute) DiagnosticCapabilities {
	return providerdiag.DiagnosticCapabilitiesFromSnapshot(bedrockDiagnosticCapabilitySnapshot(cfg, report, route))
}

func bedrockDiagnosticCapabilitySnapshot(cfg *config.Config, report DiagnosticReport, route bedrockRoute) providerdiag.CapabilitySnapshot {
	policy := bedrockDiagnosticCatalogPolicy(cfg, route, report.Model, report.CatalogModel)
	claudeMessages := route == bedrockRouteClaudeMessages
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Model,
		CatalogModel:                   report.CatalogModel,
		Route:                          report.Route,
		ResponsesAPI:                   false,
		ResponsesStreaming:             false,
		ResponsesStreamingAvailability: providerdiag.KnownCapabilityAvailability(false),
		ChatCompletions:                false,
		FunctionCalling:                bedrockDiagnosticFunctionCallingAvailable(report, route),
		ImageInput:                     providerdiag.KnownCapabilityAvailability(claudeMessages),
		WebSearch:                      providerdiag.KnownCapabilityAvailability(false),
		Thinking:                       bedrockDiagnosticThinkingAvailability(cfg, route),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention:                      providerdiag.NewRetentionSnapshot(false, false, false),
		ServerCompaction:               providerdiag.NewServerCompactionSnapshot(providerdiag.ServerCompactionSnapshotOptions{}),
		ContextWindowTokens:            policy.ContextWindowTokens,
		ContextWindowKnown:             policy.ContextWindowKnown,
		MaxOutput:                      policy.MaxOutput,
		Pricing:                        policy.Pricing,
	}
}

func bedrockDiagnosticThinkingAvailability(cfg *config.Config, route bedrockRoute) providerdiag.CapabilityAvailability {
	if route != bedrockRouteClaudeMessages {
		return providerdiag.KnownCapabilityAvailability(false)
	}
	return providerdiag.KnownCapabilityAvailability(api.IsThinkingEnabled(config.WithContext(context.Background(), cfg)))
}

func bedrockDiagnosticFunctionCallingAvailable(report DiagnosticReport, route bedrockRoute) bool {
	if !report.FunctionCallingEnabled {
		return false
	}
	switch route {
	case bedrockRouteClaudeMessages:
		return true
	case bedrockRouteConverseStream:
		return llmcatalog.BedrockConverseToolUseSupported(report.Model, report.CatalogModel)
	default:
		return false
	}
}

func (r *DiagnosticReport) addRequiredCapabilities(cfg *config.Config, route bedrockRoute, required []string) {
	diagnostic := providerdiag.NewRequiredCapabilityDiagnostic(
		bedrockDiagnosticCapabilitySnapshot(cfg, *r, route),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "Bedrock",
			MissingTarget:                 "Bedrock model/configuration",
			UnknownAvailabilitySuggestion: "Use --catalog-model with a known Bedrock model before requiring catalog-gated capabilities",
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
