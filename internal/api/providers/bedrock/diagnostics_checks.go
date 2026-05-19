package bedrock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

const defaultBedrockDiagnosticAWSAuthTimeout = 10 * time.Second

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

	if options.PrintRequest {
		return
	}

	if !options.requiresAWSAuthCheck() {
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

func (r *DiagnosticReport) addRouteCheck(route bedrockRoute, requestPlan bedrockDiagnosticRequestPlan) {
	switch route {
	case bedrockRouteClaudeMessages:
		r.addCheck(DiagnosticStatusOK, "route", "Bedrock Claude Messages route is selected", r.Route, "")
	case bedrockRouteConverseStream:
		if r.FunctionCallingEnabled && requestPlan.UsesToolPayload() && !llmcatalog.BedrockConverseToolUseSupported(r.Model, r.CatalogModel) {
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

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(DiagnosticStatusOK, "function_calling", "Bedrock function calling payloads are enabled", "", "Set BEDROCK_FUNCTION_CALLING=0 only for text-only troubleshooting")
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "Bedrock function calling payloads are disabled", "BEDROCK_FUNCTION_CALLING=0", "")
}

func (r *DiagnosticReport) runSmokeIfReady(ctx context.Context, cfg *config.Config, options DiagnosticOptions, requestPlan bedrockDiagnosticRequestPlan) {
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

	smoke, err := runBedrockDiagnosticSmoke(ctx, cfg, *r, options, requestPlan)
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
