package claude

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
	if strings.TrimSpace(os.Getenv(anthropicAPIKeyEnv)) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s is required", anthropicAPIKeyEnv),
			"",
			fmt.Sprintf("Set %s before running Claude requests", anthropicAPIKeyEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "Claude API key is configured", anthropicAPIKeyEnv, "")
}

func (r *DiagnosticReport) addEndpointCheck() {
	raw := strings.TrimSpace(os.Getenv(anthropicAPIURLEnv))
	if raw == "" {
		r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s uses the built-in endpoint", anthropicAPIURLEnv), defaultClaudeURL, "")
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			fmt.Sprintf("%s is not a valid absolute URL", anthropicAPIURLEnv),
			raw,
			fmt.Sprintf("Set %s to a full Claude Messages endpoint such as %s", anthropicAPIURLEnv, defaultClaudeURL),
		)
		return
	}

	if strings.Contains(parsed.Host, "anthropic.com") && parsed.Path != "/v1/messages" {
		r.addCheck(
			DiagnosticStatusWarn,
			"endpoint",
			fmt.Sprintf("%s may not match the Claude Messages route", anthropicAPIURLEnv),
			raw,
			fmt.Sprintf("Use %s unless this is an intentional proxy endpoint", defaultClaudeURL),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "endpoint", fmt.Sprintf("%s is configured", anthropicAPIURLEnv), raw, "")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("claude") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "claude provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "claude provider is not registered", "", "Ensure providers/all imports the Claude provider")
}

func (r *DiagnosticReport) addModelCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "Claude model is not resolved", "", "Pass --model or set provider_models.claude.default_model")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"model",
		"Claude request model is resolved",
		fmt.Sprintf("%s (%s)", r.Model, r.ModelSource),
		"",
	)
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "catalog_model", "Claude catalog model is not resolved", r.Model, "Use --catalog-model when the request model is an alias")
		return
	}
	if claudeCatalogModelKnown(r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusOK,
			"catalog_model",
			"Claude catalog model is resolved",
			fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
			"",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusWarn,
		"catalog_model",
		"Claude catalog model is not known to local Claude metadata",
		fmt.Sprintf("model=%s catalog_model=%s (%s)", r.Model, r.CatalogModel, r.CatalogModelSource),
		"Set --catalog-model or provider_models.claude.catalog_model to a Claude model before relying on token-limit diagnostics",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	if r.Route != DiagnosticRouteClaudeMessages {
		r.addCheck(DiagnosticStatusFail, "route", "Claude route could not be resolved", r.RouteReason, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "route", "Claude Messages route is selected", r.routeCheckDetail(), "")
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
	if !claudeCatalogModelKnown(catalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is not a Claude model known to local metadata",
			fmt.Sprintf("catalog_model=%s, context_window=unknown, max_output_tokens=unknown, pricing=unavailable", catalogModel),
			"Use a Claude catalog model before relying on token-limit diagnostics or cost estimates",
		)
		return
	}

	policy := providerdiag.ClaudeCatalogPolicy(cfg, model, catalogModel)
	detail := policy.ClaudeDetail()

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a Claude model known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use a Claude model known to XELYON, or set max_output_tokens explicitly for this model")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing pricing metadata", detail, "Use a Claude model with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(DiagnosticStatusOK, "function_calling", "Claude tool payloads are enabled", "", fmt.Sprintf("Set %s=0 only for tool-call troubleshooting", claudeFunctionCallEnv))
		return
	}
	r.addCheck(DiagnosticStatusWarn, "function_calling", "Claude tool payloads are disabled", fmt.Sprintf("%s=0", claudeFunctionCallEnv), "Unset CLAUDE_FUNCTION_CALLING before relying on tool smoke")
}

