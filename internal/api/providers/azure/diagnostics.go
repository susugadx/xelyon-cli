package azure

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	defaultDiagnosticSmokeTimeout         = 120 * time.Second
	defaultDiagnosticSmokeMaxOutputTokens = 64
)

// DiagnosticStatus は Azure 診断チェックの結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

const (
	DiagnosticRouteResponsesStreaming    = "responses_streaming"
	DiagnosticRouteResponsesNonStreaming = "responses_non_streaming"
)

// DiagnosticCheck は Azure 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeUsage は Azure smoke request で観測した usage を表す。
type DiagnosticSmokeUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	ThinkingTokens      int `json:"thinking_tokens"`
	CachedInputTokens   int `json:"cached_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
}

// DiagnosticSmokeCost は Azure smoke request の cost estimate を表す。
type DiagnosticSmokeCost struct {
	USD                float64 `json:"usd"`
	PricingUnavailable bool    `json:"pricing_unavailable"`
}

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult struct {
	Name               string               `json:"name"`
	Ran                bool                 `json:"ran"`
	Skipped            bool                 `json:"skipped,omitempty"`
	SkipReason         string               `json:"skip_reason,omitempty"`
	ToolPayload        bool                 `json:"tool_payload"`
	RetentionPayload   bool                 `json:"retention_payload"`
	Content            string               `json:"content,omitempty"`
	ResponseID         string               `json:"response_id"`
	PreviousResponseID string               `json:"previous_response_id"`
	Duration           string               `json:"duration,omitempty"`
	UsageObserved      bool                 `json:"usage_observed"`
	Usage              DiagnosticSmokeUsage `json:"usage"`
	Cost               DiagnosticSmokeCost  `json:"cost"`
	Error              string               `json:"error,omitempty"`
}

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult struct {
	Ran              bool                           `json:"ran"`
	ToolPayload      bool                           `json:"tool_payload"`
	RetentionPayload bool                           `json:"retention_payload"`
	Content          string                         `json:"content,omitempty"`
	ResponseID       string                         `json:"response_id"`
	Duration         string                         `json:"duration,omitempty"`
	UsageObserved    bool                           `json:"usage_observed"`
	Usage            DiagnosticSmokeUsage           `json:"usage"`
	Cost             DiagnosticSmokeCost            `json:"cost"`
	Requests         []DiagnosticSmokeRequestResult `json:"requests,omitempty"`
}

// DiagnosticRequestPreview は live request を送らずに構築した request shape を表す。
type DiagnosticRequestPreview struct {
	Requests []DiagnosticRequestPreviewRequest `json:"requests"`
}

// DiagnosticRequestPreviewRequest は doctor smoke request 単位の request preview を表す。
type DiagnosticRequestPreviewRequest struct {
	Name               string            `json:"name"`
	Skipped            bool              `json:"skipped,omitempty"`
	SkipReason         string            `json:"skip_reason,omitempty"`
	ToolPayload        bool              `json:"tool_payload"`
	RetentionPayload   bool              `json:"retention_payload"`
	Route              string            `json:"route,omitempty"`
	Method             string            `json:"method,omitempty"`
	URL                string            `json:"url,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Body               any               `json:"body,omitempty"`
}

