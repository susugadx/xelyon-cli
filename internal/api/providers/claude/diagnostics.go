package claude

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は Claude 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
	DiagnosticStatusInfo DiagnosticStatus = "info"
)

const (
	DiagnosticRouteClaudeMessages  = "claude_messages"
	DiagnosticRouteClaudeWebSearch = "claude_messages_web_search"
	claudeDiagnosticToolName       = "xelyon_claude_doctor_probe"
)

// DiagnosticCheck は Claude 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeUsage は Claude smoke request で観測した usage を表す。
type DiagnosticSmokeUsage = providerdiag.SmokeUsage

// DiagnosticSmokeCost は Claude smoke request の cost estimate を表す。
type DiagnosticSmokeCost = providerdiag.SmokeCost

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult = providerdiag.MultimodalSmokeRequestResult

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult = providerdiag.ThinkingMultimodalSmokeResult

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview struct {
	Requests []DiagnosticRequestPreviewRequest `json:"requests"`
}

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest = providerdiag.MultimodalRequestPreviewRequest

// DiagnosticReport は Claude の設定診断結果を表す。
type DiagnosticReport struct {
	Provider                  string                    `json:"provider"`
	APIURL                    string                    `json:"api_url"`
	Model                     string                    `json:"model"`
	ModelSource               string                    `json:"model_source"`
	CatalogModel              string                    `json:"catalog_model"`
	CatalogModelSource        string                    `json:"catalog_model_source"`
	Route                     string                    `json:"route"`
	RouteReason               string                    `json:"route_reason,omitempty"`
	MaxOutputTokens           int                       `json:"max_output_tokens"`
	ContextWindowTokens       int                       `json:"context_window_tokens,omitempty"`
	FunctionCallingEnabled    bool                      `json:"function_calling_enabled"`
	ImageInputSupported       bool                      `json:"image_input_supported"`
	WebSearchSupported        bool                      `json:"web_search_supported"`
	ThinkingEnabled           bool                      `json:"thinking_enabled"`
	ThinkingType              string                    `json:"thinking_type,omitempty"`
	ContextManagementEnabled  bool                      `json:"context_management_enabled"`
	ClaudeCompactionSupported bool                      `json:"claude_compaction_supported"`
	AnthropicVersion          string                    `json:"anthropic_version"`
	AnthropicBeta             []string                  `json:"anthropic_beta,omitempty"`
	Checks                    []DiagnosticCheck         `json:"checks"`
	Capabilities              *DiagnosticCapabilities   `json:"capabilities,omitempty"`
	RequestPreview            *DiagnosticRequestPreview `json:"request_preview,omitempty"`
	Smoke                     *DiagnosticSmokeResult    `json:"smoke,omitempty"`
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

// DiagnosticOptions は Claude 診断の入力を表す。
type DiagnosticOptions struct {
	Config               *config.Config
	Model                string
	CatalogModel         string
	RunSmoke             bool
	TextSmoke            bool
	ToolSmoke            bool
	ImageSmoke           bool
	ThinkingSmoke        bool
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

// Diagnose は Claude のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveClaudeDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveClaudeDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	policyCfg := claudeDiagnosticPolicyConfig(cfg, model, catalogModel, 0)
	policy := providerdiag.ClaudeCatalogPolicy(policyCfg, model, catalogModel)
	configCtx := config.WithContext(context.Background(), policyCfg)

	provider := New("diagnostic-key")
	contextManagement := buildContextManagementForModel(policyCfg.ModelCatalogName("claude", model), policyCfg.Compression)
	thinkingEnabled := claudeThinkingActiveForModel(configCtx, catalogModel)

	report := DiagnosticReport{
		Provider:                  "claude",
		APIURL:                    New(os.Getenv(anthropicAPIKeyEnv)).APIURL,
		Model:                     model,
		ModelSource:               modelSource,
		CatalogModel:              catalogModel,
		CatalogModelSource:        catalogSource,
		Route:                     DiagnosticRouteClaudeMessages,
		RouteReason:               "Claude text, tool, image, thinking, and native web search diagnostics use Anthropic Messages",
		MaxOutputTokens:           providerdiag.RuntimeMaxOutputTokens(policyCfg, "claude", model),
		ContextWindowTokens:       policy.ContextWindowTokens,
		FunctionCallingEnabled:    provider.IsFunctionCallingEnabled(),
		ImageInputSupported:       true,
		WebSearchSupported:        true,
		ThinkingEnabled:           thinkingEnabled,
		ThinkingType:              claudeDiagnosticThinkingType(thinkingEnabled, catalogModel),
		ContextManagementEnabled:  contextManagement != nil,
		ClaudeCompactionSupported: provider.SupportsClaudeCompactionWithContext(configCtx, model),
		AnthropicVersion:          provider.anthropicVersion(configCtx, model),
		AnthropicBeta:             provider.anthropicBetaHeaders(configCtx, model, contextManagement),
	}

	if options.requiresEndpointCheck() {
		report.addEndpointCheck()
	}
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
	report.addThinkingCheck()
	report.addContextManagementCheck()
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
	}

	return report
}

func resolveClaudeDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticModel(cfg, "claude", explicitModel, llmcatalog.DefaultModelForProvider("claude"))
}

func resolveClaudeDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, "claude", model, explicitCatalogModel)
}

func claudeDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	return providerdiag.ProviderDiagnosticPolicyConfig(cfg, providerdiag.ProviderDiagnosticPolicyConfigOptions{
		Provider:        "claude",
		Model:           model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
	})
}

func claudeCatalogModelKnown(model string) bool {
	return providerdiag.IsProviderCatalogModelKnown("claude", model)
}

func claudeDiagnosticThinkingType(enabled bool, catalogModel string) string {
	if !enabled {
		return ""
	}
	if IsAdaptiveThinkingModel(catalogModel) {
		return "adaptive"
	}
	return "enabled"
}
