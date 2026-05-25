package gemini

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は Gemini 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
	DiagnosticStatusInfo DiagnosticStatus = "info"
)

const (
	DiagnosticRouteStreamGenerateContentSSE = "stream_generate_content_sse"
	DiagnosticRouteGenerateContent          = "generate_content"
	defaultGeminiDiagnosticModel            = "gemini-3.1-pro-preview-customtools"
)

// DiagnosticCheck は Gemini 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeUsage は Gemini smoke request で観測した usage を表す。
type DiagnosticSmokeUsage = providerdiag.SmokeUsage

// DiagnosticSmokeCost は Gemini smoke request の cost estimate を表す。
type DiagnosticSmokeCost = providerdiag.SmokeCost

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult = providerdiag.MultimodalSmokeRequestResult

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult = providerdiag.MultimodalSmokeResult

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview struct {
	Requests []DiagnosticRequestPreviewRequest `json:"requests"`
}

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest = providerdiag.MultimodalRequestPreviewRequest

// DiagnosticReport は Gemini の設定診断結果を表す。
type DiagnosticReport struct {
	Provider               string                                 `json:"provider"`
	APIURL                 string                                 `json:"api_url"`
	Model                  string                                 `json:"model"`
	ModelSource            string                                 `json:"model_source"`
	CatalogModel           string                                 `json:"catalog_model"`
	CatalogModelSource     string                                 `json:"catalog_model_source"`
	Route                  string                                 `json:"route"`
	RouteReason            string                                 `json:"route_reason,omitempty"`
	MaxOutputTokens        int                                    `json:"max_output_tokens"`
	ContextWindowTokens    int                                    `json:"context_window_tokens,omitempty"`
	FunctionCallingEnabled bool                                   `json:"function_calling_enabled"`
	ImageInputSupported    bool                                   `json:"image_input_supported"`
	WebSearchSupported     bool                                   `json:"web_search_supported"`
	ContextCachingEnabled  bool                                   `json:"context_caching_enabled"`
	ThinkingEnabled        bool                                   `json:"thinking_enabled"`
	ServiceTier            providerdiag.GeminiServiceTierSnapshot `json:"service_tier"`
	Checks                 []DiagnosticCheck                      `json:"checks"`
	Capabilities           *DiagnosticCapabilities                `json:"capabilities,omitempty"`
	RequestPreview         *DiagnosticRequestPreview              `json:"request_preview,omitempty"`
	Smoke                  *DiagnosticSmokeResult                 `json:"smoke,omitempty"`
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

// DiagnosticOptions は Gemini 診断の入力を表す。
type DiagnosticOptions struct {
	Config               *config.Config
	Model                string
	CatalogModel         string
	RunSmoke             bool
	TextSmoke            bool
	ToolSmoke            bool
	ImageSmoke           bool
	WebSearchSmoke       bool
	Capabilities         bool
	PrintRequest         bool
	RequiredCapabilities []string
	SmokeTimeout         time.Duration
	MaxOutputTokens      int
	SmokeOutput          io.Writer
}

func (o DiagnosticOptions) requiresAuthCheck() bool {
	return o.localCapabilityRequest().RequiresAuthCheck()
}

func (o DiagnosticOptions) requiresEndpointCheck() bool {
	return o.localCapabilityRequest().RequiresExternalSetupCheck()
}

func (o DiagnosticOptions) localCapabilityRequest() providerdiag.LocalCapabilityRequest {
	return providerdiag.LocalCapabilityRequest{
		Capabilities:         o.Capabilities,
		RequiredCapabilities: o.RequiredCapabilities,
		RunSmoke:             o.RunSmoke,
		PrintRequest:         o.PrintRequest,
	}
}

// Diagnose は Gemini のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveGeminiDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveGeminiDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	policyCfg := geminiDiagnosticPolicyConfig(cfg, model, catalogModel, 0)
	configCtx := config.WithContext(context.Background(), policyCfg)
	policy := providerdiag.GeminiCatalogPolicy(policyCfg, model, catalogModel)
	functionCallingEnabled := llmcatalog.NewGeminiFunctionCallingPolicy(model, catalogModel).Enabled()

	report := DiagnosticReport{
		Provider:               "gemini",
		APIURL:                 getGeminiURL(model),
		Model:                  model,
		ModelSource:            modelSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		Route:                  DiagnosticRouteStreamGenerateContentSSE,
		RouteReason:            "Gemini text, tool, and image requests use streamGenerateContent?alt=sse; native web search uses generateContent",
		MaxOutputTokens:        providerdiag.RuntimeMaxOutputTokens(policyCfg, "gemini", model),
		ContextWindowTokens:    policy.ContextWindowTokens,
		FunctionCallingEnabled: functionCallingEnabled,
		ImageInputSupported:    true,
		WebSearchSupported:     true,
		ContextCachingEnabled:  os.Getenv("GEMINI_CONTEXT_CACHING") != "0",
		ThinkingEnabled:        api.IsThinkingEnabled(configCtx),
		ServiceTier:            providerdiag.NewGeminiServiceTierSnapshot(policyCfg, nil),
	}

	if options.requiresAuthCheck() {
		report.addAuthCheck()
	}
	if options.requiresEndpointCheck() {
		report.addEndpointCheck(options)
	}
	report.addProviderRegistrationCheck()
	report.addModelCheck()
	report.addCatalogModelCheck()
	report.addModelLifecycleCheck()
	report.addRouteCheck()
	report.addServiceTierCheck(policyCfg)
	report.addCatalogPolicyCheck(policyCfg)
	report.addFunctionCallingCheck()
	report.addImageInputCheck()
	report.addThinkingCheck()
	report.addContextCachingCheck()
	report.addWebSearchCheck()
	if options.Capabilities {
		report.addCapabilities(policyCfg)
	}
	report.addRequiredCapabilities(policyCfg, options.RequiredCapabilities)
	if options.PrintRequest {
		report.addRequestPreview(ctx, policyCfg, options)
	}
	if options.RunSmoke && !options.PrintRequest {
		report.runSmokeIfReady(ctx, policyCfg, options)
		report.addServiceTierCheck(policyCfg)
	}

	return report
}

func resolveGeminiDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticModel(cfg, "gemini", explicitModel, defaultGeminiDiagnosticModel)
}

func resolveGeminiDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, "gemini", model, explicitCatalogModel)
}

func geminiDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	return providerdiag.ProviderDiagnosticPolicyConfig(cfg, providerdiag.ProviderDiagnosticPolicyConfigOptions{
		Provider:        "gemini",
		Model:           model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
	})
}

func geminiCatalogModelKnown(model string) bool {
	return providerdiag.IsProviderCatalogModelKnown("gemini", model)
}
