package openai

import (
	"context"
	"io"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は OpenAI 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

const (
	DiagnosticRouteResponsesStreaming    = "responses_streaming"
	DiagnosticRouteResponsesNonStreaming = "responses_non_streaming"
	DiagnosticRouteChatCompletions       = "chat_completions"
)

// DiagnosticCheck は OpenAI 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeUsage は OpenAI smoke request で観測した usage を表す。
type DiagnosticSmokeUsage = providerdiag.SmokeUsage

// DiagnosticSmokeCost は OpenAI smoke request の cost estimate を表す。
type DiagnosticSmokeCost = providerdiag.SmokeCost

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult = providerdiag.RoutedResponsesSmokeRequestResult

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult = providerdiag.RoutedResponsesSmokeResult

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview struct {
	Requests []DiagnosticRequestPreviewRequest `json:"requests"`
}

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest = providerdiag.RoutedResponsesRequestPreviewRequest

// DiagnosticReport は OpenAI の設定診断結果を表す。
type DiagnosticReport struct {
	Provider                   string                    `json:"provider"`
	APIURL                     string                    `json:"api_url"`
	ResponsesURL               string                    `json:"responses_url"`
	Model                      string                    `json:"model"`
	ModelSource                string                    `json:"model_source"`
	CatalogModel               string                    `json:"catalog_model"`
	CatalogModelSource         string                    `json:"catalog_model_source"`
	Route                      string                    `json:"route"`
	RouteReason                string                    `json:"route_reason,omitempty"`
	MaxOutputTokens            int                       `json:"max_output_tokens"`
	ContextWindowTokens        int                       `json:"context_window_tokens,omitempty"`
	FunctionCallingEnabled     bool                      `json:"function_calling_enabled"`
	ResponsesStore             bool                      `json:"responses_store"`
	ResponsesPersistResponseID bool                      `json:"responses_persist_response_id"`
	Checks                     []DiagnosticCheck         `json:"checks"`
	Capabilities               *DiagnosticCapabilities   `json:"capabilities,omitempty"`
	RequestPreview             *DiagnosticRequestPreview `json:"request_preview,omitempty"`
	Smoke                      *DiagnosticSmokeResult    `json:"smoke,omitempty"`
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

// DiagnosticOptions は OpenAI 診断の入力を表す。
type DiagnosticOptions struct {
	Config               *config.Config
	Model                string
	CatalogModel         string
	RunSmoke             bool
	TextSmoke            bool
	ToolSmoke            bool
	Capabilities         bool
	RetentionSmoke       bool
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

// Diagnose は OpenAI のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveOpenAIDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveOpenAIDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	policyCfg := openAIDiagnosticPolicyConfig(cfg, model, catalogModel)
	configCtx := config.WithContext(context.Background(), policyCfg)
	routeResolution := resolveOpenAIDiagnosticRouteResolution(policyCfg, model, catalogModel)
	policy := providerdiag.OpenAICatalogPolicy(policyCfg, model, catalogModel)

	report := DiagnosticReport{
		Provider:                   "openai",
		APIURL:                     New("diagnostic-key").APIURL,
		ResponsesURL:               resolveResponsesAPIURL(),
		Model:                      model,
		ModelSource:                modelSource,
		CatalogModel:               catalogModel,
		CatalogModelSource:         catalogSource,
		Route:                      routeResolution.Route,
		RouteReason:                routeResolution.ReasonString(),
		MaxOutputTokens:            policy.MaxOutput.CapabilityTokens(),
		ContextWindowTokens:        policy.ContextWindowTokens,
		FunctionCallingEnabled:     New("diagnostic-key").IsFunctionCallingEnabled(),
		ResponsesStore:             cfg.ResponsesStoreEnabled(),
		ResponsesPersistResponseID: cfg.ResponsesPersistResponseIDEnabled(),
	}

	if options.requiresAuthCheck() {
		report.addAuthCheck()
	}
	if options.requiresEndpointCheck() {
		report.addAPIURLCheck()
		report.addResponsesURLCheck()
	}
	report.addProviderRegistrationCheck()
	report.addModelConfigCheck()
	report.addRouteCheck()
	report.addCatalogPolicyCheck(policyCfg)
	report.addFunctionCallingCheck()
	report.addResponsesRetentionCheck()

	if options.Capabilities {
		report.addCapabilities(configCtx, policyCfg)
	}
	report.addRequiredCapabilities(configCtx, policyCfg, options.RequiredCapabilities)
	if options.PrintRequest {
		report.addRequestPreview(ctx, policyCfg, options)
	}
	if options.RunSmoke && !options.PrintRequest {
		report.runSmokeIfReady(ctx, policyCfg, options)
	}

	return report
}
