package bedrock

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	defaultBedrockDiagnosticSmokeTimeout         = 120 * time.Second
	defaultBedrockDiagnosticSmokeMaxOutputTokens = 64
	defaultBedrockDiagnosticAWSAuthTimeout       = 10 * time.Second
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
type DiagnosticSmokeUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	ThinkingTokens      int `json:"thinking_tokens"`
	CachedInputTokens   int `json:"cached_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
}

// DiagnosticSmokeCost は Bedrock smoke request の cost estimate を表す。
type DiagnosticSmokeCost struct {
	USD                float64 `json:"usd"`
	PricingUnavailable bool    `json:"pricing_unavailable"`
}

// DiagnosticSmokeRequestResult は live smoke の request 単位の結果を表す。
type DiagnosticSmokeRequestResult struct {
	Name            string               `json:"name"`
	Ran             bool                 `json:"ran"`
	Skipped         bool                 `json:"skipped,omitempty"`
	SkipReason      string               `json:"skip_reason,omitempty"`
	ToolPayload     bool                 `json:"tool_payload,omitempty"`
	ImagePayload    bool                 `json:"image_payload,omitempty"`
	ThinkingEnabled bool                 `json:"thinking_enabled,omitempty"`
	Content         string               `json:"content,omitempty"`
	RequestID       string               `json:"request_id"`
	Duration        string               `json:"duration,omitempty"`
	UsageObserved   bool                 `json:"usage_observed"`
	Usage           DiagnosticSmokeUsage `json:"usage"`
	Cost            DiagnosticSmokeCost  `json:"cost"`
	Error           string               `json:"error,omitempty"`
}

// DiagnosticSmokeResult は live smoke 実行の結果を表す。
type DiagnosticSmokeResult struct {
	Ran           bool                           `json:"ran"`
	UsageObserved bool                           `json:"usage_observed"`
	Usage         DiagnosticSmokeUsage           `json:"usage"`
	Cost          DiagnosticSmokeCost            `json:"cost"`
	Requests      []DiagnosticSmokeRequestResult `json:"requests,omitempty"`
}

// DiagnosticReport は Bedrock の設定診断結果を表す。
type DiagnosticReport struct {
	Provider               string                 `json:"provider"`
	Region                 string                 `json:"region"`
	Model                  string                 `json:"model"`
	ModelSource            string                 `json:"model_source"`
	CatalogModel           string                 `json:"catalog_model"`
	CatalogModelSource     string                 `json:"catalog_model_source"`
	Route                  string                 `json:"route"`
	FunctionCallingEnabled bool                   `json:"function_calling_enabled"`
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
	SmokeTimeout    time.Duration
	MaxOutputTokens int
	SmokeOutput     io.Writer

	invokeClient     invokeModelWithResponseStreamClient
	converseClient   converseStreamClient
	skipAWSAuthCheck bool
}

// Diagnose は Bedrock のローカル設定と、必要に応じて live smoke を検証する。
func Diagnose(ctx context.Context, options DiagnosticOptions) DiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := resolveBedrockDiagnosticModel(cfg, options.Model)
	catalogModel, catalogSource := resolveBedrockDiagnosticCatalogModel(cfg, model, options.CatalogModel)
	route := resolveBedrockRoute(model, catalogModel)
	smokeRequests := bedrockDiagnosticSmokeRequests(options)
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
	report.addRouteCheck(route, smokeRequests)
	report.addCatalogPolicyCheck(cfg, route)
	report.addFunctionCallingCheck()

	if options.RunSmoke {
		report.runSmokeIfReady(ctx, cfg, options, smokeRequests)
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

func (r *DiagnosticReport) addAWSConfigChecks(ctx context.Context, awsCfg aws.Config, loadErr error, options DiagnosticOptions) {
	if ctx == nil {
		ctx = context.Background()
	}
	if loadErr != nil {
		r.addCheck(
			DiagnosticStatusFail,
			"aws_config",
			"AWS config could not be loaded",
			loadErr.Error(),
			"Verify AWS_REGION/AWS_DEFAULT_REGION and the AWS shared config files",
		)
		return
	}

	regionSource := "default"
	if explicitAWSRegionFromEnv() != "" {
		regionSource = "environment"
	}
	r.addCheck(DiagnosticStatusOK, "region", "AWS region is resolved", fmt.Sprintf("%s (%s)", r.Region, regionSource), "")

	if options.skipAWSAuthCheck || options.invokeClient != nil || options.converseClient != nil {
		r.addCheck(DiagnosticStatusOK, "auth", "AWS credential check was skipped for injected diagnostic clients", "", "")
		return
	}

	authCtx, cancel := context.WithTimeout(ctx, defaultBedrockDiagnosticAWSAuthTimeout)
	defer cancel()
	creds, err := awsCfg.Credentials.Retrieve(authCtx)
	if err != nil {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			"AWS credentials could not be resolved",
			err.Error(),
			"Configure IAM role credentials, AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY, AWS_PROFILE, or another AWS SDK credential source",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "AWS credentials are resolved", creds.Source, "")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("bedrock") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "bedrock provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "bedrock provider is not registered", "", "Ensure providers/all imports the Bedrock provider")
}

func (r *DiagnosticReport) addModelConfigCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"model",
			"Bedrock model is not configured",
			"",
			"Pass --model <bedrock-model-id> or set provider_models.bedrock.default_model",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "model", "Bedrock model is resolved", fmt.Sprintf("%s (%s)", r.Model, r.ModelSource), "")

	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_model",
			"catalog_model is not resolved",
			"",
			"Set provider_models.bedrock.catalog_model when the runtime model is an alias",
		)
		return
	}
	if r.CatalogModel == r.Model && r.CatalogModelSource == "model name fallback" && !llmcatalog.IsBedrockModelID(r.Model) {
		r.addCheck(
			DiagnosticStatusWarn,
			"catalog_model",
			"catalog_model falls back to the runtime model",
			r.CatalogModel,
			"Set provider_models.bedrock.catalog_model when this is an internal alias rather than an AWS model ID",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "catalog_model", "catalog_model is resolved", fmt.Sprintf("%s (%s)", r.CatalogModel, r.CatalogModelSource), "")
}

func (r *DiagnosticReport) addRouteCheck(route bedrockRoute, smokeRequests []bedrockDiagnosticSmokeRequest) {
	switch route {
	case bedrockRouteClaudeMessages:
		r.addCheck(DiagnosticStatusOK, "route", "Bedrock Claude Messages route is selected", r.Route, "")
	case bedrockRouteConverseStream:
		if r.FunctionCallingEnabled && bedrockDiagnosticSmokeRequestsUseToolPayload(smokeRequests) && !llmcatalog.BedrockConverseToolUseSupported(r.Model, r.CatalogModel) {
			r.addCheck(
				DiagnosticStatusFail,
				"route",
				"Bedrock ConverseStream route is selected with unsupported streaming tool use",
				fmt.Sprintf("model=%s, catalog_model=%s", r.Model, r.CatalogModel),
				"Use a Converse model verified for streaming tool use, set a supported catalog_model, or omit --tool-smoke for text-only diagnostics",
			)
			return
		}
		r.addCheck(DiagnosticStatusOK, "route", "Bedrock ConverseStream route is selected", r.Route, "")
	default:
		r.addCheck(DiagnosticStatusFail, "route", "Bedrock route could not be resolved", r.Route, "")
	}
}

func (r *DiagnosticReport) addCatalogPolicyCheck(cfg *config.Config, route bedrockRoute) {
	model := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if model == "" || catalogModel == "" {
		return
	}

	policyCfg := bedrockDiagnosticPolicyConfig(cfg, model, catalogModel)
	contextWindow, contextOK := llmcatalog.KnownModelContextLimit(catalogModel)
	maxOutput, maxOutputOK := bedrockDiagnosticMaxOutputTokens(policyCfg, route, model, catalogModel)
	pricing := cost.GetPricingInfoForConfig(policyCfg, "bedrock", model)
	detail := fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, %s",
		catalogModel,
		bedrockDiagnosticIntDetail(contextWindow, contextOK),
		bedrockDiagnosticIntDetail(maxOutput, maxOutputOK),
		bedrockDiagnosticPricingDetail(pricing),
	)

	switch {
	case !contextOK:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a Bedrock model ID known to XELYON before relying on token-limit diagnostics")
	case !maxOutputOK:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Converse requests will omit maxTokens unless a known catalog model or max_output_tokens override is configured")
	case pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing Bedrock pricing metadata", detail, "Use a Bedrock model ID with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func bedrockDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	override := config.ModelOverride{CatalogModel: catalogModel}
	if existingOverride, ok := policyCfg.ModelOverrideForProvider("bedrock", model); ok {
		override = existingOverride
		override.CatalogModel = catalogModel
	}
	policyCfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel: model,
		CatalogModel: catalogModel,
		ModelOverrides: map[string]config.ModelOverride{
			model: override,
		},
	})
	return policyCfg
}

func bedrockDiagnosticMaxOutputTokens(cfg *config.Config, route bedrockRoute, model, catalogModel string) (int, bool) {
	if route == bedrockRouteConverseStream {
		return converseMaxTokens(bedrockRequestContext{
			model:        model,
			catalogModel: catalogModel,
			route:        route,
			cfg:          cfg,
		})
	}
	ctx := config.WithContext(context.Background(), cfg)
	maxTokens := api.GetMaxOutputTokens(ctx, "bedrock", model)
	return maxTokens, maxTokens > 0
}

func bedrockDiagnosticIntDetail(value int, ok bool) string {
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%d", value)
}

func bedrockDiagnosticPricingDetail(pricing cost.PricingInfo) string {
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
		r.addCheck(DiagnosticStatusOK, "function_calling", "Bedrock function calling payloads are enabled", "", "Set BEDROCK_FUNCTION_CALLING=0 only for text-only troubleshooting")
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "Bedrock function calling payloads are disabled", "BEDROCK_FUNCTION_CALLING=0", "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions, smokeRequests []bedrockDiagnosticSmokeRequest) {
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

	smoke, err := runBedrockDiagnosticSmoke(ctx, cfg, *r, options, smokeRequests)
	r.Smoke = &smoke
	r.addSmokeObservationChecks(smoke)
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "smoke", "live Bedrock smoke request failed", err.Error(), "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live Bedrock smoke requests completed", "", "")
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	for _, request := range smoke.Requests {
		if request.Skipped {
			r.addCheck(DiagnosticStatusWarn, request.Name+"_smoke", "Bedrock smoke request was skipped", request.SkipReason, "")
			continue
		}
		if !request.Ran {
			continue
		}
		if request.Error == "" {
			r.addCheck(DiagnosticStatusOK, request.Name+"_smoke", "Bedrock smoke request succeeded", request.Duration, "")
		} else {
			r.addCheck(DiagnosticStatusFail, request.Name+"_smoke", "Bedrock smoke request failed", request.Error, "")
			continue
		}
		if strings.TrimSpace(request.RequestID) != "" {
			r.addCheck(DiagnosticStatusOK, request.Name+"_request_id", "Bedrock smoke returned a request ID", request.RequestID, "")
		} else {
			r.addCheck(DiagnosticStatusWarn, request.Name+"_request_id", "Bedrock smoke succeeded but request ID was not returned", "", "Check AWS SDK ResultMetadata request ID propagation")
		}
		if request.UsageObserved {
			r.addCheck(DiagnosticStatusOK, request.Name+"_usage", "Bedrock smoke usage was observed", diagnosticSmokeUsageDetail(request.Usage), "")
		} else {
			r.addCheck(DiagnosticStatusWarn, request.Name+"_usage", "Bedrock smoke succeeded but usage was not observed", "", "Check whether the Bedrock stream emitted usage metadata")
		}
		switch {
		case !request.UsageObserved:
			r.addCheck(DiagnosticStatusWarn, request.Name+"_cost", "Bedrock smoke cost estimate was skipped because usage was not observed", "", "Rerun smoke after usage metadata is available")
		case request.Cost.PricingUnavailable:
			r.addCheck(DiagnosticStatusWarn, request.Name+"_cost", "Bedrock smoke cost pricing is unavailable", "", "Use a Bedrock catalog model with pricing metadata before relying on smoke cost estimates")
		default:
			r.addCheck(DiagnosticStatusOK, request.Name+"_cost", "Bedrock smoke cost estimate is available", fmt.Sprintf("$%.8f USD", request.Cost.USD), "")
		}
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

func resolveBedrockDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	if model := strings.TrimSpace(explicitModel); model != "" {
		return model, "--model"
	}
	if envModel := strings.TrimSpace(os.Getenv("XELYON_MODEL")); envModel != "" {
		return envModel, "XELYON_MODEL"
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("bedrock")); explicit != "" {
		return explicit, "provider_models.bedrock.default_model"
	}
	if config.SameProviderRuntimeIdentity("bedrock", cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider("bedrock"))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if selected := strings.TrimSpace(cfg.GetEffectiveModelForProvider("bedrock")); selected != "" {
		return selected, "built-in provider default"
	}
	return defaultModel, "provider fallback"
}

func resolveBedrockDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	model = strings.TrimSpace(model)
	if catalogModel := strings.TrimSpace(explicitCatalogModel); catalogModel != "" {
		return catalogModel, "--catalog-model"
	}
	if model == "" {
		return "", ""
	}

	if override, ok := cfg.ModelOverrideForProvider("bedrock", model); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.bedrock.model_overrides"
		}
	}

	if pm, ok := cfg.GetProviderModelConfig("bedrock"); ok && strings.TrimSpace(pm.DefaultModel) == model {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.bedrock.catalog_model"
		}
	}

	return cfg.ModelCatalogName("bedrock", model), "model name fallback"
}

type bedrockDiagnosticSmokeRequest struct {
	Name            string
	SystemPrompt    string
	UserContent     string
	ToolPayload     bool
	ImagePayload    bool
	ThinkingEnabled bool
}

func runBedrockDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions, smokeRequests []bedrockDiagnosticSmokeRequest) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultBedrockDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultBedrockDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := bedrockDiagnosticSmokeConfig(cfg, report, maxOutputTokens)

	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider, err := newBedrockDiagnosticSmokeProvider(smokeCfg, report.Region, options)
	if err != nil {
		return DiagnosticSmokeResult{Ran: true}, err
	}
	result := DiagnosticSmokeResult{Ran: true}
	for _, request := range smokeRequests {
		if skipReason, ok := bedrockDiagnosticSmokeSkipReason(report, request); ok {
			result.Requests = append(result.Requests, DiagnosticSmokeRequestResult{
				Name:            request.Name,
				Skipped:         true,
				SkipReason:      skipReason,
				ToolPayload:     request.ToolPayload,
				ImagePayload:    request.ImagePayload,
				ThinkingEnabled: request.ThinkingEnabled,
			})
			continue
		}
		requestResult, requestErr := runBedrockDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report.Model, request, output)
		result.Requests = append(result.Requests, requestResult)
		result.addRequestObservation(requestResult)
		if requestErr != nil {
			return result, requestErr
		}
	}
	return result, nil
}

func bedrockDiagnosticSmokeConfig(cfg *config.Config, report DiagnosticReport, maxOutputTokens int) *config.Config {
	smokeCfg := config.CloneConfig(cfg)
	catalogModel := strings.TrimSpace(report.CatalogModel)
	if catalogModel == "" {
		catalogModel = report.Model
	}
	smokeCfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel: report.Model,
		CatalogModel: catalogModel,
		ModelOverrides: map[string]config.ModelOverride{
			report.Model: {
				CatalogModel:    catalogModel,
				MaxOutputTokens: maxOutputTokens,
			},
		},
	})
	smokeCfg.PromptCache.Enabled = false
	smokeCfg.Compression.ClaudeCompaction = false
	return smokeCfg
}

func newBedrockDiagnosticSmokeProvider(cfg *config.Config, region string, options DiagnosticOptions) (*Provider, error) {
	if options.invokeClient != nil || options.converseClient != nil {
		return &Provider{
			client:         options.invokeClient,
			converseClient: options.converseClient,
			region:         region,
			runtimeConfig:  cfg,
		}, nil
	}
	provider, err := New()
	if err != nil {
		return nil, err
	}
	provider.SetRuntimeConfig(cfg)
	return provider, nil
}

func bedrockDiagnosticSmokeRequests(options DiagnosticOptions) []bedrockDiagnosticSmokeRequest {
	var requests []bedrockDiagnosticSmokeRequest
	if options.TextSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:         "text",
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon bedrock doctor ok",
		})
	}
	if options.ToolSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:         "tool",
			SystemPrompt: "Use the diagnostic tool.",
			UserContent:  `Call xelyon_bedrock_doctor_probe exactly once with {"value":"bedrock-tool-ok"} and do not answer in prose.`,
			ToolPayload:  true,
		})
	}
	if options.ImageSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:         "image",
			SystemPrompt: "Reply briefly.",
			UserContent:  "Look at the attached tiny diagnostic image and reply with a short non-empty response.",
			ImagePayload: true,
		})
	}
	if options.ThinkingSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:            "thinking",
			SystemPrompt:    "Think briefly, then reply briefly.",
			UserContent:     "Reply with: xelyon bedrock thinking ok",
			ThinkingEnabled: true,
		})
	}
	return requests
}

func bedrockDiagnosticSmokeRequestsUseToolPayload(requests []bedrockDiagnosticSmokeRequest) bool {
	for _, request := range requests {
		if request.ToolPayload {
			return true
		}
	}
	return false
}

func bedrockDiagnosticSmokeSkipReason(report DiagnosticReport, request bedrockDiagnosticSmokeRequest) (string, bool) {
	if request.ToolPayload && !report.FunctionCallingEnabled {
		return "Bedrock function calling payloads are disabled (BEDROCK_FUNCTION_CALLING=0)", true
	}
	if report.Route == string(bedrockRouteConverseStream) && (request.ImagePayload || request.ThinkingEnabled) {
		return "Bedrock ConverseStream route does not support image or thinking smoke", true
	}
	return "", false
}

func runBedrockDiagnosticSmokeRequest(ctx context.Context, cfg *config.Config, provider *Provider, model string, request bedrockDiagnosticSmokeRequest, output io.Writer) (DiagnosticSmokeRequestResult, error) {
	requestCfg := config.CloneConfig(cfg)
	requestCfg.Thinking.Enabled = request.ThinkingEnabled
	if request.ThinkingEnabled && strings.TrimSpace(requestCfg.Thinking.Level) == "" {
		requestCfg.Thinking.Level = "low"
	}
	requestCtx := newBedrockDiagnosticSmokeRequestContext(ctx, requestCfg, request, output)

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})
	provider.SetMCPTools(nil)

	started := time.Now()
	var content string
	var err error
	if request.ImagePayload {
		content, err = provider.ChatWithImage(requestCtx, request.SystemPrompt, nil, request.UserContent, bedrockDiagnosticImage(), model)
	} else {
		content, err = provider.ChatWithTools(requestCtx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, model)
	}
	elapsed := time.Since(started).Round(time.Millisecond)
	requestID := provider.lastBedrockRequestID()
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(requestCfg, "bedrock", model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:            request.Name,
		Ran:             true,
		ToolPayload:     request.ToolPayload,
		ImagePayload:    request.ImagePayload,
		ThinkingEnabled: request.ThinkingEnabled,
		Content:         strings.TrimSpace(content),
		RequestID:       requestID,
		Duration:        elapsed.String(),
		UsageObserved:   usageObserved,
		Usage:           diagnosticSmokeUsage(usage),
		Cost:            diagnosticSmokeCost(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !diagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s tool call", diagnosticSmokeToolName)
			return result, errors.New(result.Error)
		}
		return result, nil
	}
	if strings.TrimSpace(content) == "" {
		result.Error = fmt.Sprintf("%s smoke response content is empty", request.Name)
		return result, errors.New(result.Error)
	}
	return result, nil
}

func newBedrockDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request bedrockDiagnosticSmokeRequest, output io.Writer) context.Context {
	requestCtx := newBedrockDiagnosticContext(ctx, cfg, output)
	if request.ToolPayload {
		return api.WithToolDefinitions(requestCtx, diagnosticSmokeToolDefinitions())
	}
	requestCtx = api.WithToolDefinitions(requestCtx, nil)
	return api.WithToolUseDisabled(requestCtx)
}

func newBedrockDiagnosticContext(ctx context.Context, cfg *config.Config, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = config.WithContext(requestCtx, cfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	return requestCtx
}

func (r *DiagnosticSmokeResult) addRequestObservation(request DiagnosticSmokeRequestResult) {
	if request.Skipped {
		return
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

func bedrockDiagnosticImage() *api.ImageData {
	const redPNG16x16 = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAAFklEQVR42mP4z8BAEmIY1TCqYfhqAACQ+f8B8u7oVwAAAABJRU5ErkJggg=="
	return &api.ImageData{
		Path:      "bedrock-doctor-red-16x16.png",
		MediaType: "image/png",
		Base64:    redPNG16x16,
		Size:      79,
	}
}

const diagnosticSmokeToolName = "xelyon_bedrock_doctor_probe"

func diagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        diagnosticSmokeToolName,
		Description: "No-op diagnostic probe used to verify Bedrock tool calling.",
		Parameters: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"value"},
		},
	}}
}

func diagnosticSmokeContentHasToolCall(content string) bool {
	return strings.Contains(content, `"tool":"`+diagnosticSmokeToolName+`"`)
}
