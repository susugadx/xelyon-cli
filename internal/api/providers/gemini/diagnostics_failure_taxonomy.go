package gemini

import (
	"fmt"
	"os"
	"strings"
)

type geminiDiagnosticSmokeFailureKind string

const (
	geminiDiagnosticSmokeFailureKindAuth          geminiDiagnosticSmokeFailureKind = "auth"
	geminiDiagnosticSmokeFailureKindCapacity      geminiDiagnosticSmokeFailureKind = "capacity"
	geminiDiagnosticSmokeFailureKindModel         geminiDiagnosticSmokeFailureKind = "model"
	geminiDiagnosticSmokeFailureKindTool          geminiDiagnosticSmokeFailureKind = "tool"
	geminiDiagnosticSmokeFailureKindImage         geminiDiagnosticSmokeFailureKind = "image"
	geminiDiagnosticSmokeFailureKindWebSearch     geminiDiagnosticSmokeFailureKind = "web_search"
	geminiDiagnosticSmokeFailureKindEmptyResponse geminiDiagnosticSmokeFailureKind = "empty_response"
	geminiDiagnosticSmokeFailureKindEndpoint      geminiDiagnosticSmokeFailureKind = "endpoint"
	geminiDiagnosticSmokeFailureKindGeneric       geminiDiagnosticSmokeFailureKind = "generic"
)

type geminiDiagnosticEndpointFailureContext struct {
	OverrideURL string
}

func geminiDiagnosticEndpointFailureContextFromEnv() geminiDiagnosticEndpointFailureContext {
	return geminiDiagnosticEndpointFailureContext{OverrideURL: strings.TrimSpace(os.Getenv(geminiAPIURLEnv))}
}

func (c geminiDiagnosticEndpointFailureContext) HasOverride() bool {
	return strings.TrimSpace(c.OverrideURL) != ""
}

type geminiDiagnosticSmokeFailureContext struct {
	Detail           string
	LowerDetail      string
	FailedRequest    DiagnosticSmokeRequestResult
	HasFailedRequest bool
	Endpoint         geminiDiagnosticEndpointFailureContext
}

type geminiDiagnosticSmokeFailureRule struct {
	kind    geminiDiagnosticSmokeFailureKind
	matches func(geminiDiagnosticSmokeFailureContext) bool
}

var geminiDiagnosticSmokeFailureRules = []geminiDiagnosticSmokeFailureRule{
	{kind: geminiDiagnosticSmokeFailureKindAuth, matches: geminiDiagnosticSmokeFailureIsAuth},
	{kind: geminiDiagnosticSmokeFailureKindCapacity, matches: geminiDiagnosticSmokeFailureIsCapacity},
	{kind: geminiDiagnosticSmokeFailureKindModel, matches: geminiDiagnosticSmokeFailureIsModel},
	{kind: geminiDiagnosticSmokeFailureKindTool, matches: geminiDiagnosticSmokeFailureIsTool},
	{kind: geminiDiagnosticSmokeFailureKindImage, matches: geminiDiagnosticSmokeFailureIsImage},
	{kind: geminiDiagnosticSmokeFailureKindWebSearch, matches: geminiDiagnosticSmokeFailureIsWebSearch},
	{kind: geminiDiagnosticSmokeFailureKindEmptyResponse, matches: geminiDiagnosticSmokeFailureIsEmptyResponse},
	{kind: geminiDiagnosticSmokeFailureKindEndpoint, matches: geminiDiagnosticSmokeFailureIsEndpoint},
}

func geminiDiagnosticSmokeFailureKindFor(ctx geminiDiagnosticSmokeFailureContext) geminiDiagnosticSmokeFailureKind {
	for _, rule := range geminiDiagnosticSmokeFailureRules {
		if rule.matches(ctx) {
			return rule.kind
		}
	}
	return geminiDiagnosticSmokeFailureKindGeneric
}

func newGeminiDiagnosticSmokeFailure(kind geminiDiagnosticSmokeFailureKind, detail string) geminiDiagnosticSmokeFailure {
	switch kind {
	case geminiDiagnosticSmokeFailureKindAuth:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "live Gemini smoke authentication or authorization failed",
			Detail:     detail,
			Suggestion: fmt.Sprintf("Check %s, API key permissions, and that the key belongs to a project with Gemini API access", geminiAPIKeyEnv),
		}
	case geminiDiagnosticSmokeFailureKindCapacity:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "live Gemini smoke hit quota, rate limit, or capacity",
			Detail:     detail,
			Suggestion: "Check Gemini API quota and rate limits for the selected model, honor Retry-After if present, or rerun later",
		}
	case geminiDiagnosticSmokeFailureKindModel:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "live Gemini smoke model is unavailable",
			Detail:     detail,
			Suggestion: "Use --model with an available Gemini model; keep --catalog-model for alias token/pricing policy",
		}
	case geminiDiagnosticSmokeFailureKindTool:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "Gemini tool smoke was not accepted by the selected model or endpoint",
			Detail:     detail,
			Suggestion: "Use a Gemini model and endpoint with function calling support, then rerun --tool-smoke --print-request to inspect the diagnostic ANY payload",
		}
	case geminiDiagnosticSmokeFailureKindImage:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "Gemini image smoke was not accepted by the selected model or endpoint",
			Detail:     detail,
			Suggestion: "Use a Gemini model and endpoint with inline_data image input support, or rerun without --image-smoke",
		}
	case geminiDiagnosticSmokeFailureKindWebSearch:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "Gemini native web search smoke was not accepted by the selected model or endpoint",
			Detail:     detail,
			Suggestion: "Use a Gemini model and endpoint that support native google_search on generateContent, or rerun without --web-search-smoke",
		}
	case geminiDiagnosticSmokeFailureKindEmptyResponse:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "live Gemini smoke response was empty",
			Detail:     detail,
			Suggestion: "The selected Gemini model returned an SSE stream without text or tool calls; retry, change --model if it repeats, or inspect --print-request for the request shape",
		}
	case geminiDiagnosticSmokeFailureKindEndpoint:
		return geminiDiagnosticSmokeFailure{
			Kind:       kind,
			Message:    "live Gemini smoke endpoint does not match the selected request route",
			Detail:     detail,
			Suggestion: fmt.Sprintf("Run --print-request and check %s: text/tool/image need streamGenerateContent?alt=sse; web search needs generateContent or a proxy that accepts both shapes", geminiAPIURLEnv),
		}
	default:
		return geminiDiagnosticSmokeFailure{
			Kind:       geminiDiagnosticSmokeFailureKindGeneric,
			Message:    "live Gemini smoke request failed",
			Detail:     detail,
			Suggestion: "Inspect the request-level smoke error and rerun with --print-request; set XELYON_DEBUG_GEMINI=1 when raw Gemini error details are needed",
		}
	}
}

