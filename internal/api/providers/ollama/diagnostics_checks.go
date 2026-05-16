package ollama

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func (r *DiagnosticReport) addCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	r.Checks = append(r.Checks, DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     detail,
		Suggestion: suggestion,
	})
}

func (r *DiagnosticReport) addAuthCheck() {
	r.addCheck(DiagnosticStatusOK, "auth", "Ollama does not require an API key", ollamaBaseURLEnv+" is optional", "")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("ollama") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "ollama provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "ollama provider is not registered", "", "Ensure providers/all imports the Ollama provider")
}

func (r *DiagnosticReport) addModelCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "Ollama model is not resolved", "", "Pass --model or set provider_models.ollama.default_model")
		return
	}
	r.addCheck(DiagnosticStatusOK, "model", "Ollama request model is resolved", fmt.Sprintf("%s (%s)", r.Model, r.ModelSource), "")
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "catalog_model", "Ollama catalog model is not resolved", r.Model, "Use --catalog-model when the request model is an alias")
		return
	}
	if ollamaCatalogModelKnown(r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusOK,
			"catalog_model",
			"Ollama catalog model is resolved",
			fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
			"",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusWarn,
		"catalog_model",
		"Ollama catalog model is not known to local metadata",
		fmt.Sprintf("model=%s catalog_model=%s (%s)", r.Model, r.CatalogModel, r.CatalogModelSource),
		"Set --catalog-model or provider_models.ollama.catalog_model to an Ollama model known to XELYON before relying on token-limit diagnostics",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	if r.Route != DiagnosticRouteOllamaChat {
		r.addCheck(DiagnosticStatusFail, "route", "Ollama route could not be resolved", r.RouteReason, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "route", "Ollama /api/chat route is selected", r.routeCheckDetail(), "")
}

func (r DiagnosticReport) routeCheckDetail() string {
	if strings.TrimSpace(r.RouteReason) == "" {
		return r.Route
	}
	return fmt.Sprintf("%s; %s", r.Route, r.RouteReason)
}

func (r *DiagnosticReport) addCatalogPolicyCheck(cfg *config.Config) {
	model := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if model == "" || catalogModel == "" {
		return
	}

	policy := providerdiag.OllamaCatalogPolicy(cfg, model, catalogModel)
	detail := policy.OllamaDetail()
	if !ollamaCatalogModelKnown(catalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not an Ollama model known to local metadata",
			detail,
			"Use an Ollama model known to XELYON before relying on token-limit diagnostics",
		)
		return
	}

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use an Ollama model known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use an Ollama model known to XELYON, or set max_output_tokens explicitly for this model")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing pricing metadata", detail, "Ollama pricing should be local zero-cost; check provider metadata")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"Ollama function calling payloads are enabled",
			"",
			"Set OLLAMA_FUNCTION_CALLING=0 only if the selected local model rejects tool payloads",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "Ollama function calling payloads are disabled", "OLLAMA_FUNCTION_CALLING=0", "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live Ollama smoke was skipped because prerequisite checks failed",
			"",
			"Fix failed checks, then rerun with --smoke",
		)
		return
	}
	if options.ToolSmoke && !r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusWarn,
			"tool_smoke",
			"tool payload smoke was skipped because function calling is disabled",
			"OLLAMA_FUNCTION_CALLING=0",
			"Unset OLLAMA_FUNCTION_CALLING or set it to 1 before rerunning --tool-smoke",
		)
	}

	smoke, err := runOllamaDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		if ollamaSmokeErrorIsToolFailure(smoke) {
			r.addCheck(DiagnosticStatusFail, "tool_smoke", "Ollama tool smoke response did not include the diagnostic tool call", err.Error(), "")
		}
		r.addCheck(DiagnosticStatusFail, "smoke", "live Ollama smoke request failed", err.Error(), "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live Ollama smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "Ollama endpoint accepted a tool payload", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "Ollama smoke usage was observed", ollamaDiagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"Ollama smoke succeeded but usage was not observed",
			"",
			"Check whether the local model returns prompt_eval_count/eval_count in the done chunk",
		)
	}

	if !smoke.UsageObserved {
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Ollama smoke cost estimate was skipped because usage was not observed",
			"",
			"Rerun smoke after usage metadata is available",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "cost", "Ollama smoke cost estimate is available", fmt.Sprintf("$%.8f USD", smoke.Cost.USD), "")
}

func ollamaDiagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	return fmt.Sprintf(
		"input=%d cached=%d output=%d reasoning=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}
