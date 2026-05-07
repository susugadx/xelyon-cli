package kimi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	defaultKimiDiagnosticSmokeTimeout         = 120 * time.Second
	defaultKimiDiagnosticSmokeMaxOutputTokens = 64
)

// DiagnosticStatus は Kimi 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
	DiagnosticStatusInfo DiagnosticStatus = "info"
)

// DiagnosticCheck は Kimi 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticUsageObservation は smoke request で観測した usage を表す。
type DiagnosticUsageObservation struct {
	InputTokens       int `json:"input_tokens,omitempty"`
	OutputTokens      int `json:"output_tokens,omitempty"`
	ThinkingTokens    int `json:"thinking_tokens,omitempty"`
	CachedInputTokens int `json:"cached_input_tokens,omitempty"`
}

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult struct {
	Name                  string                     `json:"name"`
	Content               string                     `json:"content,omitempty"`
	Duration              string                     `json:"duration,omitempty"`
	UsageObserved         bool                       `json:"usage_observed"`
	Usage                 DiagnosticUsageObservation `json:"usage,omitempty"`
	PromptCacheKeyPresent bool                       `json:"prompt_cache_key_present"`
	PromptCacheKey        string                     `json:"prompt_cache_key,omitempty"`
	ImagePayload          bool                       `json:"image_payload,omitempty"`
	WebSearchPayload      bool                       `json:"web_search_payload,omitempty"`
}

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult struct {
	Ran               bool                           `json:"ran"`
	ToolPayload       bool                           `json:"tool_payload"`
	ImagePayload      bool                           `json:"image_payload"`
	WebSearchPayload  bool                           `json:"web_search_payload"`
	Content           string                         `json:"content,omitempty"`
	Duration          string                         `json:"duration,omitempty"`
	UsageObserved     bool                           `json:"usage_observed"`
	CachedInputTokens int                            `json:"cached_input_tokens"`
	Requests          []DiagnosticSmokeRequestResult `json:"requests,omitempty"`
}

// DiagnosticReport は Kimi の設定診断結果を表す。
type DiagnosticReport struct {
	Provider               string                 `json:"provider"`
	APIURL                 string                 `json:"api_url"`
	Model                  string                 `json:"model"`
	ModelSource            string                 `json:"model_source"`
	CatalogModel           string                 `json:"catalog_model"`
	CatalogModelSource     string                 `json:"catalog_model_source"`
	MaxOutputTokens        int                    `json:"max_output_tokens"`
	ContextWindowTokens    int                    `json:"context_window_tokens,omitempty"`
	FunctionCallingEnabled bool                   `json:"function_calling_enabled"`
	UnsupportedFeatures    []string               `json:"unsupported_features"`
	PromptCacheKeyPresent  bool                   `json:"prompt_cache_key_present"`
	Checks                 []DiagnosticCheck      `json:"checks"`
	Smoke                  *DiagnosticSmokeResult `json:"smoke,omitempty"`
}

// HasFailures は診断に fail 項目が含まれるか返す。
func (r DiagnosticReport) HasFailures() bool {
	for _, check := range r.Checks {
		if check.Status == DiagnosticStatusFail {
			return true
		}
	}
	return false
}

// SummaryStatus はレポート全体の代表 status を返す。
func (r DiagnosticReport) SummaryStatus() DiagnosticStatus {
	if r.HasFailures() {
		return DiagnosticStatusFail
	}
	for _, check := range r.Checks {
		if check.Status == DiagnosticStatusWarn {
			return DiagnosticStatusWarn
		}
	}
	return DiagnosticStatusOK
}

// DiagnosticOptions は Kimi 診断の入力を表す。
type DiagnosticOptions struct {
	Config          *config.Config
	Model           string
	RunSmoke        bool
	TextSmoke       bool
	ToolSmoke       bool
	ImageSmoke      bool
	WebSearchSmoke  bool
	SmokeTimeout    time.Duration
	MaxOutputTokens int
	SmokeOutput     io.Writer
}