// DiagnosticReport は Azure OpenAI の設定診断結果を表す。
type DiagnosticReport struct {
	Provider               string                    `json:"provider"`
	BaseURL                string                    `json:"base_url,omitempty"`
	NormalizedBaseURL      string                    `json:"normalized_base_url,omitempty"`
	AuthMode               string                    `json:"auth_mode,omitempty"`
	Deployment             string                    `json:"deployment,omitempty"`
	DeploymentSource       string                    `json:"deployment_source,omitempty"`
	CatalogModel           string                    `json:"catalog_model,omitempty"`
	CatalogModelSource     string                    `json:"catalog_model_source,omitempty"`
	Route                  string                    `json:"route,omitempty"`
	RouteReason            string                    `json:"route_reason,omitempty"`
	FunctionCallingEnabled bool                      `json:"function_calling_enabled"`
	ResponsesStore         bool                      `json:"responses_store"`
	ResponsesPersistID     bool                      `json:"responses_persist_response_id"`
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

// DiagnosticOptions は Azure 診断の入力を表す。
type DiagnosticOptions struct {
	Config          *config.Config
	Deployment      string
	CatalogModel    string
	RunSmoke        bool
	TextSmoke       bool
	ToolSmoke       bool
	Capabilities    bool
	RetentionSmoke  bool
	PrintRequest    bool
	SmokeTimeout    time.Duration
	MaxOutputTokens int
	SmokeOutput     io.Writer
}

func (o DiagnosticOptions) requiresBaseURLCheck() bool {
	return !o.Capabilities || o.RunSmoke || o.PrintRequest
}

func (o DiagnosticOptions) requiresAuthCheck() bool {
	return !o.PrintRequest && (!o.Capabilities || o.RunSmoke)
}

// Diagnose は Azure OpenAI のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	deployment, deploymentSource := resolveDiagnosticDeployment(cfg, options.Deployment)
	catalogModel, catalogSource := resolveDiagnosticCatalogModel(cfg, deployment, options.CatalogModel)
	route, routeReason := resolveDiagnosticRoute(deployment, catalogModel)

	report := DiagnosticReport{
		Provider:               "azure",
		BaseURL:                strings.TrimSpace(os.Getenv(baseURLEnv)),
		NormalizedBaseURL:      normalizeBaseURL(os.Getenv(baseURLEnv)),
		AuthMode:               diagnosticAuthMode(),
		Deployment:             deployment,
		DeploymentSource:       deploymentSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		Route:                  route,
		RouteReason:            routeReason,
		FunctionCallingEnabled: os.Getenv("AZURE_OPENAI_FUNCTION_CALLING") != "0",
		ResponsesStore:         cfg.ResponsesStoreEnabled(),
		ResponsesPersistID:     cfg.ResponsesPersistResponseIDEnabled(),
	}

	if options.requiresBaseURLCheck() {
		report.addBaseURLChecks()
	}
	if options.requiresAuthCheck() {
		report.addAuthChecks(ctx)
	}
	report.addDeploymentCheck(cfg, options.Deployment)
	report.addCatalogModelCheck()
	report.addRouteCheck()
	report.addCatalogPolicyCheck(cfg)
	report.addFunctionCallingCheck()
	report.addResponsesRetentionCheck()

	if options.Capabilities {
		report.addCapabilities(ctx, cfg)
	}
	if options.PrintRequest {
		report.addRequestPreview(ctx, cfg, options)
	}
	if options.RunSmoke && !options.PrintRequest {
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

func (r *DiagnosticReport) addBaseURLChecks() {
	if r.BaseURL == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"base_url",
			fmt.Sprintf("%s is not set", baseURLEnv),
			"",
			fmt.Sprintf("Set %s=https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1", baseURLEnv),
		)
		return
	}

	parsed, err := url.Parse(r.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"base_url",
			fmt.Sprintf("%s is not a valid absolute URL", baseURLEnv),
			r.BaseURL,
			"Use the Azure OpenAI v1 base URL, for example https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1",
		)
		return
	}
	if strings.EqualFold(parsed.Hostname(), "api.openai.com") {
		r.addCheck(
			DiagnosticStatusFail,
			"base_url",
			fmt.Sprintf("%s points to the public OpenAI API, not Azure OpenAI", baseURLEnv),
			r.BaseURL,
			fmt.Sprintf("Set %s to your Azure OpenAI resource URL, for example https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1", baseURLEnv),
		)
		return
	}

	normalizedParsed, err := url.Parse(r.NormalizedBaseURL)
	if err == nil && strings.Contains(strings.ToLower(normalizedParsed.Path), "/deployments/") {
		r.addCheck(
			DiagnosticStatusFail,
			"base_url",
			"deployment-scoped Azure OpenAI URLs are not supported by the Responses v1 path",
			r.NormalizedBaseURL,
			fmt.Sprintf("Set %s to the resource v1 base URL, not an /openai/deployments/<name> URL", baseURLEnv),
		)
		return
	}

	if parsed.Query().Get("api-version") != "" {
		r.addCheck(
			DiagnosticStatusWarn,
			"api_version",
			"api-version query is ignored by the Azure OpenAI v1 Responses path",
			r.BaseURL,
			fmt.Sprintf("Prefer %s without query parameters: %s", baseURLEnv, r.NormalizedBaseURL),
		)
	}

	if normalizedParsed != nil && strings.TrimRight(normalizedParsed.Path, "/") != "/openai/v1" {
		r.addCheck(
			DiagnosticStatusWarn,
			"base_url_path",
			"base URL path is not the standard Azure OpenAI v1 path",
			r.NormalizedBaseURL,
			"Use /openai/v1 unless this endpoint is an intentional proxy",
		)
	}

	r.addCheck(DiagnosticStatusOK, "base_url", "Azure OpenAI base URL is configured", r.NormalizedBaseURL, "")
}

