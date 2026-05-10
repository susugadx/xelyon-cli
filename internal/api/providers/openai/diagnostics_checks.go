package openai

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

const (
	openAIAPIKeyEnv       = "OPENAI_API_KEY"
	openAIAPIURLEnv       = "OPENAI_API_URL"
	openAIResponsesURLEnv = "OPENAI_RESPONSES_URL"
)

func (r *DiagnosticReport) addCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	r.Checks = append(r.Checks, DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     detail,
		Suggestion: suggestion,
	})
}

func (r *DiagnosticReport) addAuthCheck() {
	if strings.TrimSpace(os.Getenv(openAIAPIKeyEnv)) == "" {
		r.addCheck(
			DiagnosticStatusFail,
			"auth",
			fmt.Sprintf("%s is required", openAIAPIKeyEnv),
			"",
			fmt.Sprintf("Set %s before running OpenAI requests", openAIAPIKeyEnv),
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "auth", "OpenAI API key is configured", openAIAPIKeyEnv, "")
}

func (r *DiagnosticReport) addAPIURLCheck() {
	r.addEndpointURLCheck(
		"api_url",
		openAIAPIURLEnv,
		r.APIURL,
		defaultOpenAIURL,
		"/v1/chat/completions",
		r.Route == DiagnosticRouteChatCompletions,
	)
}

func (r *DiagnosticReport) addResponsesURLCheck() {
	r.addEndpointURLCheck(
		"responses_url",
		openAIResponsesURLEnv,
		r.ResponsesURL,
		defaultOpenAIResponsesURL,
		"/v1/responses",
		r.Route != DiagnosticRouteChatCompletions,
	)
}

func (r *DiagnosticReport) addEndpointURLCheck(name, envName, endpoint, defaultEndpoint, expectedPath string, activeRoute bool) {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		r.addCheck(DiagnosticStatusOK, name, fmt.Sprintf("%s uses the built-in endpoint", envName), defaultEndpoint, "")
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		status := DiagnosticStatusWarn
		suggestion := fmt.Sprintf("Set %s to a valid absolute URL such as %s", envName, defaultEndpoint)
		if activeRoute {
			status = DiagnosticStatusFail
		}
		r.addCheck(status, name, fmt.Sprintf("%s is not a valid absolute URL", envName), raw, suggestion)
		return
	}

	if strings.TrimRight(parsed.Path, "/") != expectedPath {
		r.addCheck(
			DiagnosticStatusWarn,
			name+"_path",
			fmt.Sprintf("%s does not end with %s", envName, expectedPath),
			raw,
			"This is OK only for an intentional proxy endpoint",
		)
	}
	r.addCheck(DiagnosticStatusOK, name, fmt.Sprintf("%s is configured", envName), endpoint, "")
}

func (r *DiagnosticReport) addProviderRegistrationCheck() {
	if api.IsRegisteredProvider("openai") {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "openai provider is registered", "", "")
		return
	}
	r.addCheck(DiagnosticStatusFail, "provider_registration", "openai provider is not registered", "", "Ensure providers/all imports the OpenAI provider")
}

func (r *DiagnosticReport) addModelConfigCheck() {
	if strings.TrimSpace(r.Model) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "OpenAI model is not resolved", "", "Pass --model gpt-5.4 or set provider_models.openai.default_model")
		return
	}
	if strings.TrimSpace(r.CatalogModel) == "" {
		r.addCheck(DiagnosticStatusFail, "model", "OpenAI catalog model is not resolved", r.Model, "Use a known OpenAI model such as gpt-5.4")
		return
	}
	if !looksLikeOpenAICatalogModel(r.CatalogModel) {
		r.addCheck(
			DiagnosticStatusWarn,
			"model",
			"resolved catalog model does not look like an OpenAI model",
			fmt.Sprintf("model=%s catalog_model=%s", r.Model, r.CatalogModel),
			"Set --catalog-model or provider_models.openai.catalog_model to the underlying OpenAI model",
		)
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"model",
		"OpenAI model config is resolved",
		fmt.Sprintf("%s (%s), catalog_model=%s (%s)", r.Model, r.ModelSource, r.CatalogModel, r.CatalogModelSource),
		"",
	)
}

