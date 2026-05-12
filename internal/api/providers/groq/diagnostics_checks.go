package groq

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	groqAPIKeyEnv = "GROQ_API_KEY"
	groqAPIURLEnv = "GROQ_API_URL"
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
	if strings.TrimSpace(os.Getenv(groqAPIKeyEnv)) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s is required", groqAPIKeyEnv),
			"",
			fmt.Sprintf("Set %s before running Groq requests", groqAPIKeyEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "Groq API key is configured", groqAPIKeyEnv, "")
}

func (r *DiagnosticReport) addEndpointCheck() {
	raw := strings.TrimSpace(os.Getenv(groqAPIURLEnv))
	if raw == "" {
		r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s uses the built-in endpoint", groqAPIURLEnv), defaultGroqURL, "")
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			fmt.Sprintf("%s is not a valid absolute URL", groqAPIURLEnv),
			raw,
			fmt.Sprintf("Set %s to a valid absolute URL such as %s", groqAPIURLEnv, defaultGroqURL),
		)
		return
	}

	if strings.TrimRight(parsed.Path, "/") != "/openai/v1/chat/completions" {
		r.addCheck(
			DiagnosticStatusWarn,
			"endpoint",
			fmt.Sprintf("%s does not end with /openai/v1/chat/completions", groqAPIURLEnv),
			raw,
			"This is OK only for an intentional proxy endpoint",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s is configured", groqAPIURLEnv), r.APIURL, "")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("groq") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "groq provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "groq provider is not registered", "", "Ensure providers/all imports the Groq provider")
}

func (r *DiagnosticReport) addModelCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "Groq model is not resolved", "", "Pass --model or set provider_models.groq.default_model")
		return
	}
	r.addCheck(DiagnosticStatusOK, "model", "Groq request model is resolved", fmt.Sprintf("%s (%s)", r.Model, r.ModelSource), "")
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "catalog_model", "Groq catalog model is not resolved", r.Model, "Use --catalog-model when the request model is an alias")
		return
	}
	if groqCatalogModelKnown(r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusOK,
			"catalog_model",
			"Groq catalog model is resolved",
			fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
			"",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusWarn,
		"catalog_model",
		"Groq catalog model is not known to local metadata",
		fmt.Sprintf("model=%s catalog_model=%s (%s)", r.Model, r.CatalogModel, r.CatalogModelSource),
		"Set --catalog-model or provider_models.groq.catalog_model to a Groq model known to XELYON before relying on token-limit diagnostics",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	if r.Route != DiagnosticRouteChatCompletions {
		r.addCheck(DiagnosticStatusFail, "route", "Groq route could not be resolved", r.RouteReason, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "route", "Groq Chat Completions route is selected", r.routeCheckDetail(), "")
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
	if !groqCatalogModelKnown(catalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not a Groq model known to local metadata",
			fmt.Sprintf("catalog_model=%s, context_window=unknown, max_output_tokens=unknown, pricing=unavailable", catalogModel),
			"Use a Groq model known to XELYON before relying on token-limit diagnostics",
		)
		return
	}

	policy := providerdiag.GroqCatalogPolicy(cfg, model, catalogModel)
	detail := policy.GroqDetail()

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a Groq model known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use a Groq model known to XELYON, or set max_output_tokens explicitly for this model")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing pricing metadata", detail, "Use a Groq model with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"Groq function calling payloads are enabled",
			"",
			"Set GROQ_FUNCTION_CALLING=0 only if the selected endpoint rejects tool payloads",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "Groq function calling payloads are disabled", "GROQ_FUNCTION_CALLING=0", "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live Groq smoke was skipped because prerequisite checks failed",
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
			"GROQ_FUNCTION_CALLING=0",
			"Unset GROQ_FUNCTION_CALLING or set it to 1 before rerunning --tool-smoke",
		)
	}

	smoke, err := runGroqDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		if groqSmokeErrorIsToolFailure(smoke) {
			r.addCheck(DiagnosticStatusFail, "tool_smoke", "Groq tool smoke response did not include the diagnostic tool call", err.Error(), "")
		}
		r.addCheck(DiagnosticStatusFail, "smoke", "live Groq smoke request failed", err.Error(), "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live Groq smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "Groq endpoint accepted a tool payload", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "Groq smoke usage was observed", groqDiagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"Groq smoke succeeded but usage was not observed",
			"",
			"Check whether the endpoint returns stream_options.include_usage metadata",
		)
	}

	switch {
	case !smoke.UsageObserved:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Groq smoke cost estimate was skipped because usage was not observed",
			"",
			"Rerun smoke after usage metadata is available",
		)
	case smoke.Cost.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Groq smoke cost pricing is unavailable",
			"",
			"Use a Groq catalog model with pricing metadata before relying on smoke cost estimates",
		)
	default:
		r.addCheck(DiagnosticStatusOK, "cost", "Groq smoke cost estimate is available", fmt.Sprintf("$%.8f USD", smoke.Cost.USD), "")
	}
}

func groqDiagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	return fmt.Sprintf(
		"input=%d cached=%d output=%d reasoning=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}
