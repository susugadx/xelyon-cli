package kimi

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は Kimi 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
	DiagnosticStatusInfo DiagnosticStatus = "info"
)

const (
	DiagnosticRouteChatCompletions          = "chat_completions"
	DiagnosticRouteChatCompletionsWebSearch = "chat_completions_web_search"
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
type DiagnosticUsageObservation = providerdiag.KimiSmokeUsageObservation

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult = providerdiag.KimiSmokeRequestResult

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult = providerdiag.KimiSmokeResult

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview = providerdiag.KimiRequestPreview

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest = providerdiag.KimiRequestPreviewRequest

// DiagnosticReport は Kimi の設定診断結果を表す。
type DiagnosticReport struct {
	Provider               string                    `json:"provider"`
	APIURL                 string                    `json:"api_url"`
	Model                  string                    `json:"model"`
	ModelSource            string                    `json:"model_source"`
	CatalogModel           string                    `json:"catalog_model"`
	CatalogModelSource     string                    `json:"catalog_model_source"`
	Route                  string                    `json:"route"`
	RouteReason            string                    `json:"route_reason,omitempty"`
	MaxOutputTokens        int                       `json:"max_output_tokens"`
	ContextWindowTokens    int                       `json:"context_window_tokens,omitempty"`
	FunctionCallingEnabled bool                      `json:"function_calling_enabled"`
	UnsupportedFeatures    []string                  `json:"unsupported_features"`
	PromptCacheKeyPresent  bool                      `json:"prompt_cache_key_present"`
	Checks                 []DiagnosticCheck         `json:"checks"`
	RequestPreview         *DiagnosticRequestPreview `json:"request_preview,omitempty"`
	Smoke                  *DiagnosticSmokeResult    `json:"smoke,omitempty"`
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
	CatalogModel    string
	RunSmoke        bool
	TextSmoke       bool
	ToolSmoke       bool
	ImageSmoke      bool
	WebSearchSmoke  bool
	PrintRequest    bool
	SmokeTimeout    time.Duration
	MaxOutputTokens int
	SmokeOutput     io.Writer
}

func (o DiagnosticOptions) requiresAuthCheck() bool {
	return !o.PrintRequest
}

// Diagnose は Kimi のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveKimiDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveKimiDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	policyCfg := kimiDiagnosticPolicyConfig(cfg, model, catalogModel, 0)
	policy := providerdiag.KimiCatalogPolicy(policyCfg, model, catalogModel)

	report := DiagnosticReport{
		Provider:               "kimi",
		APIURL:                 New(os.Getenv(kimiAPIKeyEnv)).APIURL(),
		Model:                  model,
		ModelSource:            modelSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		Route:                  DiagnosticRouteChatCompletions,
		RouteReason:            "Kimi text, tool, image, and built-in $web_search diagnostics use Moonshot Chat Completions",
		MaxOutputTokens:        policy.MaxOutput.CapabilityTokens(),
		ContextWindowTokens:    policy.ContextWindowTokens,
		FunctionCallingEnabled: os.Getenv(kimiFunctionCallingEnv) != "0",
		UnsupportedFeatures: []string{
			"video input",
			"memory",
			"code runner",
			"file upload",
		},
	}

	report.addAPIURLCheck()
	if options.requiresAuthCheck() {
		report.addAuthCheck()
	}
	report.addProviderRegistrationCheck()
	report.addModelCheck()
	report.addCatalogModelCheck()
	report.addRouteCheck()
	report.addCatalogPolicyCheck(policyCfg)
	report.addFunctionCallingCheck()
	report.addImageInputCheck()
	report.addUnsupportedFeaturesCheck()
	report.addPromptCacheKeyCheck(ctx, policyCfg)
	if options.PrintRequest {
		report.addRequestPreview(ctx, policyCfg, options)
	}

	if options.RunSmoke && !options.PrintRequest {
		report.runSmokeIfReady(ctx, policyCfg, options)
	}

	return report
}