// Diagnose は Kimi のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveKimiDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveKimiDiagnosticCatalogModel(cfg, model)
	contextWindow, _ := llmcatalog.KnownModelContextLimit(catalogModel)
	configCtx := config.WithContext(context.Background(), cfg)

	report := DiagnosticReport{
		Provider:               "kimi",
		APIURL:                 New(os.Getenv(kimiAPIKeyEnv)).APIURL(),
		Model:                  model,
		ModelSource:            modelSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		MaxOutputTokens:        api.GetMaxOutputTokens(configCtx, "kimi", model),
		ContextWindowTokens:    contextWindow,
		FunctionCallingEnabled: os.Getenv("KIMI_FUNCTION_CALLING") != "0",
		UnsupportedFeatures: []string{
			"video input",
			"memory",
			"code runner",
			"file upload",
		},
	}

	report.addAPIURLCheck()
	report.addAuthCheck()
	report.addProviderRegistrationCheck()
	report.addModelConfigCheck()
	report.addFunctionCallingCheck()
	report.addImageInputCheck()
	report.addUnsupportedFeaturesCheck()
	report.addPromptCacheKeyCheck(ctx, cfg)

	if options.RunSmoke {
		report.runSmokeIfReady(ctx, cfg, options)
	}

	return report
}

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
		r.addCheck(DiagnosticStatusOK, "web_search_smoke", "Kimi endpoint completed a built-in $web_search request", smoke.Duration, "Kimi charges a separate web search call fee when $web_search is triggered")
	}
}

func resolveKimiDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	if model := strings.TrimSpace(explicitModel); model != "" {
		return model, "--model"
	}
	if model := strings.TrimSpace(os.Getenv("XELYON_MODEL")); model != "" {
		return model, "XELYON_MODEL"
	}
	if model := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("kimi")); model != "" {
		return model, "provider_models.kimi.default_model"
	}
	if config.SameProviderRuntimeIdentity("kimi", cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider("kimi"))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if model := strings.TrimSpace(cfg.GetSelectedModelForProvider("kimi")); model != "" {
		return model, "built-in provider default"
	}
	return defaultKimiModel, "fallback"
}

func resolveKimiDiagnosticCatalogModel(cfg *config.Config, model string) (string, string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", ""
	}
	resolution := cfg.ResolveModelCatalog("kimi", model)
	if strings.TrimSpace(resolution.Model) == "" {
		return model, "model"
	}
	if resolution.Model != model {
		return resolution.Model, "provider_models.kimi.catalog_model"
	}
	if resolution.ConfiguredWithoutCatalog {
		return resolution.Model, "configured model"
	}
	return resolution.Model, "model"
}

type kimiDiagnosticSmokeRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	Thinking         bool
	SessionID        string
	ToolPayload      bool
	ImagePayload     bool
	WebSearchPayload bool
}

const (
	kimiDiagnosticSmokeCacheFirstName  = "thinking_off_cache_first"
	kimiDiagnosticSmokeCacheSecondName = "thinking_off_cache_second"
	kimiDiagnosticSmokeThinkingName    = "thinking_on"
	kimiDiagnosticSmokeImageName       = "image_smoke"
	kimiDiagnosticSmokeWebSearchName   = "web_search_smoke"
	kimiDiagnosticSmokeToolName        = "tool_smoke"
)

func runKimiDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultKimiDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultKimiDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := config.CloneConfig(cfg)
	catalogModel := report.CatalogModel
	if strings.TrimSpace(catalogModel) == "" {
		catalogModel = report.Model
	}
	smokeCfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    report.Model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
		ModelOverrides: map[string]config.ModelOverride{
			report.Model: {
				CatalogModel:    catalogModel,
				MaxOutputTokens: maxOutputTokens,
			},
		},
	})

	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(kimiAPIKeyEnv))
	result := DiagnosticSmokeResult{Ran: true}
	requests, runTextSmoke := kimiDiagnosticSmokeRequests(options, report.FunctionCallingEnabled)

	for _, request := range requests {
		requestResult, err := runKimiDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report.Model, request, output)
		result.Requests = append(result.Requests, requestResult)
		if requestResult.UsageObserved {
			result.UsageObserved = true
			result.CachedInputTokens += requestResult.Usage.CachedInputTokens
		}
		if requestResult.Content != "" {
			result.Content = requestResult.Content
		}
		if request.ImagePayload {
			result.ImagePayload = true
			if !requestResult.PromptCacheKeyPresent {
				return result, fmt.Errorf("image smoke request did not include prompt_cache_key")
			}
		}
		if request.WebSearchPayload {
			result.WebSearchPayload = true
			if !requestResult.PromptCacheKeyPresent {
				return result, fmt.Errorf("web search smoke request did not include prompt_cache_key")
			}
		}
		if request.ToolPayload && err == nil {
			result.ToolPayload = true
			if !diagnosticSmokeContentHasToolCall(requestResult.Content) {
				return result, fmt.Errorf("tool smoke response did not include %s function_call", diagnosticSmokeToolName)
			}
		}
		if err != nil {
			return result, err
		}
	}

	if runTextSmoke {
		first := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheFirstName)
		second := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheSecondName)
		if first == "" || second == "" || first != second {
			return result, fmt.Errorf("session-aware prompt_cache_key mismatch: first=%q second=%q", first, second)
		}
	}

	return result, nil
}

func kimiDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) ([]kimiDiagnosticSmokeRequest, bool) {
	runTextSmoke := options.TextSmoke || options.ToolSmoke || (!options.ImageSmoke && !options.WebSearchSmoke)
	var requests []kimiDiagnosticSmokeRequest
	if runTextSmoke {
		requests = append(requests, kimiDiagnosticTextSmokeRequests()...)
	}
	if options.ImageSmoke {
		requests = append(requests, kimiDiagnosticImageSmokeRequest())
	}
	if options.WebSearchSmoke {
		requests = append(requests, kimiDiagnosticWebSearchSmokeRequest())
	}
	if options.ToolSmoke && functionCallingEnabled {
		requests = append(requests, kimiDiagnosticToolSmokeRequest())
	}
	return requests, runTextSmoke
}

func kimiDiagnosticTextSmokeRequests() []kimiDiagnosticSmokeRequest {
	return []kimiDiagnosticSmokeRequest{
		{
			Name:         kimiDiagnosticSmokeCacheFirstName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor cache one",
			Thinking:     false,
			SessionID:    "xelyon-kimi-doctor-cache",
		},
		{
			Name:         kimiDiagnosticSmokeCacheSecondName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor cache two",
			Thinking:     false,
			SessionID:    "xelyon-kimi-doctor-cache",
		},
		{
			Name:         kimiDiagnosticSmokeThinkingName,
			SystemPrompt: "Think briefly, then reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor thinking ok",
			Thinking:     true,
			SessionID:    "xelyon-kimi-doctor-thinking",
		},
	}
}

func kimiDiagnosticImageSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:         kimiDiagnosticSmokeImageName,
		SystemPrompt: "Reply briefly.",
		UserContent:  "Look at the attached tiny diagnostic image and reply with a short non-empty response.",
		Thinking:     false,
		SessionID:    "xelyon-kimi-doctor-image",
		ImagePayload: true,
	}
}

func kimiDiagnosticWebSearchSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:             kimiDiagnosticSmokeWebSearchName,
		SystemPrompt:     "Use web search and reply briefly.",
		UserContent:      "Search the web for Moonshot AI Kimi API web search pricing and reply with one short non-empty summary.",
		Thinking:         false,
		SessionID:        "xelyon-kimi-doctor-web-search",
		WebSearchPayload: true,
	}
}

func kimiDiagnosticToolSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:         kimiDiagnosticSmokeToolName,
		SystemPrompt: "Use the diagnostic tool.",
		UserContent:  `Call xelyon_kimi_doctor_probe exactly once with {"value":"kimi-tool-ok"} and do not answer in prose.`,
		Thinking:     false,
		SessionID:    "xelyon-kimi-doctor-tool",
		ToolPayload:  true,
	}
}

func diagnosticSmokePromptCacheKey(requests []DiagnosticSmokeRequestResult, name string) string {
	for _, request := range requests {
		if request.Name == name {
			return request.PromptCacheKey
		}
	}
	return ""
}

