package kimi

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

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

func (r *DiagnosticReport) addAPIURLCheck() {
	raw := strings.TrimSpace(os.Getenv(kimiAPIURLEnv))
	if raw == "" {
		r.addCheck(DiagnosticStatusOK, "api_url", "Kimi API URL uses the built-in Moonshot endpoint", defaultKimiURL, "")
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"api_url",
			fmt.Sprintf("%s is not a valid absolute URL", kimiAPIURLEnv),
			raw,
			fmt.Sprintf("Set %s to a full chat completions endpoint such as %s", kimiAPIURLEnv, defaultKimiURL),
		)
		return
	}

	if strings.TrimRight(parsed.Path, "/") != "/v1/chat/completions" {
		r.addCheck(
			DiagnosticStatusWarn,
			"api_url_path",
			fmt.Sprintf("%s does not end with /v1/chat/completions", kimiAPIURLEnv),
			raw,
			"This is OK only for an intentional proxy endpoint",
		)
	}
	r.addCheck(DiagnosticStatusOK, "api_url", "Kimi API URL is configured", raw, "")
}

func (r *DiagnosticReport) addAuthCheck() {
	if strings.TrimSpace(os.Getenv(kimiAPIKeyEnv)) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s is required", kimiAPIKeyEnv),
			"",
			fmt.Sprintf("Set %s to a Moonshot API key before running Kimi requests", kimiAPIKeyEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "Moonshot API key is configured", kimiAPIKeyEnv, "")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if !api.IsRegisteredProvider("kimi") {
		r.addCheck(DiagnosticStatusFail, "provider_registration", "kimi provider is not registered", "", "Ensure providers/all imports the Kimi provider")
		return
	}
	if !api.IsRegisteredProvider("moonshot") {
		r.addCheck(DiagnosticStatusFail, "provider_registration", "moonshot alias provider is not registered", "", "Ensure the Kimi provider registers the moonshot alias")
		return
	}
	r.addCheck(DiagnosticStatusOK, "provider_registration", "kimi and moonshot providers are registered", "", "")
}

func (r *DiagnosticReport) addModelCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "Kimi model is not resolved", "", "Pass --model kimi-k2.6 or set provider_models.kimi.default_model")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"model",
		"Kimi request model is resolved",
		fmt.Sprintf("%s (%s)", r.Model, r.ModelSource),
		"",
	)
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "catalog_model", "Kimi catalog model is not resolved", r.Model, "Use --catalog-model when the request model is an alias")
		return
	}
	if !providerdiag.IsProviderCatalogModelKnown("kimi", r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_model",
			"Kimi catalog model is not known to local Kimi metadata",
			fmt.Sprintf("model=%s catalog_model=%s (%s)", r.Model, r.CatalogModel, r.CatalogModelSource),
			"Set --catalog-model or provider_models.kimi.catalog_model to kimi-k2.6 or kimi-k2.5 before relying on token-limit diagnostics",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"catalog_model",
		"Kimi catalog model is resolved",
		fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
		"",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	if r.Route != DiagnosticRouteChatCompletions {
		r.addCheck(DiagnosticStatusFail, "route", "Kimi route could not be resolved", r.RouteReason, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "route", "Kimi Chat Completions route is selected", r.routeCheckDetail(), "")
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
	policy := providerdiag.KimiCatalogPolicy(cfg, model, catalogModel)
	detail := policy.KimiDetail()
	if !providerdiag.IsProviderCatalogModelKnown("kimi", catalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not a Kimi model known to local metadata",
			detail,
			"Use a Kimi catalog model before relying on token-limit diagnostics or cost estimates",
		)
		return
	}

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a Kimi model known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use a Kimi model known to XELYON, or set max_output_tokens explicitly for this model")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing Kimi pricing metadata", detail, "Use a Kimi model with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"Kimi function calling payloads are enabled",
			"",
			fmt.Sprintf("Set %s=0 only if the selected endpoint rejects tool payloads", kimiFunctionCallingEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "Kimi function calling payloads are disabled", kimiFunctionCallingEnv+"=0", "")
}

func (r *DiagnosticReport) addImageInputCheck() {
	r.addCheck(
		DiagnosticStatusOK,
		"image_input",
		"Kimi native provider supports base64 image input",
		"data:image/{png,jpeg,webp,gif};base64,...",
		"URL images, video input, and file upload are not implemented in XELYON's Kimi provider",
	)
}

