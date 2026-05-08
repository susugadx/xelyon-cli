package kimi

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
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
	InputTokens              int     `json:"input_tokens,omitempty"`
	OutputTokens             int     `json:"output_tokens,omitempty"`
	ThinkingTokens           int     `json:"thinking_tokens,omitempty"`
	CachedInputTokens        int     `json:"cached_input_tokens,omitempty"`
	WebSearchCallCount       int     `json:"web_search_call_count,omitempty"`
	WebSearchCallFeeEstimate float64 `json:"web_search_call_fee_estimate,omitempty"`
	SearchResultTotalTokens  int     `json:"search_result_total_tokens,omitempty"`
}

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult struct {
	Name                     string                     `json:"name"`
	Content                  string                     `json:"content,omitempty"`
	Duration                 string                     `json:"duration,omitempty"`
	UsageObserved            bool                       `json:"usage_observed"`
	Usage                    DiagnosticUsageObservation `json:"usage,omitempty"`
	PromptCacheKeyPresent    bool                       `json:"prompt_cache_key_present"`
	PromptCacheKey           string                     `json:"prompt_cache_key,omitempty"`
	ImagePayload             bool                       `json:"image_payload,omitempty"`
	WebSearchPayload         bool                       `json:"web_search_payload,omitempty"`
	WebSearchCallCount       int                        `json:"web_search_call_count,omitempty"`
	WebSearchCallFeeEstimate float64                    `json:"web_search_call_fee_estimate,omitempty"`
	WebSearchUsageObserved   bool                       `json:"web_search_usage_observed,omitempty"`
	SearchResultTotalTokens  int                        `json:"search_result_total_tokens,omitempty"`
}

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult struct {
	Ran                      bool                           `json:"ran"`
	ToolPayload              bool                           `json:"tool_payload"`
	ImagePayload             bool                           `json:"image_payload"`
	WebSearchPayload         bool                           `json:"web_search_payload"`
	Content                  string                         `json:"content,omitempty"`
	Duration                 string                         `json:"duration,omitempty"`
	UsageObserved            bool                           `json:"usage_observed"`
	CachedInputTokens        int                            `json:"cached_input_tokens"`
	WebSearchCallCount       int                            `json:"web_search_call_count,omitempty"`
	WebSearchCallFeeEstimate float64                        `json:"web_search_call_fee_estimate,omitempty"`
	WebSearchUsageObserved   bool                           `json:"web_search_usage_observed"`
	SearchResultTotalTokens  int                            `json:"search_result_total_tokens,omitempty"`
	Requests                 []DiagnosticSmokeRequestResult `json:"requests,omitempty"`
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
