package gemini

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	geminiAPIKeyEnv         = "GEMINI_API_KEY"
	geminiAPIURLEnv         = "GEMINI_API_URL"
	geminiContextCachingEnv = "GEMINI_CONTEXT_CACHING"
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

func (r *DiagnosticReport) setCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	check := DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     detail,
		Suggestion: suggestion,
	}
	for i := range r.Checks {
		if r.Checks[i].Name == name {
			r.Checks[i] = check
			return
		}
	}
	r.Checks = append(r.Checks, check)
}

func (r *DiagnosticReport) addAuthCheck() {
	if strings.TrimSpace(os.Getenv(geminiAPIKeyEnv)) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s is required", geminiAPIKeyEnv),
			"",
			fmt.Sprintf("Set %s before running Gemini requests", geminiAPIKeyEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "Gemini API key is configured", geminiAPIKeyEnv, "")
}

func (r *DiagnosticReport) addEndpointCheck(options DiagnosticOptions) {
	raw := strings.TrimSpace(os.Getenv(geminiAPIURLEnv))
	expectStream, expectGenerate := geminiDiagnosticEndpointExpectations(options)
	if raw == "" {
		r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s uses the built-in endpoint", geminiAPIURLEnv), geminiDiagnosticEndpointDetail(r.Model, expectStream, expectGenerate), "")
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			fmt.Sprintf("%s is not a valid absolute URL", geminiAPIURLEnv),
			raw,
			fmt.Sprintf("Set %s to a full Gemini endpoint such as %s", geminiAPIURLEnv, fmt.Sprintf(defaultGeminiURLTemplate, r.Model)),
		)
		return
	}

	var missing []string
	if expectStream {
		if !strings.Contains(parsed.Path, ":streamGenerateContent") {
			missing = append(missing, "streamGenerateContent?alt=sse for text/tool/image")
		} else if parsed.Query().Get("alt") != "sse" {
			missing = append(missing, "alt=sse for streamGenerateContent text/tool/image")
		}
	}
	if expectGenerate && !strings.Contains(parsed.Path, ":generateContent") {
		missing = append(missing, "generateContent for native web search")
	}
	if len(missing) > 0 {
		r.addCheck(
			DiagnosticStatusWarn,
			"endpoint",
			fmt.Sprintf("%s may not match the selected Gemini route", geminiAPIURLEnv),
			fmt.Sprintf("%s; missing=%s; %s", raw, strings.Join(missing, ", "), geminiDiagnosticEndpointDetail(r.Model, expectStream, expectGenerate)),
			"This is OK only for an intentional proxy endpoint that accepts the selected Gemini request shape",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s is configured", geminiAPIURLEnv), geminiDiagnosticEndpointDetail(r.Model, expectStream, expectGenerate), "")
}

func geminiDiagnosticEndpointExpectations(options DiagnosticOptions) (bool, bool) {
	if options.RunSmoke || options.PrintRequest {
		var expectStream, expectGenerate bool
		for _, request := range geminiDiagnosticRequests(options) {
			if request.WebSearchPayload {
				expectGenerate = true
			} else {
				expectStream = true
			}
		}
		if expectStream || expectGenerate {
			return expectStream, expectGenerate
		}
	}
	return true, false
}

func geminiDiagnosticEndpointDetail(model string, expectStream, expectGenerate bool) string {
	var parts []string
	if expectStream {
		parts = append(parts, "stream_url="+getGeminiURL(model))
	}
	if expectGenerate {
		parts = append(parts, "generate_url="+getGeminiFunctionCallingURL(model))
	}
	return strings.Join(parts, ", ")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("gemini") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "gemini provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "gemini provider is not registered", "", "Ensure providers/all imports the Gemini provider")
}

func (r *DiagnosticReport) addModelCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "Gemini model is not resolved", "", "Pass --model or set provider_models.gemini.default_model")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"model",
		"Gemini request model is resolved",
		fmt.Sprintf("%s (%s)", r.Model, r.ModelSource),
		"",
	)
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "catalog_model", "Gemini catalog model is not resolved", r.Model, "Use --catalog-model when the request model is an alias")
		return
	}
	if geminiCatalogModelKnown(r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusOK,
			"catalog_model",
			"Gemini catalog model is resolved",
			fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
			"",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusWarn,
		"catalog_model",
		"Gemini catalog model is not known to local Gemini metadata",
		fmt.Sprintf("model=%s catalog_model=%s (%s)", r.Model, r.CatalogModel, r.CatalogModelSource),
		"Set --catalog-model or provider_models.gemini.catalog_model to a Gemini model before relying on token-limit diagnostics",
	)
}

func (r *DiagnosticReport) addModelLifecycleCheck() {
	warnings := r.modelLifecycleWarnings()
	if len(warnings) == 0 {
		return
	}

	r.addCheck(
		DiagnosticStatusWarn,
		"model_lifecycle",
		geminiModelLifecycleMessage(warnings),
		geminiModelLifecycleDetail(warnings),
		geminiModelLifecycleSuggestion(warnings),
	)
}