func (r *DiagnosticReport) addUnsupportedFeaturesCheck() {
	r.addCheck(
		DiagnosticStatusInfo,
		"unsupported_features",
		"Kimi video, memory, code runner, and file upload are not implemented in the native provider",
		strings.Join(r.UnsupportedFeatures, ", "),
		"",
	)
}

func (r *DiagnosticReport) addPromptCacheKeyCheck(ctx context.Context, cfg *config.Config) {
	dryCtx := newKimiDiagnosticContext(ctx, cfg, false, io.Discard)
	dryCtx = api.WithPromptCacheScope(dryCtx, api.PromptCacheScope{SessionID: "xelyon-kimi-doctor"})

	provider := New("diagnostic-key")
	built := provider.buildChatCompletionsRequest(
		dryCtx,
		"Kimi doctor dry run.",
		[]api.Message{{Role: "user", Content: "Build a diagnostic request only."}},
		r.Model,
	)
	r.PromptCacheKeyPresent = strings.TrimSpace(built.Request.PromptCacheKey) != ""
	if !r.PromptCacheKeyPresent {
		r.addCheck(
			DiagnosticStatusFail,
			"prompt_cache_key",
			"Kimi request builder did not attach prompt_cache_key",
			"",
			"Keep Kimi diagnostics and ChatWithTools on the shared request builder",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "prompt_cache_key", "Kimi request builder attaches prompt_cache_key", built.Request.PromptCacheKey, "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live Kimi smoke was skipped because prerequisite checks failed",
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
			kimiFunctionCallingEnv+"=0",
			fmt.Sprintf("Unset %s or set it to 1 before rerunning --tool-smoke", kimiFunctionCallingEnv),
		)
	}

	started := time.Now()
	smoke, err := runKimiDiagnosticSmoke(ctx, cfg, *r, options)
	smoke.Duration = time.Since(started).Round(time.Millisecond).String()
	r.Smoke = &smoke
	if err != nil {
		if isTransientKimiSmokeError(err) {
			r.addCheck(DiagnosticStatusWarn, "smoke", "live Kimi smoke hit a transient API condition", err.Error(), "Retry later or rerun with a longer --timeout")
			return
		}
		r.addCheck(DiagnosticStatusFail, "smoke", "live Kimi smoke request failed", err.Error(), "")
		return
	}

	r.addCheck(DiagnosticStatusOK, "smoke", "live Kimi smoke request succeeded", smoke.Duration, "")
	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "Kimi usage callback was observed", fmt.Sprintf("cached_input_tokens=%d", smoke.CachedInputTokens), "")
	} else {
		r.addCheck(DiagnosticStatusWarn, "usage", "Kimi smoke succeeded but usage callback was not observed", "", "Check whether the endpoint returns stream_options.include_usage")
	}
	r.addCheck(DiagnosticStatusInfo, "cached_tokens", "Kimi cached token observation is informational", fmt.Sprintf("cached_input_tokens=%d", smoke.CachedInputTokens), "")
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "Kimi endpoint accepted a dummy tool payload", smoke.Duration, "")
	}
	if smoke.ImagePayload {
		r.addCheck(DiagnosticStatusOK, "image_smoke", "Kimi endpoint accepted a base64 image payload", smoke.Duration, "")
	}
	if smoke.WebSearchPayload {
		status := DiagnosticStatusOK
		message := "Kimi endpoint completed a built-in $web_search request"
		suggestion := "Kimi charges a separate web search call fee when $web_search is triggered; search result tokens are counted in the next prompt_tokens response"
		if smoke.WebSearchCallCount <= 0 {
			status = DiagnosticStatusFail
			message = "Kimi web search smoke did not trigger a built-in $web_search call"
			suggestion = "Check Kimi model/tool support and rerun --web-search-smoke; the smoke is only valid when $web_search calls are observed"
		}
		r.addCheck(
			status,
			"web_search_smoke",
			message,
			fmt.Sprintf("calls=%d fee_estimate=$%.4f search_result_tokens=%d", smoke.WebSearchCallCount, smoke.WebSearchCallFeeEstimate, smoke.SearchResultTotalTokens),
			suggestion,
		)
	}
}
