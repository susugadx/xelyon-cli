package groq

import (
	"context"
	"io"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は Groq 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

const DiagnosticRouteChatCompletions = "chat_completions"

// DiagnosticCheck は Groq 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeUsage は Groq smoke request で観測した usage を表す。
type DiagnosticSmokeUsage = providerdiag.SmokeUsage

// DiagnosticSmokeCost は Groq smoke request の cost estimate を表す。
type DiagnosticSmokeCost = providerdiag.SmokeCost

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult struct {
	Name          string               `json:"name"`
	Ran           bool                 `json:"ran"`
	Skipped       bool                 `json:"skipped,omitempty"`
	SkipReason    string               `json:"skip_reason,omitempty"`
	ToolPayload   bool                 `json:"tool_payload"`
	Route         string               `json:"route"`
	Content       string               `json:"content,omitempty"`
	Duration      string               `json:"duration,omitempty"`
	UsageObserved bool                 `json:"usage_observed"`
	Usage         DiagnosticSmokeUsage `json:"usage"`
	Cost          DiagnosticSmokeCost  `json:"cost"`
	Error         string               `json:"error,omitempty"`
}

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult struct {
	Ran           bool                           `json:"ran"`
	ToolPayload   bool                           `json:"tool_payload"`
	Route         string                         `json:"route"`
	Content       string                         `json:"content,omitempty"`
	Duration      string                         `json:"duration,omitempty"`
	UsageObserved bool                           `json:"usage_observed"`
	Usage         DiagnosticSmokeUsage           `json:"usage"`
	Cost          DiagnosticSmokeCost            `json:"cost"`
	Requests      []DiagnosticSmokeRequestResult `json:"requests,omitempty"`
}

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview struct {
	Requests []DiagnosticRequestPreviewRequest `json:"requests"`
}

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest struct {
	Name        string            `json:"name"`
	Skipped     bool              `json:"skipped,omitempty"`
	SkipReason  string            `json:"skip_reason,omitempty"`
	ToolPayload bool              `json:"tool_payload"`
	Route       string            `json:"route"`
	Method      string            `json:"method,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        any               `json:"body,omitempty"`
}

// DiagnosticReport は Groq の設定診断結果を表す。
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

// DiagnosticOptions は Groq 診断の入力を表す。
type DiagnosticOptions struct {
	Config          *config.Config
	Model           string
	CatalogModel    string
	RunSmoke        bool
	TextSmoke       bool
	ToolSmoke       bool
	PrintRequest    bool
	SmokeTimeout    time.Duration
	MaxOutputTokens int
	SmokeOutput     io.Writer
}

func (o DiagnosticOptions) requiresAuthCheck() bool {
	return !o.PrintRequest
}

// Diagnose は Groq のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveGroqDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveGroqDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	policyCfg := groqDiagnosticPolicyConfig(cfg, model, catalogModel, 0)
	contextWindow := 0
	if groqCatalogModelKnown(catalogModel) {
		contextWindow, _ = llmcatalog.KnownModelContextLimit(catalogModel)
	}
	configCtx := config.WithContext(context.Background(), policyCfg)

	report := DiagnosticReport{
		Provider:               "groq",
		APIURL:                 New("diagnostic-key").APIURL(),
		Model:                  model,
		ModelSource:            modelSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		Route:                  DiagnosticRouteChatCompletions,
		RouteReason:            "Groq provider uses OpenAI-compatible Chat Completions",
		MaxOutputTokens:        api.GetMaxOutputTokens(configCtx, "groq", model),
		ContextWindowTokens:    contextWindow,
		FunctionCallingEnabled: New("diagnostic-key").IsFunctionCallingEnabled(),
	}

	if options.requiresAuthCheck() {
		report.addAuthCheck()
	}
	report.addEndpointCheck()
	report.addProviderRegistrationCheck()
	report.addModelCheck()
	report.addCatalogModelCheck()
	report.addRouteCheck()
	report.addCatalogPolicyCheck(policyCfg)
	report.addFunctionCallingCheck()
	if options.PrintRequest {
		report.addRequestPreview(ctx, policyCfg, options)
	}
	if options.RunSmoke && !options.PrintRequest {
		report.runSmokeIfReady(ctx, policyCfg, options)
	}

	return report
}