func (r *DiagnosticReport) addRouteCheck() {
	detail := r.routeCheckDetail()
	switch r.Route {
	case DiagnosticRouteResponsesStreaming:
		r.addCheck(DiagnosticStatusOK, "route", "OpenAI Responses streaming route is selected", detail, "")
	case DiagnosticRouteResponsesNonStreaming:
		r.addCheck(DiagnosticStatusOK, "route", "OpenAI Responses non-streaming route is selected", detail, "")
	case DiagnosticRouteChatCompletions:
		r.addCheck(DiagnosticStatusOK, "route", "OpenAI Chat Completions route is selected", detail, "")
	default:
		r.addCheck(DiagnosticStatusFail, "route", "OpenAI route could not be resolved", detail, "")
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
	model := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if model == "" || catalogModel == "" {
		return
	}

	contextWindow, contextOK := llmcatalog.KnownModelContextLimit(catalogModel)
	maxOutput, maxOutputOK := openAIDiagnosticMaxOutputTokens(cfg, model, catalogModel)
	pricing := cost.GetPricingInfoForConfig(cfg, "openai", model)
	detail := fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, %s",
		catalogModel,
		openAIDiagnosticIntDetail(contextWindow, contextOK),
		openAIDiagnosticIntDetail(maxOutput, maxOutputOK),
		openAIDiagnosticPricingDetail(pricing),
	)

	switch {
	case !contextOK:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use an OpenAI model known to XELYON before relying on token-limit diagnostics")
	case !maxOutputOK:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Use an OpenAI model known to XELYON, or set max_output_tokens explicitly for this model")
	case pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing pricing metadata", detail, "Use an OpenAI model with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func (r *DiagnosticReport) addFunctionCallingCheck() {
	if r.FunctionCallingEnabled {
		r.addCheck(
			DiagnosticStatusOK,
			"function_calling",
			"OpenAI function calling payloads are enabled",
			"",
			"Set OPENAI_FUNCTION_CALLING=0 only if the selected endpoint rejects tool payloads",
		)
		return
	}
	r.addCheck(DiagnosticStatusOK, "function_calling", "OpenAI function calling payloads are disabled", "OPENAI_FUNCTION_CALLING=0", "")
}

func (r *DiagnosticReport) addResponsesRetentionCheck() {
	message := fmt.Sprintf("responses.store=%t, responses.persist_response_id=%t", r.ResponsesStore, r.ResponsesPersistResponseID)
	if !r.ResponsesStore || !r.ResponsesPersistResponseID {
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
			"OPENAI_FUNCTION_CALLING=0",
			"Unset OPENAI_FUNCTION_CALLING or set it to 1 before rerunning --tool-smoke",
		)
	}
	if options.RetentionSmoke && r.Route == DiagnosticRouteChatCompletions {
		r.addCheck(
			DiagnosticStatusFail,
			"retention_smoke",
			"OpenAI Responses retention smoke is not supported on the Chat Completions route",
			r.Route,
			"Use a Responses API model before rerunning --retention-smoke",
		)
		return
	}

	smoke, err := runOpenAIDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &smoke
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "smoke", "live OpenAI smoke request failed", err.Error(), "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "live OpenAI smoke request succeeded", smoke.Duration, "")
	r.addSmokeObservationChecks(smoke)
	if smoke.ToolPayload {
		r.addCheck(DiagnosticStatusOK, "tool_smoke", "OpenAI endpoint accepted a tool payload", smoke.Duration, "")
	}
	if smoke.RetentionPayload {
		r.addCheck(DiagnosticStatusOK, "retention_smoke", "OpenAI endpoint accepted a previous_response_id chain", smoke.Duration, "")
	}
}

func (r *DiagnosticReport) addSmokeObservationChecks(smoke DiagnosticSmokeResult) {
	if smoke.Route == DiagnosticRouteChatCompletions {
		if strings.TrimSpace(smoke.ResponseID) == "" {
			r.addCheck(DiagnosticStatusOK, "response_id", "Chat Completions smoke does not return a Responses response ID", "", "")
		}
	} else if strings.TrimSpace(smoke.ResponseID) != "" {
		r.addCheck(DiagnosticStatusOK, "response_id", "OpenAI smoke returned a response ID", smoke.ResponseID, "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"response_id",
			"OpenAI smoke succeeded but response ID was not returned",
			"",
			"Check whether the endpoint returns Responses API response.created/id metadata",
		)
	}

	if smoke.UsageObserved {
		r.addCheck(DiagnosticStatusOK, "usage", "OpenAI smoke usage was observed", openAIDiagnosticSmokeUsageDetail(smoke.Usage), "")
	} else {
		r.addCheck(
			DiagnosticStatusWarn,
			"usage",
			"OpenAI smoke succeeded but usage was not observed",
			"",
			"Check whether the endpoint returns usage metadata",
		)
	}

	switch {
	case !smoke.UsageObserved:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"OpenAI smoke cost estimate was skipped because usage was not observed",
			"",
			"Rerun smoke after usage metadata is available",
		)
	case smoke.Cost.PricingUnavailable:
		r.addCheck(
			DiagnosticStatusWarn,
			"cost",
			"OpenAI smoke cost pricing is unavailable",
			"",
			"Use an OpenAI catalog model with pricing metadata before relying on smoke cost estimates",
		)
	default:
		r.addCheck(DiagnosticStatusOK, "cost", "OpenAI smoke cost estimate is available", fmt.Sprintf("$%.8f USD", smoke.Cost.USD), "")
	}
}

func openAIDiagnosticSmokeUsageDetail(usage DiagnosticSmokeUsage) string {
	return fmt.Sprintf(
		"input_tokens=%d, cached_input_tokens=%d, output_tokens=%d, thinking_tokens=%d, cache_creation_tokens=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}
