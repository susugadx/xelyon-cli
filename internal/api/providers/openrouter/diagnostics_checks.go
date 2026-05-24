package openrouter

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
	openRouterAPIKeyEnv = "OPENROUTER_API_KEY"
	openRouterAPIURLEnv = "OPENROUTER_API_URL"
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
	if strings.TrimSpace(os.Getenv(openRouterAPIKeyEnv)) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s is required", openRouterAPIKeyEnv),
			"",
			fmt.Sprintf("Set %s before running OpenRouter requests", openRouterAPIKeyEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "OpenRouter API key is configured", openRouterAPIKeyEnv, "")
}

func (r *DiagnosticReport) addEndpointCheck(route openRouterRoutePlan) {
	raw := strings.TrimSpace(os.Getenv(openRouterAPIURLEnv))
	if raw == "" {
		r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s uses the built-in endpoint", openRouterAPIURLEnv), defaultOpenRouterURL, "")
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			fmt.Sprintf("%s is not a valid absolute URL", openRouterAPIURLEnv),
			raw,
			fmt.Sprintf("Set %s to a valid absolute URL such as %s", openRouterAPIURLEnv, defaultOpenRouterURL),
		)
		return
	}

	path := strings.TrimRight(parsed.Path, "/")
	if openRouterEndpointLooksLikeMessagesPath(path) {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			fmt.Sprintf("%s is an Anthropic Skin Messages endpoint, but %s route expects a Chat Completions endpoint or proxy", openRouterAPIURLEnv, route.Route),
			raw,
			fmt.Sprintf("Set %s to %s or to an intentional proxy path; the provider derives %s for anthropic_messages route", openRouterAPIURLEnv, defaultOpenRouterURL, openRouterAnthropicMessagesEndpointPath),
		)
		return
	}
	if !strings.HasSuffix(path, openRouterChatCompletionsEndpointPath) {
		r.addCheck(
			DiagnosticStatusWarn,
			"endpoint",
			fmt.Sprintf("%s does not end with %s", openRouterAPIURLEnv, openRouterChatCompletionsEndpointPath),
			raw,
			fmt.Sprintf("This is OK only for an intentional proxy endpoint; Anthropic Skin preview/smoke derives %s from this URL", openRouterAnthropicMessagesEndpointPath),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s is configured", openRouterAPIURLEnv), raw, "")
}

func openRouterEndpointLooksLikeMessagesPath(path string) bool {
	return strings.HasSuffix(strings.TrimRight(path, "/"), openRouterAnthropicMessagesPathSuffix)
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("openrouter") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "openrouter provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "openrouter provider is not registered", "", "Ensure providers/all imports the OpenRouter provider")
}

func (r *DiagnosticReport) addModelCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "OpenRouter model is not resolved", "", "Pass --model or set provider_models.openrouter.default_model")
		return
	}
	r.addCheck(DiagnosticStatusOK, "model", "OpenRouter request model is resolved", fmt.Sprintf("%s (%s)", r.Model, r.ModelSource), "")
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "catalog_model", "OpenRouter catalog model is not resolved", r.Model, "Use --catalog-model when the request model is an alias")
		return
	}
	catalogUse := resolveOpenRouterDiagnosticCatalogModelUse(r.Model, r.CatalogModel)
	if catalogUse.CatalogKnown {
		if !catalogUse.Trusted {
			r.addCheck(
				DiagnosticStatusWarn,
				"catalog_model",
				"OpenRouter catalog_model does not match the routed request model",
				catalogUse.MismatchDetail,
				catalogUse.MismatchSuggestion,
			)
			return
		}
		r.addCheck(
			DiagnosticStatusOK,
			"catalog_model",
			"OpenRouter catalog model is resolved",
			fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
			"",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusWarn,
		"catalog_model",
		"OpenRouter catalog model is not known to local metadata",
		fmt.Sprintf("model=%s catalog_model=%s (%s)", r.Model, r.CatalogModel, r.CatalogModelSource),
		"Set --catalog-model or provider_models.openrouter.catalog_model to an OpenRouter model known to XELYON before relying on token-limit diagnostics",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	switch r.Route {
	case DiagnosticRouteChatCompletions, DiagnosticRouteAnthropicMessages:
	default:
		r.addCheck(DiagnosticStatusFail, "route", "OpenRouter route could not be resolved", r.RouteReason, "")
		return
	}

	if r.Route == DiagnosticRouteChatCompletions && openRouterCatalogModelKnown(r.CatalogModel) && isClaudeModel(r.CatalogModel) && !isClaudeModel(r.Model) {
		r.addCheck(
			DiagnosticStatusWarn,
			"route",
			"OpenRouter route is selected from the request model, not catalog_model",
			r.routeCheckDetail(),
			"Use an anthropic/claude-* request model if you need OpenRouter Anthropic Skin context management",
		)
		return
	}

	r.addCheck(DiagnosticStatusOK, "route", "OpenRouter request route is selected", r.routeCheckDetail(), "")
}

func (r DiagnosticReport) routeCheckDetail() string {
	detail := r.Route
	if strings.TrimSpace(r.APIURL) != "" {
		detail += fmt.Sprintf(" url=%s", r.APIURL)
	}
	if strings.TrimSpace(r.RouteReason) != "" {
		detail += fmt.Sprintf("; %s", r.RouteReason)
	}
	if strings.TrimSpace(r.UpstreamProvider) != "" || strings.TrimSpace(r.UpstreamModel) != "" {
		detail += fmt.Sprintf("; upstream=%s/%s", r.UpstreamProvider, r.UpstreamModel)
	}
	return detail
}

func (r *DiagnosticReport) addCatalogPolicyCheck(cfg *config.Config) {
	model := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if model == "" || catalogModel == "" {
		return
	}
	if !openRouterCatalogModelKnown(catalogModel) {
		policy := providerdiag.OpenRouterCatalogPolicy(cfg, model, catalogModel)
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not an OpenRouter model known to local metadata",
			policy.OpenRouterDetail(),
			"Use an OpenRouter model known to XELYON before relying on token-limit diagnostics",
		)
		return
	}

	catalogUse := resolveOpenRouterDiagnosticCatalogModelUse(model, catalogModel)
	if !catalogUse.Trusted {
		policy := providerdiag.OpenRouterCatalogPolicy(cfg, model, catalogUse.PolicyCatalogModel)
		if strings.TrimSpace(catalogUse.PolicyCatalogModel) == "" || !openRouterCatalogModelKnown(catalogUse.PolicyCatalogModel) {
			r.addCheck(
				DiagnosticStatusWarn,
				"catalog_policy",
				"catalog_model is not trusted for this OpenRouter request model",
				fmt.Sprintf("%s; %s", catalogUse.MismatchDetail, openRouterDiagnosticCatalogPolicyDetail(policy, catalogModel)),
				catalogUse.MismatchSuggestion,
			)
			return
		}

		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not trusted for this OpenRouter request model; policy uses request model metadata",
			fmt.Sprintf("%s; %s", catalogUse.MismatchDetail, openRouterDiagnosticCatalogPolicyDetail(policy, catalogModel)),
			catalogUse.MismatchSuggestion,
		)
		return
	}

	policy := providerdiag.OpenRouterCatalogPolicy(cfg, model, catalogUse.PolicyCatalogModel)
	detail := policy.OpenRouterDetail()

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use an OpenRouter model known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use an OpenRouter model known to XELYON, or set max_output_tokens explicitly for this model")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing pricing metadata", detail, "Use an OpenRouter catalog model with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func openRouterDiagnosticCatalogPolicyDetail(policy providerdiag.CatalogPolicy, requestedCatalogModel string) string {
	if strings.EqualFold(strings.TrimSpace(requestedCatalogModel), strings.TrimSpace(policy.CatalogModel)) {
		return policy.OpenRouterDetail()
	}
	return fmt.Sprintf("requested_catalog_model=%s, policy_%s", strings.TrimSpace(requestedCatalogModel), policy.OpenRouterDetail())
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"OpenRouter function calling payloads are enabled",
			"",
			"Set OPENROUTER_FUNCTION_CALLING=0 only if the selected endpoint rejects tool payloads",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "OpenRouter function calling payloads are disabled", "OPENROUTER_FUNCTION_CALLING=0", "")
}