func (r *DiagnosticReport) addAuthChecks(ctx context.Context) {
	apiKeySet := strings.TrimSpace(os.Getenv(apiKeyEnv)) != ""
	authTokenSet := strings.TrimSpace(os.Getenv(authTokenEnv)) != ""
	authTokenCommandSet := strings.TrimSpace(os.Getenv(authTokenCommandEnv)) != ""

	switch {
	case !apiKeySet && !authTokenSet && !authTokenCommandSet:
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s, %s, or %s is required", apiKeyEnv, authTokenEnv, authTokenCommandEnv),
			"",
			fmt.Sprintf("Set %s for API key auth, or %s / %s for Microsoft Entra ID auth", apiKeyEnv, authTokenEnv, authTokenCommandEnv),
		)
	case apiKeySet && (authTokenSet || authTokenCommandSet):
		r.addCheck(
			DiagnosticStatusWarn,
			"auth",
			fmt.Sprintf("%s is set with Microsoft Entra ID auth env; API key auth will be used", apiKeyEnv),
			"",
			fmt.Sprintf("Unset %s if you want to test Microsoft Entra ID auth", apiKeyEnv),
		)
	case apiKeySet:
		r.addCheck(DiagnosticStatusOK, "auth", "API key auth is configured", apiKeyEnv, "")
		if strings.HasPrefix(strings.TrimSpace(os.Getenv(apiKeyEnv)), "sk-") {
			r.addCheck(
				DiagnosticStatusWarn,
				"auth_key_shape",
				fmt.Sprintf("%s looks like a public OpenAI API key", apiKeyEnv),
				"value starts with sk-",
				fmt.Sprintf("Use an Azure OpenAI resource key, or unset it and use %s / %s for Microsoft Entra ID auth", authTokenEnv, authTokenCommandEnv),
			)
		}
	case authTokenSet && authTokenCommandSet:
		r.addCheck(DiagnosticStatusOK, "auth", "Microsoft Entra ID bearer token auth is configured with refresh command", fmt.Sprintf("%s + %s", authTokenEnv, authTokenCommandEnv), "")
		r.addAuthTokenCommandCheck(ctx, DiagnosticStatusWarn)
	case authTokenSet:
		r.addCheck(DiagnosticStatusOK, "auth", "Microsoft Entra ID bearer token auth is configured", authTokenEnv, "")
	case authTokenCommandSet:
		if r.addAuthTokenCommandCheck(ctx, DiagnosticStatusFail) {
			r.addCheck(DiagnosticStatusOK, "auth", "Microsoft Entra ID token command is configured", authTokenCommandEnv, "")
		}
	}
}

func (r *DiagnosticReport) addAuthTokenCommandCheck(ctx context.Context, failureStatus DiagnosticStatus) bool {
	timeout, timeoutErr := parseAzureAuthTokenCommandTimeout()
	if timeoutErr != nil {
		r.addCheck(
			DiagnosticStatusWarn,
			"auth_token_command_timeout",
			fmt.Sprintf("invalid %s; using %s", authTokenCommandTimeoutEnv, timeout),
			timeoutErr.Error(),
			fmt.Sprintf("Set %s to a positive Go duration such as 10s", authTokenCommandTimeoutEnv),
		)
	}

	_, err := runAzureAuthTokenCommand(ctx, os.Getenv(authTokenCommandEnv), timeout)
	if err != nil {
		r.addCheck(
			failureStatus,
			"auth_token_command",
			fmt.Sprintf("%s failed", authTokenCommandEnv),
			err.Error(),
			"Fix the command, verify Azure CLI login, or set AZURE_OPENAI_AUTH_TOKEN directly",
		)
		return false
	}

	r.addCheck(
		DiagnosticStatusOK,
		"auth_token_command",
		fmt.Sprintf("%s executed successfully", authTokenCommandEnv),
		"token was returned on stdout",
		"",
	)
	return true
}

func (r *DiagnosticReport) addDeploymentCheck(cfg *config.Config, explicitDeployment string) {
	if strings.TrimSpace(r.Deployment) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"deployment",
			"Azure OpenAI deployment is not configured",
			"",
			"Pass --deployment <deployment> or set provider_models.azure.default_model",
		)
		return
	}

	if isUnconfiguredPlaceholderDeployment(cfg, r.Deployment, explicitDeployment) {
		r.addCheck(
			DiagnosticStatusFail,
			"deployment",
			"Azure OpenAI deployment is still the built-in placeholder",
			r.Deployment,
			"Set provider_models.azure.default_model to your Azure deployment name, or pass --deployment <deployment>",
		)
		return
	}

	r.addCheck(
		DiagnosticStatusOK,
		"deployment",
		"Azure OpenAI deployment is configured",
		fmt.Sprintf("%s (%s)", r.Deployment, r.DeploymentSource),
		"",
	)

	if looksLikeOpenAICatalogModel(r.Deployment) &&
		r.CatalogModelSource == "deployment name fallback" {
		r.addCheck(
			DiagnosticStatusWarn,
			"deployment_catalog_mixup",
			"deployment name looks like an OpenAI catalog model",
			r.Deployment,
			"This is OK only if the Azure deployment is named exactly this way; otherwise pass the Azure deployment name as --deployment and keep this value in --catalog-model",
		)
	}
}

