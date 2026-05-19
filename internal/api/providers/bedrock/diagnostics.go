package bedrock

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は Bedrock 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

// DiagnosticCheck は Bedrock 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeUsage は Bedrock smoke request で観測した usage を表す。
type DiagnosticSmokeUsage = providerdiag.SmokeUsage

// DiagnosticSmokeCost は Bedrock smoke request の cost estimate を表す。
type DiagnosticSmokeCost = providerdiag.SmokeCost

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult = providerdiag.InvocationSmokeRequestResult

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult = providerdiag.InvocationSmokeResult

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview struct {
	Requests []DiagnosticRequestPreviewRequest `json:"requests"`
}

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest = providerdiag.InvocationRequestPreviewRequest

// DiagnosticReport は Bedrock の設定診断結果を表す。
type DiagnosticReport struct {
	Provider               string                    `json:"provider"`
	Region                 string                    `json:"region"`
	Model                  string                    `json:"model"`
	ModelSource            string                    `json:"model_source"`
	CatalogModel           string                    `json:"catalog_model"`
	CatalogModelSource     string                    `json:"catalog_model_source"`
	Route                  string                    `json:"route"`
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

// DiagnosticOptions は Bedrock 診断の入力を表す。
type DiagnosticOptions struct {
	Config          *config.Config
	Model           string
	CatalogModel    string
	RunSmoke        bool
	TextSmoke       bool
	ToolSmoke       bool
	ImageSmoke      bool
	ThinkingSmoke   bool
	PrintRequest    bool
	SmokeTimeout    time.Duration
	MaxOutputTokens int
	SmokeOutput     io.Writer

	invokeClient     invokeModelWithResponseStreamClient
	converseClient   converseStreamClient
	skipAWSAuthCheck bool
}

func (o DiagnosticOptions) requiresAWSAuthCheck() bool {
	return !o.PrintRequest && !o.skipAWSAuthCheck && o.invokeClient == nil && o.converseClient == nil
}

// Diagnose は Bedrock のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveBedrockDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveBedrockDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	route := resolveBedrockRoute(model, catalogModel)
	requestPlan := buildBedrockDiagnosticRequestPlan(options)
	awsCfg, awsLoadErr := loadBedrockAWSConfig(ctx)
	region := awsCfg.Region
	if strings.TrimSpace(region) == "" {
		region = defaultRegion
	}

	report := DiagnosticReport{
		Provider:               "bedrock",
		Region:                 region,
		Model:                  model,
		ModelSource:            modelSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		Route:                  string(route),
		FunctionCallingEnabled: os.Getenv("BEDROCK_FUNCTION_CALLING") != "0",
	}

	report.addAWSConfigChecks(ctx, awsCfg, awsLoadErr, options)
	report.addProviderRegistrationCheck()
	report.addModelConfigCheck()
	report.addRouteCheck(route, requestPlan)
	report.addCatalogPolicyCheck(cfg, route)
	report.addFunctionCallingCheck()

	if options.PrintRequest {
		report.addRequestPreview(ctx, cfg, options, requestPlan)
	}
	if options.RunSmoke && !options.PrintRequest {
		report.runSmokeIfReady(ctx, cfg, options, requestPlan)
	}

	return report
}