func geminiDiagnosticSmokeFailureIsAuth(ctx geminiDiagnosticSmokeFailureContext) bool {
	return geminiDiagnosticErrorHasStatus(ctx.LowerDetail, 401, 403) ||
		geminiDiagnosticErrorContainsAny(ctx.LowerDetail, "unauthorized", "forbidden", "permission", "api key not valid", "invalid api key")
}

func geminiDiagnosticSmokeFailureIsCapacity(ctx geminiDiagnosticSmokeFailureContext) bool {
	return geminiDiagnosticErrorHasStatus(ctx.LowerDetail, 429, 503) ||
		geminiDiagnosticErrorContainsAny(ctx.LowerDetail, "rate limit", "quota", "resource_exhausted", "capacity", "service unavailable", "backend overloaded", "overloaded")
}

func geminiDiagnosticSmokeFailureIsModel(ctx geminiDiagnosticSmokeFailureContext) bool {
	if geminiDiagnosticErrorContainsAny(ctx.LowerDetail, "model not found", "model is not found", "not found for api version", "not supported for this api version", "model name") {
		return true
	}
	return !ctx.Endpoint.HasOverride() && geminiDiagnosticErrorHasStatus(ctx.LowerDetail, 404)
}

func geminiDiagnosticSmokeFailureIsTool(ctx geminiDiagnosticSmokeFailureContext) bool {
	return ctx.HasFailedRequest && ctx.FailedRequest.ToolPayload && geminiDiagnosticErrorContainsAny(
		ctx.LowerDetail,
		"tool smoke response did not include",
		"function_call",
		"function call",
		"functioncalling",
		"tool_config",
		"tool use",
	)
}

func geminiDiagnosticSmokeFailureIsImage(ctx geminiDiagnosticSmokeFailureContext) bool {
	return ctx.HasFailedRequest && ctx.FailedRequest.ImagePayload &&
		geminiDiagnosticErrorContainsAny(ctx.LowerDetail, "inline_data", "mime_type", "image input", "image data")
}

func geminiDiagnosticSmokeFailureIsWebSearch(ctx geminiDiagnosticSmokeFailureContext) bool {
	return ctx.HasFailedRequest && ctx.FailedRequest.WebSearchPayload && geminiDiagnosticErrorContainsAny(
		ctx.LowerDetail,
		"web search smoke response did not include",
		"web search",
		"google_search",
		"google search",
		"grounding",
	)
}

func geminiDiagnosticSmokeFailureIsEmptyResponse(ctx geminiDiagnosticSmokeFailureContext) bool {
	return !ctx.Endpoint.HasOverride() && geminiDiagnosticEmptySSEResponseSmokeFailure(ctx.LowerDetail)
}

func geminiDiagnosticSmokeFailureIsEndpoint(ctx geminiDiagnosticSmokeFailureContext) bool {
	if !ctx.Endpoint.HasOverride() {
		return false
	}
	if geminiDiagnosticErrorHasStatus(ctx.LowerDetail, 404) {
		return true
	}
	if geminiDiagnosticErrorContainsAny(ctx.LowerDetail, "failed to decode response", "invalid character") {
		return true
	}
	if geminiDiagnosticEmptySSEResponseSmokeFailure(ctx.LowerDetail) {
		return true
	}
	return geminiDiagnosticErrorContainsAny(ctx.LowerDetail, "empty response body", "streamgeneratecontent", "alt=sse", "generatecontent")
}

func geminiDiagnosticErrorHasStatus(text string, statuses ...int) bool {
	for _, status := range statuses {
		if strings.Contains(text, fmt.Sprintf("status %d", status)) ||
			strings.Contains(text, fmt.Sprintf("api error (%d)", status)) ||
			strings.Contains(text, fmt.Sprintf("(%d)", status)) {
			return true
		}
	}
	return false
}

func geminiDiagnosticErrorContainsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func geminiDiagnosticEmptySSEResponseSmokeFailure(text string) bool {
	return geminiDiagnosticErrorContainsAny(text, "no content in gemini sse response", "stream ended without")
}