type geminiModelLifecycleWarning struct {
	Label     string
	Model     string
	Lifecycle llmcatalog.ModelLifecycle
}

func (r DiagnosticReport) modelLifecycleWarnings() []geminiModelLifecycleWarning {
	var warnings []geminiModelLifecycleWarning
	addLifecycleWarning := func(label, model string) {
		model = strings.TrimSpace(model)
		if model == "" {
			return
		}
		lifecycle, ok := llmcatalog.ModelLifecycleForProvider("gemini", model)
		if !ok || !lifecycle.ShouldWarn() {
			return
		}
		warnings = append(warnings, geminiModelLifecycleWarning{
			Label:     label,
			Model:     model,
			Lifecycle: lifecycle,
		})
	}

	requestModel := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	addLifecycleWarning("request_model", requestModel)
	if !strings.EqualFold(catalogModel, requestModel) {
		addLifecycleWarning("catalog_model", catalogModel)
	}
	return warnings
}

func geminiModelLifecycleMessage(warnings []geminiModelLifecycleWarning) string {
	subject := "Gemini model"
	if len(warnings) == 1 {
		switch warnings[0].Label {
		case "request_model":
			subject = "Gemini request model"
		case "catalog_model":
			subject = "Gemini catalog model"
		}
	}

	for _, warning := range warnings {
		if warning.Lifecycle.Stage == llmcatalog.ModelLifecycleShutdown {
			return subject + " has been shut down"
		}
	}
	for _, warning := range warnings {
		if warning.Lifecycle.Stage == llmcatalog.ModelLifecycleDeprecated {
			return subject + " is deprecated or near shutdown"
		}
	}
	return subject + " is not recommended for new configurations"
}

func geminiModelLifecycleDetail(warnings []geminiModelLifecycleWarning) string {
	parts := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		parts = append(parts, fmt.Sprintf("%s{%s}", warning.Label, warning.Lifecycle.DiagnosticDetail(warning.Model)))
	}
	return strings.Join(parts, "; ")
}

func geminiModelLifecycleSuggestion(warnings []geminiModelLifecycleWarning) string {
	seen := make(map[string]bool)
	suggestions := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		suggestion := strings.TrimSpace(warning.Lifecycle.DiagnosticSuggestion())
		if suggestion == "" || seen[suggestion] {
			continue
		}
		seen[suggestion] = true
		suggestions = append(suggestions, suggestion)
	}
	return strings.Join(suggestions, "; ")
}