func (r *DiagnosticReport) addCatalogModelCheck() {
	if strings.TrimSpace(r.Deployment) == "" {
		return
	}
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_model",
			"catalog_model is not resolved",
			"",
			"Set provider_models.azure.catalog_model when the deployment name differs from the actual model name",
		)
		return
	}

	if r.CatalogModel == r.Deployment && r.CatalogModelSource == "deployment name fallback" {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_model",
			"catalog_model falls back to the deployment name",
			r.CatalogModel,
			"Set provider_models.azure.catalog_model if this deployment is not named exactly like the underlying model",
		)
		return
	}

	if !looksLikeOpenAICatalogModel(r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_model",
			"catalog_model does not look like an OpenAI catalog model",
			r.CatalogModel,
			"Use the underlying model name such as gpt-5.4; do not put the Azure deployment name here",
		)
		return
	}

	r.addCheck(
		DiagnosticStatusOK,
		"catalog_model",
		"catalog_model is resolved",
		fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource),
		"",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	detail := r.routeCheckDetail()
	switch r.Route {
	case DiagnosticRouteResponsesStreaming:
		r.addCheck(DiagnosticStatusOK, "route", "Azure OpenAI Responses streaming route is selected", detail, "")
	case DiagnosticRouteResponsesNonStreaming:
		r.addCheck(DiagnosticStatusOK, "route", "Azure OpenAI Responses non-streaming route is selected", detail, "")
	default:
		r.addCheck(DiagnosticStatusFail, "route", "Azure OpenAI route could not be resolved", detail, "")
	}
}

func (r DiagnosticReport) routeCheckDetail() string {
	route := strings.TrimSpace(r.Route)
	reason := strings.TrimSpace(r.RouteReason)
	switch {
	case route == "":
		return reason
	case reason == "":
		return route
	default:
		return fmt.Sprintf("%s; %s", route, reason)
	}
}

func (r *DiagnosticReport) addCatalogPolicyCheck(cfg *config.Config) {
	deployment := strings.TrimSpace(r.Deployment)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if deployment == "" || catalogModel == "" || !looksLikeOpenAICatalogModel(catalogModel) {
		return
	}

	policyCfg := diagnosticCatalogPolicyConfig(cfg, deployment, catalogModel)
	maxOutput := diagnosticMaxOutputPolicy(policyCfg, deployment, catalogModel)
	contextWindow, contextOK := llmcatalog.KnownModelContextLimit(catalogModel)
	pricing := cost.GetPricingInfoForConfig(policyCfg, "azure", deployment)
	responsesStreaming := openai.ShouldStreamResponses(catalogModel)
	detail := diagnosticCatalogPolicyDetail(catalogModel, contextWindow, contextOK, maxOutput, pricing, responsesStreaming)

	switch {
	case !contextOK:
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is missing context window metadata",
			detail,
			"Use an OpenAI catalog model known to XELYON, or update the model catalog before relying on token limits",
		)
	case !maxOutput.Available:
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is missing max output metadata",
			detail,
			"Use an OpenAI catalog model known to XELYON, or set max_output_tokens explicitly for this deployment",
		)
	case pricing.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_policy",
			"catalog_model is missing pricing metadata",
			detail,
			"Use an OpenAI catalog model with pricing metadata before relying on cost estimates",
		)
	default:
		r.addCheck(
			DiagnosticStatusOK,
			"catalog_policy",
			"catalog_model policy is available",
			detail,
			"",
		)
	}
}

func diagnosticCatalogPolicyConfig(cfg *config.Config, deployment, catalogModel string) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	policyCfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: deployment,
		CatalogModel: catalogModel,
	})
	_ = policyCfg.PatchProviderModelConfig("azure", func(pm *config.ProviderModelConfig) {
		if pm.ModelOverrides == nil {
			pm.ModelOverrides = map[string]config.ModelOverride{}
		}
		override := pm.ModelOverrides[deployment]
		override.CatalogModel = catalogModel
		pm.ModelOverrides[deployment] = override
	})
	return policyCfg
}

type diagnosticMaxOutputPolicyResult struct {
	Tokens          int
	Source          string
	Available       bool
	RuntimeFallback int
}

func diagnosticMaxOutputPolicy(cfg *config.Config, deployment, catalogModel string) diagnosticMaxOutputPolicyResult {
	if override, ok := cfg.ModelOverrideForProvider("azure", deployment); ok && override.MaxOutputTokens > 0 {
		return diagnosticMaxOutputPolicyResult{
			Tokens:    override.MaxOutputTokens,
			Source:    "model_overrides",
			Available: true,
		}
	}

	if tokens, ok := llmcatalog.KnownMaxOutputTokens(catalogModel); ok {
		return diagnosticMaxOutputPolicyResult{
			Tokens:    tokens,
			Source:    "catalog",
			Available: true,
		}
	}

	runtimeFallback := api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), "azure", deployment)
	return diagnosticMaxOutputPolicyResult{
		Source:          "missing",
		RuntimeFallback: runtimeFallback,
	}
}

