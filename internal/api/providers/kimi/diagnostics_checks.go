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
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
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

func (r *DiagnosticReport) addModelConfigCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "Kimi model is not resolved", "", "Pass --model kimi-k2.6 or set provider_models.kimi.default_model")
		return
	}
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "Kimi catalog model is not resolved", r.Model, "Use a known Kimi model such as kimi-k2.6")
		return
	}
	if llmcatalog.InferProviderFromModel(r.CatalogModel) != "kimi" {
		r.addCheck(
			DiagnosticStatusWarn,
			"model",
			"resolved catalog model does not look like a native Kimi model",
			fmt.Sprintf("model=%s catalog_model=%s", r.Model, r.CatalogModel),
			"Set provider_models.kimi.catalog_model or model_overrides.<model>.catalog_model to kimi-k2.6 or kimi-k2.5 for custom aliases",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"model",
		"Kimi model config is resolved",
		fmt.Sprintf("%s (%s), catalog_model=%s (%s), max_output_tokens=%d", r.Model, r.ModelSource, r.CatalogModel, r.CatalogModelSource, r.MaxOutputTokens),
		"",
	)
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"Kimi function calling payloads are enabled",
			"",
			"Set KIMI_FUNCTION_CALLING=0 only if the selected endpoint rejects tool payloads",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "Kimi function calling payloads are disabled", "KIMI_FUNCTION_CALLING=0", "")
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
			"KIMI_FUNCTION_CALLING=0",
			"Unset KIMI_FUNCTION_CALLING or set it to 1 before rerunning --tool-smoke",
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
