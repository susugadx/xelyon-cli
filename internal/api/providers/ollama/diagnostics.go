package ollama

import (
	"context"
	"io"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は Ollama 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

const DiagnosticRouteOllamaChat = "ollama_chat"

// DiagnosticCheck は Ollama 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeUsage は Ollama smoke request で観測した usage を表す。
type DiagnosticSmokeUsage = providerdiag.SmokeUsage

// DiagnosticSmokeCost は Ollama smoke request の cost estimate を表す。
type DiagnosticSmokeCost = providerdiag.SmokeCost

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult = providerdiag.TextToolSmokeRequestResult

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult = providerdiag.TextToolSmokeResult

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview struct {
	Requests []DiagnosticRequestPreviewRequest `json:"requests"`
}

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest = providerdiag.TextToolRequestPreviewRequest

// DiagnosticReport は Ollama の設定診断結果を表す。
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
	Capabilities           *DiagnosticCapabilities   `json:"capabilities,omitempty"`
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

// DiagnosticOptions は Ollama 診断の入力を表す。
type DiagnosticOptions struct {
	Config               *config.Config
	Model                string
	CatalogModel         string
	RunSmoke             bool
	TextSmoke            bool
	ToolSmoke            bool
	Capabilities         bool
	PrintRequest         bool
	RequiredCapabilities []string
	SmokeTimeout         time.Duration
	MaxOutputTokens      int
	SmokeOutput          io.Writer
}

func (o DiagnosticOptions) requiresEndpointCheck() bool {
	return o.requiresInstalledModelLookup() || o.localCapabilityRequest().RequiresExternalSetupCheck()
}

func (o DiagnosticOptions) localCapabilityRequest() providerdiag.LocalCapabilityRequest {
	return providerdiag.LocalCapabilityRequest{
		Capabilities:         o.Capabilities,
		RequiredCapabilities: o.RequiredCapabilities,
		RunSmoke:             o.RunSmoke,
		PrintRequest:         o.PrintRequest,
	}
}

func (o DiagnosticOptions) requiresInstalledModelLookup() bool {
	return providerdiag.HasRequiredCapability(o.RequiredCapabilities, providerdiag.RequiredCapabilityLocalModelAvailable)
}

// Diagnose は Ollama のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveOllamaDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveOllamaDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	policyCfg := ollamaDiagnosticPolicyConfig(cfg, model, catalogModel, 0)
	policy := providerdiag.OllamaCatalogPolicy(policyCfg, model, catalogModel)
	baseURL := resolveOllamaDiagnosticBaseURL()

	report := DiagnosticReport{
		Provider:               "ollama",
		APIURL:                 baseURL,
		Model:                  model,
		ModelSource:            modelSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		Route:                  DiagnosticRouteOllamaChat,
		RouteReason:            "Ollama provider uses the local /api/chat JSONL stream endpoint",
		MaxOutputTokens:        providerdiag.RuntimeMaxOutputTokens(policyCfg, "ollama", model),
		ContextWindowTokens:    policy.ContextWindowTokens,
		FunctionCallingEnabled: New(baseURL).IsFunctionCallingEnabled(),
	}

	report.addAuthCheck()
	endpointOK := false
	installedModels := []string(nil)
	installedModelLookupOK := false
	if options.requiresEndpointCheck() {
		previewOnly := options.PrintRequest && !options.requiresInstalledModelLookup()
		endpointOK, installedModels = report.addEndpointCheck(previewOnly)
		installedModelLookupOK = endpointOK && !previewOnly
		report.addInstalledModelCheck(endpointOK, installedModels, previewOnly)
	}
	report.addProviderRegistrationCheck()
	report.addModelCheck()
	report.addCatalogModelCheck()
	report.addRouteCheck()
	report.addCatalogPolicyCheck(policyCfg)
	report.addFunctionCallingCheck()
	localModelAvailable := providerdiag.UnknownCapabilityAvailability()
	if installedModelLookupOK {
		localModelAvailable = providerdiag.KnownCapabilityAvailability(ollamaInstalledModelMatches(report.Model, installedModels))
	}
	if options.Capabilities {
		report.addCapabilities(policyCfg, localModelAvailable)
	}
	report.addRequiredCapabilities(policyCfg, options.RequiredCapabilities, localModelAvailable)
	if options.PrintRequest {
		report.addRequestPreview(ctx, policyCfg, options)
	}
	if options.RunSmoke && !options.PrintRequest {
		report.runSmokeIfReady(ctx, policyCfg, options)
	}

	return report
}
