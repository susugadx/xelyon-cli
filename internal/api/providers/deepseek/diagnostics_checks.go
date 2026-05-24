package deepseek

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
	deepSeekAPIKeyEnv = "DEEPSEEK_API_KEY"
	deepSeekAPIURLEnv = "DEEPSEEK_API_URL"
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
	if strings.TrimSpace(os.Getenv(deepSeekAPIKeyEnv)) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s is required", deepSeekAPIKeyEnv),
			"",
			fmt.Sprintf("Set %s before running DeepSeek requests", deepSeekAPIKeyEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "DeepSeek API key is configured", deepSeekAPIKeyEnv, "")
}

func (r *DiagnosticReport) addEndpointCheck() {
	raw := strings.TrimSpace(os.Getenv(deepSeekAPIURLEnv))
	if raw == "" {
		r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s uses the built-in endpoint", deepSeekAPIURLEnv), defaultDeepSeekURL, "")
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			fmt.Sprintf("%s is not a valid absolute URL", deepSeekAPIURLEnv),
			raw,
			fmt.Sprintf("Set %s to a valid absolute URL such as %s", deepSeekAPIURLEnv, defaultDeepSeekURL),
		)
		return
	}

	if strings.TrimRight(parsed.Path, "/") != deepSeekChatCompletionsEndpointPath {
		r.addCheck(
			DiagnosticStatusWarn,
			"endpoint",
			fmt.Sprintf("%s does not end with %s", deepSeekAPIURLEnv, deepSeekChatCompletionsEndpointPath),
			raw,
			"This is OK only for an intentional proxy endpoint",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s is configured", deepSeekAPIURLEnv), r.APIURL, "")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("deepseek") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "deepseek provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "deepseek provider is not registered", "", "Ensure providers/all imports the DeepSeek provider")
}

func (r *DiagnosticReport) addModelCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "DeepSeek model is not resolved", "", "Pass --model or set provider_models.deepseek.default_model")
		return
	}
	r.addCheck(DiagnosticStatusOK, "model", "DeepSeek request model is resolved", fmt.Sprintf("%s (%s), api_model=%s", r.Model, r.ModelSource, r.APIModel), "")
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "catalog_model", "DeepSeek catalog model is not resolved", r.Model, "Use --catalog-model when the request model is an alias")
		return
	}
	if deepSeekCatalogModelKnown(r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusOK,
			"catalog_model",
			"DeepSeek catalog model is resolved",
			fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
			"",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusWarn,
		"catalog_model",
		"DeepSeek catalog model is not known to local metadata",
		fmt.Sprintf("model=%s catalog_model=%s (%s)", r.Model, r.CatalogModel, r.CatalogModelSource),
		"Set --catalog-model or provider_models.deepseek.catalog_model to a DeepSeek model known to XELYON before relying on token-limit diagnostics",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	if r.Route != DiagnosticRouteChatCompletions {
		r.addCheck(DiagnosticStatusFail, "route", "DeepSeek route could not be resolved", r.RouteReason, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "route", "DeepSeek Chat Completions route is selected", r.routeCheckDetail(), "")
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
	policy := providerdiag.DeepSeekCatalogPolicy(cfg, model, catalogModel)
	detail := policy.DeepSeekDetail()
	if !deepSeekCatalogModelKnown(catalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not a DeepSeek model known to local metadata",
			detail,
			"Use a DeepSeek model known to XELYON before relying on token-limit diagnostics",
		)
		return
	}

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a DeepSeek model known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use a DeepSeek model known to XELYON, or set max_output_tokens explicitly for this model")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing pricing metadata", detail, "Use a DeepSeek model with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func (r *DiagnosticReport) addThinkingCheck() {
	if !r.ThinkingSupported {
		status := DiagnosticStatusOK
		message := "DeepSeek thinking payload is not sent for this model"
		suggestion := ""
		if r.ThinkingEnabled {
			status = DiagnosticStatusWarn
			message = "DeepSeek thinking is requested but the catalog model does not support V4 thinking"
			suggestion = "Use deepseek-v4-flash, deepseek-v4-pro, or set catalog_model to a DeepSeek V4 model"
		}
		r.addCheck(status, "thinking", message, fmt.Sprintf("api_model=%s catalog_model=%s", r.APIModel, r.CatalogModel), suggestion)
		return
	}

	detail := fmt.Sprintf("thinking.type=%s", r.ThinkingType)
	if strings.TrimSpace(r.ReasoningEffort) != "" {
		detail += fmt.Sprintf(", reasoning_effort=%s", r.ReasoningEffort)
	}
	r.addCheck(DiagnosticStatusOK, "thinking", "DeepSeek thinking request config is resolved", detail, "")
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"DeepSeek function calling payloads are enabled",
			"",
			fmt.Sprintf("Set %s=0 only if the selected endpoint rejects tool payloads", deepSeekFunctionCallingEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "DeepSeek function calling payloads are disabled", fmt.Sprintf("%s=0", deepSeekFunctionCallingEnv), "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live DeepSeek smoke was skipped because prerequisite checks failed",
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
			fmt.Sprintf("%s=0", deepSeekFunctionCallingEnv),
			fmt.Sprintf("Unset %s or set it to 1 before rerunning --tool-smoke", deepSeekFunctionCallingEnv),
		)
	}

	smoke, err := runDeepSeekDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		failure := providerdiag.ClassifySmokeFailure(providerdiag.TextToolSmokeFailureContext(
			providerdiag.SmokeFailureContextOptions{
				Provider:         "DeepSeek",
				AuthEnv:          deepSeekAPIKeyEnv,
				EndpointEnv:      deepSeekAPIURLEnv,
				DebugEnv:         "XELYON_DEBUG_DEEPSEEK",
				EndpointOverride: strings.TrimSpace(os.Getenv(deepSeekAPIURLEnv)) != "",
			},
			smoke,
			err,
		))
		if deepSeekSmokeErrorIsToolFailure(smoke) {
			r.addCheck(DiagnosticStatusFail, "tool_smoke", "DeepSeek tool smoke response did not include the diagnostic tool call", failure.Detail, failure.Suggestion)
		}
		r.addCheck(DiagnosticStatusFail, "smoke", failure.Message, failure.Detail, failure.Suggestion)
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live DeepSeek smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "DeepSeek endpoint accepted a tool payload", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "DeepSeek smoke usage was observed", deepSeekDiagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"DeepSeek smoke succeeded but usage was not observed",
			"",
			"Check whether the endpoint returns stream_options.include_usage metadata",
		)
	}

	switch {
	case !smoke.UsageObserved:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"DeepSeek smoke cost was not estimated because usage was not observed",
			"",
			"Check usage metadata before relying on cost estimates",
		)
	case smoke.Cost.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"DeepSeek smoke usage was observed but pricing metadata is unavailable",
			deepSeekDiagnosticSmokeUsageDetail(smoke.Usage),
			"Set provider_models.deepseek.catalog_model to a DeepSeek model with pricing metadata",
		)
	default:
		r.addCheck(
			DiagnosticStatusOK,
			"cost",
			"DeepSeek smoke cost estimate is available",
			fmt.Sprintf("$%.6f USD", smoke.Cost.USD),
			"",
		)
	}
}

func deepSeekDiagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	return fmt.Sprintf(
		"input=%d cached_input=%d output=%d thinking=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}
