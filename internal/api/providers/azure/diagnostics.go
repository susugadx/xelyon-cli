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
	"github.com/susugadx/xelyon-cli/internal/config"
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

// DiagnosticCheck は Azure 設定診断の 1 項目を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult struct {
	Ran         bool   `json:"ran"`
	ToolPayload bool   `json:"tool_payload"`
	Content     string `json:"content,omitempty"`
	ResponseID  string `json:"response_id,omitempty"`
	Duration    string `json:"duration,omitempty"`
}

// DiagnosticReport は Azure OpenAI の設定診断結果を表す。
type DiagnosticReport struct {
	Provider               string                 `json:"provider"`
	BaseURL                string                 `json:"base_url,omitempty"`
	NormalizedBaseURL      string                 `json:"normalized_base_url,omitempty"`
	AuthMode               string                 `json:"auth_mode,omitempty"`
	Deployment             string                 `json:"deployment,omitempty"`
	DeploymentSource       string                 `json:"deployment_source,omitempty"`
	CatalogModel           string                 `json:"catalog_model,omitempty"`
	CatalogModelSource     string                 `json:"catalog_model_source,omitempty"`
	FunctionCallingEnabled bool                   `json:"function_calling_enabled"`
	ResponsesStore         bool                   `json:"responses_store"`
	ResponsesPersistID     bool                   `json:"responses_persist_response_id"`
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

// DiagnosticOptions は Azure 診断の入力を表す。
type DiagnosticOptions struct {
	Config          *config.Config
	Deployment      string
	CatalogModel    string
	RunSmoke        bool
	ToolSmoke       bool
	SmokeTimeout    time.Duration
	MaxOutputTokens int
	SmokeOutput     io.Writer
}

// Diagnose は Azure OpenAI のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	deployment, deploymentSource := resolveDiagnosticDeployment(cfg, options.Deployment)
	catalogModel, catalogSource := resolveDiagnosticCatalogModel(cfg, deployment, options.CatalogModel)

	report := DiagnosticReport{
		Provider:               "azure",
		BaseURL:                strings.TrimSpace(os.Getenv(baseURLEnv)),
		NormalizedBaseURL:      normalizeBaseURL(os.Getenv(baseURLEnv)),
		AuthMode:               diagnosticAuthMode(),
		Deployment:             deployment,
		DeploymentSource:       deploymentSource,
		CatalogModel:           catalogModel,
		CatalogModelSource:     catalogSource,
		FunctionCallingEnabled: os.Getenv("AZURE_OPENAI_FUNCTION_CALLING") != "0",
		ResponsesStore:         cfg.ResponsesStoreEnabled(),
		ResponsesPersistID:     cfg.ResponsesPersistResponseIDEnabled(),
	}

	report.addBaseURLChecks()
	report.addAuthChecks(ctx)
	report.addDeploymentCheck(cfg, options.Deployment)
	report.addCatalogModelCheck()
	report.addFunctionCallingCheck()
	report.addResponsesRetentionCheck()

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
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "Azure OpenAI deployment accepted a tool payload", smoke.Duration, "")
	}
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

	smokeCfg := config.CloneConfig(cfg)
	smokeCfg.Responses.Store = false
	smokeCfg.Responses.PersistResponseID = false
	smokeCfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
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
	runtime := ui.NewRuntime(strings.NewReader(""), output, output)
	smokeCtx = ui.WithRuntime(smokeCtx, runtime)
	smokeCtx = api.WithAssistantUpdateMode(smokeCtx, api.AssistantUpdatesOff)
	toolPayload := options.ToolSmoke && report.FunctionCallingEnabled
	if toolPayload {
		smokeCtx = api.WithToolDefinitions(smokeCtx, diagnosticSmokeToolDefinitions())
	} else {
		smokeCtx = api.WithToolDefinitions(smokeCtx, nil)
	}
	smokeCtx = config.WithContext(smokeCtx, smokeCfg)

	provider := New(os.Getenv(apiKeyEnv))
	systemPrompt := "Reply briefly."
	if toolPayload {
		systemPrompt = "Use the diagnostic tool."
		provider.SetToolChoice(diagnosticSmokeToolName)
	}
	started := time.Now()
	content, err := provider.ChatWithTools(
		smokeCtx,
		systemPrompt,
		[]api.Message{{Role: "user", Content: "Reply with: xelyon azure doctor ok"}},
		report.Deployment,
	)
	elapsed := time.Since(started).Round(time.Millisecond)

	result := DiagnosticSmokeResult{
		Ran:         true,
		ToolPayload: toolPayload,
		Content:     strings.TrimSpace(content),
		ResponseID:  provider.GetResponseID(),
		Duration:    elapsed.String(),
	}
	if err != nil {
		return result, err
	}
	if toolPayload {
		if !diagnosticSmokeContentHasToolCall(content) {
			return result, fmt.Errorf("tool smoke response did not include %s function_call", diagnosticSmokeToolName)
		}
		return result, nil
	}
	if strings.TrimSpace(content) == "" {
		return result, fmt.Errorf("smoke response content is empty")
	}
	return result, nil
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