func (r *DiagnosticReport) addRouteCheck() {
	if r.Route != DiagnosticRouteStreamGenerateContentSSE {
		r.addCheck(DiagnosticStatusFail, "route", "Gemini route could not be resolved", r.RouteReason, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "route", "Gemini streamGenerateContent SSE route is selected", r.routeCheckDetail(), "")
}

func (r DiagnosticReport) routeCheckDetail() string {
	if strings.TrimSpace(r.RouteReason) == "" {
		return r.Route
	}
	return fmt.Sprintf("%s; %s", r.Route, r.RouteReason)
}

func (r *DiagnosticReport) addServiceTierCheck(cfg *config.Config) {
	var usage *api.Usage
	if r.Smoke != nil && strings.TrimSpace(r.Smoke.Usage.BillingServiceTier) != "" {
		usage = &api.Usage{BillingServiceTier: r.Smoke.Usage.BillingServiceTier}
	}
	r.ServiceTier = providerdiag.NewGeminiServiceTierSnapshot(cfg, usage)
	message := "Gemini service tier request and pricing policy are resolved"
	if strings.TrimSpace(r.ServiceTier.BillingTier) != "" {
		message = "Gemini service tier request, billing, and pricing policy are resolved"
	}
	r.setCheck(DiagnosticStatusOK, "service_tier", message, r.ServiceTier.Detail(), "")
}

func (r *DiagnosticReport) addCatalogPolicyCheck(cfg *config.Config) {
	model := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if model == "" || catalogModel == "" {
		return
	}
	if !geminiCatalogModelKnown(catalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not a Gemini model known to local metadata",
			fmt.Sprintf("catalog_model=%s, context_window=unknown, max_output_tokens=unknown, pricing=unavailable", catalogModel),
			"Use a Gemini catalog model before relying on token-limit diagnostics or cost estimates",
		)
		return
	}

	policy := providerdiag.GeminiCatalogPolicy(cfg, model, catalogModel)
	detail := policy.GeminiDetail()

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a Gemini model known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use a Gemini model known to XELYON, or set max_output_tokens explicitly for this model")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing pricing metadata", detail, "Use a Gemini model with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	policy := llmcatalog.NewGeminiFunctionCallingPolicy(r.Model, r.CatalogModel)
	support := policy.Support()
	detail := geminiDiagnosticFunctionCallingDetail(policy)
	if support.Known && !support.Supported {
		r.addCheck(
			DiagnosticStatusFail,
			"function_calling",
			"Gemini model does not support function calling",
			detail,
			geminiDiagnosticFunctionCallingSuggestion(support),
		)
		return
	}
	if !support.Known {
		r.addCheck(
			DiagnosticStatusWarn,
			"function_calling",
			"Gemini function calling support is unknown for this catalog_model",
			detail,
			"Set --catalog-model or provider_models.gemini.catalog_model to a Gemini model with function calling support, then verify with --tool-smoke",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"function_calling",
		"Gemini function calling payloads are enabled",
		detail,
		"Use request-scoped tool disable only for internal text-only paths",
	)
}

func geminiDiagnosticFunctionCallingDetail(policy llmcatalog.GeminiFunctionCallingPolicy) string {
	support := policy.Support()
	parts := []string{
		"request_model=" + policy.RequestModel(),
		"catalog_model=" + policy.CatalogModel(),
		"policy_model=" + policy.PolicyModel(),
		fmt.Sprintf("known=%t", support.Known),
		fmt.Sprintf("supported=%t", support.Supported),
	}
	if reason := strings.TrimSpace(support.Reason); reason != "" {
		parts = append(parts, "reason="+reason)
	}
	if replacement := strings.TrimSpace(support.Replacement); replacement != "" {
		parts = append(parts, "replacement="+replacement)
	}
	return strings.Join(parts, ", ")
}

func geminiDiagnosticFunctionCallingSuggestion(support llmcatalog.ModelCapabilitySupport) string {
	if replacement := strings.TrimSpace(support.Replacement); replacement != "" {
		return "Use " + replacement + " for Gemini function calling"
	}
	return "Use a Gemini model with function calling support, such as gemini-3.5-flash"
}

func (r *DiagnosticReport) addImageInputCheck() {
	if r.ImageInputSupported {
		r.addCheck(DiagnosticStatusOK, "image_input", "Gemini native provider supports base64 image input", "inline_data image/png", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "image_input", "Gemini image input is not supported", "", "")
}

func (r *DiagnosticReport) addThinkingCheck() {
	status := DiagnosticStatusOK
	message := "Gemini thinking request config uses model defaults"
	detail := fmt.Sprintf("thinking_enabled=%t", r.ThinkingEnabled)
	if r.ThinkingEnabled {
		message = "Gemini thinking request config is enabled"
	}
	r.addCheck(status, "thinking", message, detail, "")
}

func (r *DiagnosticReport) addContextCachingCheck() {
	if r.ContextCachingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"context_caching",
			"Gemini context caching is enabled when requests exceed the cache threshold",
			"",
			fmt.Sprintf("Set %s=0 only for cache troubleshooting", geminiContextCachingEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "context_caching", "Gemini context caching is disabled", fmt.Sprintf("%s=0", geminiContextCachingEnv), "")
}

func (r *DiagnosticReport) addWebSearchCheck() {
	if r.WebSearchSupported {
		r.addCheck(DiagnosticStatusOK, "web_search", "Gemini native web search route is registered", DiagnosticRouteGenerateContent, "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "web_search", "Gemini native web search route is not available", "", "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live Gemini smoke was skipped because prerequisite checks failed",
			"",
			"Fix failed checks, then rerun with --smoke",
		)
		return
	}
	smoke, err := runGeminiDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		failure := classifyGeminiDiagnosticSmokeFailure(smoke, err)
		r.addGeminiDiagnosticSmokeFailureChecks(smoke, failure)
		r.addCheck(DiagnosticStatusFail, "smoke", failure.Message, failure.Detail, failure.Suggestion)
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live Gemini smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "Gemini endpoint accepted a diagnostic tool payload", smoke.Duration, "")
	}
	if smoke.ImagePayload {
		r.addCheck(DiagnosticStatusOK, "image_smoke", "Gemini endpoint accepted a base64 image payload", smoke.Duration, "")
	}
	if smoke.WebSearchPayload {
		r.addCheck(DiagnosticStatusOK, "web_search_smoke", "Gemini native web search returned summary or sources", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "Gemini smoke usage was observed", geminiDiagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"Gemini smoke succeeded but token usage was not observed",
			"",
			"Check whether the endpoint returns usageMetadata in Gemini responses",
		)
	}

	switch {
	case !smoke.UsageObserved:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Gemini smoke cost was not estimated because usage was not observed",
			"",
			"Check usage metadata before relying on cost estimates",
		)
	case smoke.Cost.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Gemini smoke usage was observed but pricing metadata is unavailable",
			geminiDiagnosticSmokeUsageDetail(smoke.Usage),
			"Set provider_models.gemini.catalog_model to a Gemini model with pricing metadata",
		)
	default:
		r.addCheck(DiagnosticStatusOK, "cost", "Gemini smoke cost estimate is available", fmt.Sprintf("$%.6f USD", smoke.Cost.USD), "")
	}
}

func geminiDiagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	detail := fmt.Sprintf(
		"input=%d cached_input=%d output=%d thinking=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
	if tier := strings.TrimSpace(usage.BillingServiceTier); tier != "" {
		detail += fmt.Sprintf(" billing_tier=%s", tier)
	}
	return detail
}