func (r *DiagnosticReport) addImageInputCheck() {
	if r.ImageInputSupported {
		r.addCheck(DiagnosticStatusOK, "image_input", "Claude native provider supports base64 image input", "image block media_type=image/png", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "image_input", "Claude image input is not supported", "", "")
}

func (r *DiagnosticReport) addThinkingCheck() {
	detail := fmt.Sprintf("thinking_enabled=%t", r.ThinkingEnabled)
	if r.ThinkingType != "" {
		detail += ", thinking_type=" + r.ThinkingType
	}
	if r.ThinkingEnabled {
		r.addCheck(DiagnosticStatusOK, "thinking", "Claude thinking request config is enabled", detail, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "thinking", "Claude thinking request config is disabled", detail, "")
}

func (r *DiagnosticReport) addContextManagementCheck() {
	detail := fmt.Sprintf("context_management_enabled=%t, claude_compaction_supported=%t", r.ContextManagementEnabled, r.ClaudeCompactionSupported)
	if r.ContextManagementEnabled {
		r.addCheck(DiagnosticStatusOK, "context_management", "Claude context_management request config is enabled", detail, "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "context_management", "Claude context_management request config is disabled", detail, "")
}

func (r *DiagnosticReport) addWebSearchCheck() {
	if r.WebSearchSupported {
		r.addCheck(DiagnosticStatusOK, "web_search", "Claude native web search route is registered", DiagnosticRouteClaudeWebSearch, "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "web_search", "Claude native web search route is not available", "", "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live Claude smoke was skipped because prerequisite checks failed",
			"",
			"Fix failed checks, then rerun with --smoke",
		)
		return
	}
	smoke, err := runClaudeDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		r.addFailedSpecializedSmokeChecks(smoke)
		r.addCheck(DiagnosticStatusFail, "smoke", "live Claude smoke request failed", err.Error(), "Inspect smoke.requests[].error and rerun with --print-request")
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live Claude smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	r.addSpecializedSmokeChecks(smoke)
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "Claude smoke usage was observed", claudeDiagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"Claude smoke succeeded but token usage was not observed",
			"",
			"Check whether the endpoint returns Anthropic usage events",
		)
	}

	switch {
	case !smoke.UsageObserved:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Claude smoke cost was not estimated because usage was not observed",
			"",
			"Check usage metadata before relying on cost estimates",
		)
	case smoke.Cost.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Claude smoke usage was observed but pricing metadata is unavailable",
			claudeDiagnosticSmokeUsageDetail(smoke.Usage),
			"Set provider_models.claude.catalog_model to a Claude model with pricing metadata",
		)
	default:
		r.addCheck(DiagnosticStatusOK, "cost", "Claude smoke cost estimate is available", fmt.Sprintf("$%.6f USD", smoke.Cost.USD), "")
	}
}

func (r *DiagnosticReport) addSpecializedSmokeChecks(smoke DiagnosticSmokeResult) {
	for _, request := range smoke.Requests {
		if request.Skipped && request.ToolPayload {
			r.addCheck(DiagnosticStatusWarn, "tool_smoke", "Claude tool smoke was skipped", request.SkipReason, "Unset CLAUDE_FUNCTION_CALLING before rerunning --tool-smoke")
			continue
		}
	}
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "Claude endpoint accepted a diagnostic tool payload", smoke.Duration, "")
	}
	if smoke.ImagePayload {
		r.addCheck(DiagnosticStatusOK, "image_smoke", "Claude endpoint accepted a base64 image payload", smoke.Duration, "")
	}
	if smoke.ThinkingPayload {
		r.addCheck(DiagnosticStatusOK, "thinking_smoke", "Claude endpoint accepted a thinking request payload", smoke.Duration, "")
	}
	if smoke.WebSearchPayload {
		r.addCheck(DiagnosticStatusOK, "web_search_smoke", "Claude native web search returned summary or sources", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addFailedSpecializedSmokeChecks(smoke DiagnosticSmokeResult) {
	for _, request := range smoke.Requests {
		if strings.TrimSpace(request.Error) == "" {
			continue
		}
		switch {
		case request.ToolPayload:
			r.addCheck(DiagnosticStatusFail, "tool_smoke", "Claude tool smoke response did not include the diagnostic tool call", request.Error, "")
		case request.ImagePayload:
			r.addCheck(DiagnosticStatusFail, "image_smoke", "Claude image smoke failed before proving image input", request.Error, "")
		case request.ThinkingPayload:
			r.addCheck(DiagnosticStatusFail, "thinking_smoke", "Claude thinking smoke failed before proving thinking request support", request.Error, "")
		case request.WebSearchPayload:
			r.addCheck(DiagnosticStatusFail, "web_search_smoke", "Claude web search smoke failed before proving native web search", request.Error, "")
		}
	}
}

func claudeDiagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	return fmt.Sprintf(
		"input=%d cached_input=%d output=%d thinking=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}