func diagnosticCatalogPolicyDetail(
	catalogModel string,
	contextWindow int,
	contextOK bool,
	maxOutput diagnosticMaxOutputPolicyResult,
	pricing cost.PricingInfo,
	responsesStreaming bool,
) string {
	contextDetail := "unknown"
	if contextOK {
		contextDetail = fmt.Sprintf("%d", contextWindow)
	}
	return fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, responses_streaming=%t, %s",
		catalogModel,
		contextDetail,
		diagnosticMaxOutputDetail(maxOutput),
		responsesStreaming,
		diagnosticPricingDetail(pricing),
	)
}

func diagnosticMaxOutputDetail(maxOutput diagnosticMaxOutputPolicyResult) string {
	if maxOutput.Available {
		return fmt.Sprintf("%d (%s)", maxOutput.Tokens, maxOutput.Source)
	}
	if maxOutput.RuntimeFallback > 0 {
		return fmt.Sprintf("missing (runtime_fallback=%d)", maxOutput.RuntimeFallback)
	}
	return "missing"
}

func diagnosticPricingDetail(pricing cost.PricingInfo) string {
	if pricing.PricingUnavailable {
		return "pricing=unavailable"
	}
	return fmt.Sprintf(
		"pricing=input $%.2f/M cached $%.3f/M output $%.2f/M",
		pricing.InputCostPerM,
		pricing.CachedInputCostPerM,
		pricing.OutputCostPerM,
	)
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"function calling payloads are enabled for Azure OpenAI",
			"",
			"Set AZURE_OPENAI_FUNCTION_CALLING=0 if this deployment rejects tool payloads",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"function_calling",
		"function calling payloads are disabled for Azure OpenAI",
		"AZURE_OPENAI_FUNCTION_CALLING=0",
		"",
	)
}

func (r *DiagnosticReport) addResponsesRetentionCheck() {
	message := fmt.Sprintf("responses.store=%t, responses.persist_response_id=%t", r.ResponsesStore, r.ResponsesPersistID)
	if !r.ResponsesStore || !r.ResponsesPersistID {
		r.addCheck(
			DiagnosticStatusWarn,
			"responses_retention",
			"advanced Responses retention override is active",
			message,
			"Most users should leave these settings enabled; disable them only when your retention policy requires it",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "responses_retention", message, "", "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	if r.HasFailures() {
		r.addCheck(
			DiagnosticStatusWarn,
			"smoke",
			"live smoke was skipped because prerequisite checks failed",
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
			"AZURE_OPENAI_FUNCTION_CALLING=0",
			"Unset AZURE_OPENAI_FUNCTION_CALLING or set it to 1 before rerunning --tool-smoke",
		)
	}

	smoke, err := runDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "smoke", "live Azure OpenAI smoke request failed", err.Error(), "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live Azure OpenAI smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "Azure OpenAI deployment accepted a tool payload", smoke.Duration, "")
	}
	if smoke.RetentionPayload {
		r.addCheck(DiagnosticStatusOK, "retention_smoke", "Azure OpenAI deployment accepted a previous_response_id chain", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if strings.TrimSpace(smoke.ResponseID) != "" {
		r.addCheck(DiagnosticStatusOK, "response_id", "Azure OpenAI smoke returned a response ID", smoke.ResponseID, "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"response_id",
			"Azure OpenAI smoke succeeded but response ID was not returned",
			"",
			"Check whether the endpoint returns Responses API response.created/id metadata",
		)
	}

	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "Azure OpenAI smoke usage was observed", diagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"Azure OpenAI smoke succeeded but usage was not observed",
			"",
			"Check whether the endpoint returns Responses API usage metadata",
		)
	}

	switch {
	case !smoke.UsageObserved:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Azure OpenAI smoke cost estimate was skipped because usage was not observed",
			"",
			"Rerun smoke after usage metadata is available",
		)
	case smoke.Cost.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"Azure OpenAI smoke cost pricing is unavailable",
			"",
			"Use an OpenAI catalog model with pricing metadata before relying on smoke cost estimates",
		)
	default:
		r.addCheck(DiagnosticStatusOK, "cost", "Azure OpenAI smoke cost estimate is available", fmt.Sprintf("$%.8f USD", smoke.Cost.USD), "")
	}
}

func diagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	return fmt.Sprintf(
		"input_tokens=%d, cached_input_tokens=%d, output_tokens=%d, thinking_tokens=%d, cache_creation_tokens=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}

func resolveDiagnosticDeployment(cfg *config.Config, explicitDeployment string) (string, string) {
	if deployment := strings.TrimSpace(explicitDeployment); deployment != "" {
		return deployment, "--deployment"
	}
	if envModel := strings.TrimSpace(os.Getenv("XELYON_MODEL")); envModel != "" {
		return envModel, "XELYON_MODEL"
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("azure")); explicit != "" {
		return explicit, "provider_models.azure.default_model"
	}
	if config.SameProviderRuntimeIdentity("azure", cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider("azure"))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if selected := strings.TrimSpace(cfg.GetEffectiveModelForProvider("azure")); selected != "" {
		return selected, "built-in provider default"
	}
	return "", ""
}

func resolveDiagnosticCatalogModel(cfg *config.Config, deployment, explicitCatalogModel string) (string, string) {
	deployment = strings.TrimSpace(deployment)
	if catalogModel := strings.TrimSpace(explicitCatalogModel); catalogModel != "" {
		return catalogModel, "--catalog-model"
	}
	if deployment == "" {
		return "", ""
	}

	if override, ok := cfg.ModelOverrideForProvider("azure", deployment); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.azure.model_overrides"
		}
	}

	if pm, ok := cfg.GetProviderModelConfig("azure"); ok && pm.DefaultModel == deployment {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.azure.catalog_model"
		}
	}

	return cfg.ModelCatalogName("azure", deployment), "deployment name fallback"
}

func resolveDiagnosticRoute(deployment, catalogModel string) (string, string) {
	deployment = strings.TrimSpace(deployment)
	catalogModel = strings.TrimSpace(catalogModel)
	if deployment == "" {
		return "", "deployment is not resolved"
	}

	if openai.ShouldStreamResponses(catalogModel) {
		return DiagnosticRouteResponsesStreaming, fmt.Sprintf("deployment=%s uses Responses API; %s", deployment, diagnosticResponsesStreamingReason(catalogModel, true))
	}
	return DiagnosticRouteResponsesNonStreaming, fmt.Sprintf("deployment=%s uses Responses API; %s", deployment, diagnosticResponsesStreamingReason(catalogModel, false))
}

func diagnosticResponsesStreamingReason(catalogModel string, streaming bool) string {
	catalogModel = strings.TrimSpace(catalogModel)
	if catalogModel == "" {
		return "catalog_model is not resolved; Responses streaming defaults to enabled"
	}
	if streaming {
		return fmt.Sprintf("catalog_model=%s supports Responses streaming", catalogModel)
	}
	return fmt.Sprintf("catalog_model=%s disables Responses streaming", catalogModel)
}

func isUnconfiguredPlaceholderDeployment(cfg *config.Config, deployment, explicitDeployment string) bool {
	builtInDefault := strings.TrimSpace(config.DefaultConfig().GetEffectiveModelForProvider("azure"))
	if builtInDefault == "" || !strings.EqualFold(strings.TrimSpace(deployment), builtInDefault) {
		return false
	}
	if strings.TrimSpace(explicitDeployment) != "" || strings.TrimSpace(os.Getenv("XELYON_MODEL")) != "" {
		return false
	}
	return strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("azure")) == ""
}

func diagnosticAuthMode() string {
	apiKeySet := strings.TrimSpace(os.Getenv(apiKeyEnv)) != ""
	authTokenSet := strings.TrimSpace(os.Getenv(authTokenEnv)) != ""
	authTokenCommandSet := strings.TrimSpace(os.Getenv(authTokenCommandEnv)) != ""
	switch {
	case apiKeySet:
		return "api_key"
	case authTokenSet && authTokenCommandSet:
		return "entra_id_command"
	case authTokenSet:
		return "entra_id"
	case authTokenCommandSet:
		return "entra_id_command"
	default:
		return "missing"
	}
}

func looksLikeOpenAICatalogModel(model string) bool {
	return llmcatalog.InferProviderFromModel(model) == "openai"
}

func runDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseSmokeCfg := config.CloneConfig(cfg)
	baseSmokeCfg.Responses.Store = false
	baseSmokeCfg.Responses.PersistResponseID = false
	baseSmokeCfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: report.Deployment,
		CatalogModel: report.CatalogModel,
		ModelOverrides: map[string]config.ModelOverride{
			report.Deployment: {
				CatalogModel:    report.CatalogModel,
				MaxOutputTokens: maxOutputTokens,
			},
		},
	})

	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(apiKeyEnv))
	defer provider.ClearToolChoice()
	defer provider.ClearResponseID()
	result := DiagnosticSmokeResult{Ran: true}
	started := time.Now()

	for _, request := range diagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			result.Requests = append(result.Requests, DiagnosticSmokeRequestResult{
				Name:        request.Name,
				Skipped:     true,
				SkipReason:  "Azure OpenAI function calling payloads are disabled (AZURE_OPENAI_FUNCTION_CALLING=0)",
				ToolPayload: true,
			})
			continue
		}

		requestCfg := diagnosticSmokeRequestConfig(baseSmokeCfg, request)
		requestResult, err := runDiagnosticSmokeRequest(smokeCtx, requestCfg, provider, report, request, output)
		result.Requests = append(result.Requests, requestResult)
		result.addRequestObservation(requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

type diagnosticSmokeRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	ToolPayload      bool
	RetentionPayload bool
}

func diagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []diagnosticSmokeRequest {
	textSmoke := options.TextSmoke || (!options.ToolSmoke && !options.RetentionSmoke)
	if options.ToolSmoke && !functionCallingEnabled {
		textSmoke = true
	}

	var requests []diagnosticSmokeRequest
	if textSmoke {
		requests = append(requests, diagnosticSmokeRequest{
			Name:         "text",
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon azure doctor ok",
		})
	}
	if options.ToolSmoke {
		requests = append(requests, diagnosticSmokeRequest{
			Name:         "tool",
			SystemPrompt: "Use the diagnostic tool.",
			UserContent:  `Call xelyon_azure_doctor_probe exactly once with {} and do not answer in prose.`,
			ToolPayload:  true,
		})
	}
	if options.RetentionSmoke {
		requests = append(requests,
			diagnosticSmokeRequest{
				Name:             "retention_initial",
				SystemPrompt:     "Reply briefly.",
				UserContent:      "Reply with: xelyon azure retention initial ok",
				RetentionPayload: true,
			},
			diagnosticSmokeRequest{
				Name:             "retention_followup",
				SystemPrompt:     "Reply briefly.",
				UserContent:      "Reply with: xelyon azure retention followup ok",
				RetentionPayload: true,
			},
		)
	}
	return requests
}

func diagnosticSmokeRequestConfig(base *config.Config, request diagnosticSmokeRequest) *config.Config {
	cfg := config.CloneConfig(base)
	if request.RetentionPayload {
		cfg.Responses.Store = true
	}
	return cfg
}

func runDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request diagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	if request.ToolPayload {
		provider.SetToolChoice(diagnosticSmokeToolName)
	} else {
		provider.ClearToolChoice()
	}

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})

	observedRequests, restoreObserver := observeDiagnosticResponsesRequests(provider)
	defer restoreObserver()

	started := time.Now()
	content, responseID, err := runDiagnosticSmokeResponsesRequest(
		requestCtx,
		provider,
		request.SystemPrompt,
		[]api.Message{{Role: "user", Content: request.UserContent}},
		report.Deployment,
		request.RetentionPayload,
	)
	elapsed := time.Since(started).Round(time.Millisecond)
	if !request.RetentionPayload {
		provider.ClearResponseID()
	}
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "azure", report.Deployment, usage)
	observed := observedRequests()
	previousResponseID := diagnosticObservedPreviousResponseID(observed)

	result := DiagnosticSmokeRequestResult{
		Name:               request.Name,
		Ran:                true,
		ToolPayload:        request.ToolPayload,
		RetentionPayload:   request.RetentionPayload,
		Content:            strings.TrimSpace(content),
		ResponseID:         strings.TrimSpace(responseID),
		PreviousResponseID: previousResponseID,
		Duration:           elapsed.String(),
		UsageObserved:      usageObserved,
		Usage:              diagnosticSmokeUsage(usage),
		Cost:               diagnosticSmokeCost(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.RetentionPayload {
		if err := validateDiagnosticRetentionSmokeRequest(request, observed, result); err != nil {
			result.Error = err.Error()
			return result, err
		}
		if strings.TrimSpace(content) == "" {
			result.Error = fmt.Sprintf("%s smoke response content is empty", request.Name)
			return result, fmt.Errorf("%s", result.Error)
		}
		return result, nil
	}
	if request.ToolPayload {
		if !diagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", diagnosticSmokeToolName)
			return result, fmt.Errorf("%s", result.Error)
		}
		return result, nil
	}
	if strings.TrimSpace(content) == "" {
		result.Error = fmt.Sprintf("%s smoke response content is empty", request.Name)
		return result, fmt.Errorf("%s", result.Error)
	}
	return result, nil
}

func newDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request diagnosticSmokeRequest, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, diagnosticSmokeToolDefinitions())
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	}
	return config.WithContext(requestCtx, cfg)
}

func observeDiagnosticResponsesRequests(provider *Provider) (func() []responsesRequest, func()) {
	if provider == nil {
		return func() []responsesRequest { return nil }, func() {}
	}
	previousObserver := provider.responsesRequestObserver
	observed := make([]responsesRequest, 0, 1)
	provider.responsesRequestObserver = func(request responsesRequest) {
		observed = append(observed, request)
		if previousObserver != nil {
			previousObserver(request)
		}
	}
	snapshot := func() []responsesRequest {
		return append([]responsesRequest(nil), observed...)
	}
	restore := func() {
		provider.responsesRequestObserver = previousObserver
	}
	return snapshot, restore
}