func (r *DiagnosticReport) addImageInputCheck() {
	if r.ImageInputSupported {
		r.addCheck(
			DiagnosticStatusOK,
			"image_input",
			"OpenRouter provider image input path is available",
			"model support is controlled by the selected upstream OpenRouter model",
			"",
		)
		return
	}
	r.addCheck(DiagnosticStatusWarn, "image_input", "OpenRouter provider image input path is unavailable", "", "Use a provider/model with image input support")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live OpenRouter smoke was skipped because prerequisite checks failed",
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
			"OPENROUTER_FUNCTION_CALLING=0",
			"Unset OPENROUTER_FUNCTION_CALLING or set it to 1 before rerunning --tool-smoke",
		)
	}

	smoke, err := runOpenRouterDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		failure := providerdiag.ClassifySmokeFailure(providerdiag.TextToolSmokeFailureContext(
			providerdiag.SmokeFailureContextOptions{
				Provider:         "OpenRouter",
				AuthEnv:          openRouterAPIKeyEnv,
				EndpointEnv:      openRouterAPIURLEnv,
				EndpointOverride: strings.TrimSpace(os.Getenv(openRouterAPIURLEnv)) != "",
			},
			smoke,
			err,
		))
		if openRouterSmokeErrorIsToolFailure(smoke) {
			r.addCheck(DiagnosticStatusFail, "tool_smoke", "OpenRouter tool smoke response did not include the diagnostic tool call", failure.Detail, failure.Suggestion)
		}
		r.addCheck(DiagnosticStatusFail, "smoke", failure.Message, failure.Detail, failure.Suggestion)
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live OpenRouter smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "OpenRouter endpoint accepted a tool payload", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "OpenRouter smoke usage was observed", openRouterDiagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"OpenRouter smoke succeeded but usage was not observed",
			"",
			"Check whether the endpoint returns usage metadata for the selected route",
		)
	}

	switch {
	case !smoke.UsageObserved:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"OpenRouter smoke cost estimate was skipped because usage was not observed",
			"",
			"Rerun smoke after usage metadata is available",
		)
	case smoke.Cost.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"OpenRouter smoke cost pricing is unavailable",
			"",
			"Use an OpenRouter catalog model with pricing metadata before relying on smoke cost estimates",
		)
	default:
		r.addCheck(DiagnosticStatusOK, "cost", "OpenRouter smoke cost estimate is available", fmt.Sprintf("$%.8f USD", smoke.Cost.USD), "")
	}
}

func openRouterDiagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	return fmt.Sprintf(
		"input=%d cached=%d output=%d reasoning=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}