func runKimiDiagnosticSmokeRequest(ctx context.Context, cfg *config.Config, provider *Provider, model string, request kimiDiagnosticSmokeRequest, output io.Writer) (DiagnosticSmokeRequestResult, error) {
	requestCfg := config.CloneConfig(cfg)
	requestCfg.Thinking.Enabled = request.Thinking
	requestCtx := newKimiDiagnosticContext(ctx, requestCfg, request.Thinking, output)
	if request.SessionID != "" {
		requestCtx = api.WithPromptCacheScope(requestCtx, api.PromptCacheScope{SessionID: request.SessionID})
	}
	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, diagnosticSmokeToolDefinitions())
		provider.SetToolChoice(diagnosticSmokeToolName)
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		provider.ClearToolChoice()
	}
	provider.SetMCPTools(nil)
	provider.setDiagnosticFunctionCalling(request.ToolPayload)

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.InputTokens += observed.InputTokens
		usage.OutputTokens += observed.OutputTokens
		usage.ThinkingTokens += observed.ThinkingTokens
		usage.CachedInputTokens += observed.CachedInputTokens
		usage.CacheCreationTokens += observed.CacheCreationTokens
		usage.StorageCost += observed.StorageCost
		usageObserved = true
	})

	started := time.Now()
	var promptCacheKey string
	var content string
	var err error
	if request.ImagePayload {
		image := kimiDiagnosticImage()
		built, buildErr := provider.buildImageChatCompletionsRequest(requestCtx, request.SystemPrompt, nil, request.UserContent, image, model)
		if buildErr != nil {
			err = buildErr
		} else {
			promptCacheKey = built.PromptCacheKey
			content, err = provider.ChatWithImage(requestCtx, request.SystemPrompt, nil, request.UserContent, image, model)
		}
	} else if request.WebSearchPayload {
		built := buildKimiWebSearchRequest(requestCtx, initialKimiWebSearchMessages(request.UserContent), model, "kimi")
		promptCacheKey = built.PromptCacheKey
		content, err = provider.webSearch(requestCtx, request.UserContent, model, "kimi")
	} else {
		history := []api.Message{{Role: "user", Content: request.UserContent}}
		built := provider.buildChatCompletionsRequest(
			requestCtx,
			request.SystemPrompt,
			history,
			model,
		)
		promptCacheKey = built.PromptCacheKey
		content, err = provider.ChatWithTools(
			requestCtx,
			request.SystemPrompt,
			history,
			model,
		)
	}
	elapsed := time.Since(started).Round(time.Millisecond)
	if err == nil && !request.ToolPayload && strings.TrimSpace(content) == "" {
		err = fmt.Errorf("%s smoke response content is empty", request.Name)
	}

	return DiagnosticSmokeRequestResult{
		Name:                  request.Name,
		Content:               strings.TrimSpace(content),
		Duration:              elapsed.String(),
		UsageObserved:         usageObserved,
		Usage:                 diagnosticUsageObservation(usage),
		PromptCacheKeyPresent: strings.TrimSpace(promptCacheKey) != "",
		PromptCacheKey:        promptCacheKey,
		ImagePayload:          request.ImagePayload,
		WebSearchPayload:      request.WebSearchPayload,
	}, err
}

func kimiDiagnosticImage() *api.ImageData {
	return &api.ImageData{
		MediaType: "image/png",
		Base64:    kimiDiagnosticPNGBase64,
	}
}

const kimiDiagnosticPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="

func (p *Provider) setDiagnosticFunctionCalling(enabled bool) {
	if p == nil {
		return
	}
	p.functionCalling = &enabled
}

func newKimiDiagnosticContext(ctx context.Context, cfg *config.Config, thinking bool, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCfg := config.CloneConfig(cfg)
	requestCfg.Thinking.Enabled = thinking
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = config.WithContext(requestCtx, requestCfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	return requestCtx
}

func diagnosticUsageObservation(usage api.Usage) DiagnosticUsageObservation {
	return DiagnosticUsageObservation{
		InputTokens:       usage.InputTokens,
		OutputTokens:      usage.OutputTokens,
		ThinkingTokens:    usage.ThinkingTokens,
		CachedInputTokens: usage.CachedInputTokens,
	}
}

const diagnosticSmokeToolName = "xelyon_kimi_doctor_probe"

func diagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        diagnosticSmokeToolName,
		Description: "No-op diagnostic probe used to verify Kimi tool calling.",
		Parameters: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"value"},
		},
	}}
}

func diagnosticSmokeContentHasToolCall(content string) bool {
	return strings.Contains(content, `"tool":"`+diagnosticSmokeToolName+`"`)
}

func isTransientKimiSmokeError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "429") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "api error (5") ||
		strings.Contains(message, "status 5") {
		return true
	}
	return false
}