func diagnosticObservedPreviousResponseID(requests []responsesRequest) string {
	if len(requests) == 0 {
		return ""
	}
	return strings.TrimSpace(requests[0].PreviousResponseID)
}

func validateDiagnosticRetentionSmokeRequest(request diagnosticSmokeRequest, observed []responsesRequest, result DiagnosticSmokeRequestResult) error {
	if len(observed) == 0 {
		return fmt.Errorf("%s smoke did not build a Responses request", request.Name)
	}
	if !observed[0].Store {
		return fmt.Errorf("%s smoke request did not set responses.store=true", request.Name)
	}
	if request.Name != "retention_followup" {
		return nil
	}
	if strings.TrimSpace(result.PreviousResponseID) == "" {
		return fmt.Errorf("retention followup did not send previous_response_id")
	}
	for _, req := range observed {
		if strings.TrimSpace(req.PreviousResponseID) != result.PreviousResponseID {
			return fmt.Errorf("retention followup retry changed previous_response_id")
		}
	}
	return nil
}

func runDiagnosticSmokeResponsesRequest(
	ctx context.Context,
	provider *Provider,
	systemPrompt string,
	history []api.Message,
	model string,
	retentionPayload bool,
) (string, string, error) {
	if provider == nil {
		return "", "", fmt.Errorf("azure diagnostic smoke provider is nil")
	}
	if retentionPayload {
		return provider.chatWithResponsesResult(ctx, systemPrompt, history, model)
	}
	provider.responsesLocalSkip = false
	return provider.runResponsesRequest(ctx, responsesRequestRunOptions{
		URL: provider.responsesURL(),
		BuildRequest: func() responsesRequest {
			return provider.buildChatResponsesRequest(ctx, systemPrompt, history, model)
		},
		DebugName:   "Azure",
		Debug:       os.Getenv("XELYON_DEBUG_AZURE") == "1",
		DebugWriter: api.ErrorWriterFromContext(ctx),
	})
}

func (r *DiagnosticSmokeResult) addRequestObservation(request DiagnosticSmokeRequestResult) {
	if request.Skipped {
		return
	}
	if request.ToolPayload {
		r.ToolPayload = true
	}
	if request.RetentionPayload {
		r.RetentionPayload = true
	}
	if strings.TrimSpace(r.Content) == "" {
		r.Content = request.Content
	}
	if strings.TrimSpace(r.ResponseID) == "" {
		r.ResponseID = request.ResponseID
	}

	var usage api.Usage
	usage.InputTokens = request.Usage.InputTokens
	usage.OutputTokens = request.Usage.OutputTokens
	usage.ThinkingTokens = request.Usage.ThinkingTokens
	usage.CachedInputTokens = request.Usage.CachedInputTokens
	usage.CacheCreationTokens = request.Usage.CacheCreationTokens

	var current api.Usage
	current.InputTokens = r.Usage.InputTokens
	current.OutputTokens = r.Usage.OutputTokens
	current.ThinkingTokens = r.Usage.ThinkingTokens
	current.CachedInputTokens = r.Usage.CachedInputTokens
	current.CacheCreationTokens = r.Usage.CacheCreationTokens
	current.Add(usage)
	r.Usage = diagnosticSmokeUsage(current)
	if request.Cost.PricingUnavailable {
		r.Cost.PricingUnavailable = true
	} else {
		r.Cost.USD += request.Cost.USD
	}
	r.UsageObserved = r.allRanRequestsObservedUsage()
}

func (r *DiagnosticSmokeResult) allRanRequestsObservedUsage() bool {
	observedAnyRequest := false
	for _, request := range r.Requests {
		if request.Skipped || !request.Ran {
			continue
		}
		observedAnyRequest = true
		if !request.UsageObserved {
			return false
		}
	}
	return observedAnyRequest
}

func diagnosticSmokeUsage(usage api.Usage) DiagnosticSmokeUsage {
	return DiagnosticSmokeUsage{
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}

func diagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return DiagnosticSmokeCost{
		USD:                estimate.Cost,
		PricingUnavailable: estimate.PricingUnavailable,
	}
}

const diagnosticSmokeToolName = "xelyon_azure_doctor_probe"

func diagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        diagnosticSmokeToolName,
		Description: "No-op diagnostic probe used to verify Azure OpenAI tool calling.",
		Parameters: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           map[string]interface{}{},
		},
	}}
}

func diagnosticSmokeContentHasToolCall(content string) bool {
	return strings.Contains(content, `"tool":"`+diagnosticSmokeToolName+`"`)
}
