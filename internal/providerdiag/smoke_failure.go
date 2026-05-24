package providerdiag

import (
	"fmt"
	"strings"
)

// SmokeFailureKind は doctor live smoke 失敗の共通分類を表す。
type SmokeFailureKind string

const (
	SmokeFailureKindAuth               SmokeFailureKind = "auth"
	SmokeFailureKindQuota              SmokeFailureKind = "quota"
	SmokeFailureKindModelUnavailable   SmokeFailureKind = "model_unavailable"
	SmokeFailureKindEndpointMismatch   SmokeFailureKind = "endpoint_mismatch"
	SmokeFailureKindFeatureUnsupported SmokeFailureKind = "feature_unsupported"
	SmokeFailureKindEmptyResponse      SmokeFailureKind = "empty_response"
	SmokeFailureKindGeneric            SmokeFailureKind = "generic"
)

// SmokeFailureFeature は request-level smoke が検証していた capability を表す。
type SmokeFailureFeature string

const (
	SmokeFailureFeatureFunctionCalling SmokeFailureFeature = "function_calling"
	SmokeFailureFeatureImageInput      SmokeFailureFeature = "image_input"
	SmokeFailureFeatureThinking        SmokeFailureFeature = "thinking"
	SmokeFailureFeatureWebSearch       SmokeFailureFeature = "web_search"
)

// SmokeFailureContextOptions は provider 固有の分類 context を表す。
type SmokeFailureContextOptions struct {
	Provider         string
	AuthEnv          string
	EndpointEnv      string
	DebugEnv         string
	EndpointOverride bool
	ModelFlag        string
	ModelLabel       string
}

// SmokeFailureContext は共通分類器への入力。
type SmokeFailureContext struct {
	SmokeFailureContextOptions
	Detail  string
	Feature SmokeFailureFeature
}

// SmokeFailure は provider doctor の smoke failure check に使う分類済み結果。
type SmokeFailure struct {
	Kind       SmokeFailureKind
	Feature    SmokeFailureFeature
	Message    string
	Detail     string
	Suggestion string
}

// ClassifySmokeFailure は provider 横断 vocabulary へ live smoke 失敗を分類する。
func ClassifySmokeFailure(ctx SmokeFailureContext) SmokeFailure {
	ctx.Detail = strings.TrimSpace(ctx.Detail)
	kind := SmokeFailureKindFor(ctx)
	return SmokeFailure{
		Kind:       kind,
		Feature:    ctx.Feature,
		Message:    smokeFailureMessage(ctx, kind),
		Detail:     ctx.Detail,
		Suggestion: smokeFailureSuggestion(ctx, kind),
	}
}

// SmokeFailureKindFor は message/suggestion を組み立てず分類だけ返す。
func SmokeFailureKindFor(ctx SmokeFailureContext) SmokeFailureKind {
	detail := strings.ToLower(strings.TrimSpace(ctx.Detail))
	switch {
	case smokeFailureIsAuth(detail):
		return SmokeFailureKindAuth
	case smokeFailureIsQuota(detail):
		return SmokeFailureKindQuota
	case smokeFailureIsSpecificModelUnavailable(detail):
		return SmokeFailureKindModelUnavailable
	case smokeFailureIsFeatureUnsupported(detail, ctx.Feature):
		return SmokeFailureKindFeatureUnsupported
	case smokeFailureIsEndpointMismatch(detail, ctx.EndpointOverride):
		return SmokeFailureKindEndpointMismatch
	case smokeFailureIsGenericModelUnavailable(detail, ctx.EndpointOverride):
		return SmokeFailureKindModelUnavailable
	case !ctx.EndpointOverride && smokeFailureIsEmptyResponse(detail):
		return SmokeFailureKindEmptyResponse
	default:
		return SmokeFailureKindGeneric
	}
}

// TextToolSmokeFailureContext は TextToolSmokeResult から共通 failure context を作る。
func TextToolSmokeFailureContext(options SmokeFailureContextOptions, smoke TextToolSmokeResult, err error) SmokeFailureContext {
	if request, ok := firstFailedTextToolSmokeRequest(smoke); ok {
		return smokeFailureContextFromRequest(options, textToolSmokeFailureRequest(request))
	}
	return smokeFailureContextFromError(options, err)
}

// MultimodalSmokeFailureContext は ThinkingMultimodalSmokeResult から共通 failure context を作る。
func MultimodalSmokeFailureContext(options SmokeFailureContextOptions, smoke ThinkingMultimodalSmokeResult, err error) SmokeFailureContext {
	if request, ok := firstFailedMultimodalSmokeRequest(smoke.Requests); ok {
		return smokeFailureContextFromRequest(options, multimodalSmokeFailureRequest(request, request.ThinkingPayload))
	}
	return smokeFailureContextFromError(options, err)
}

// BasicMultimodalSmokeFailureContext は MultimodalSmokeResult から共通 failure context を作る。
func BasicMultimodalSmokeFailureContext(options SmokeFailureContextOptions, smoke MultimodalSmokeResult, err error) SmokeFailureContext {
	if request, ok := firstFailedMultimodalSmokeRequest(smoke.Requests); ok {
		return smokeFailureContextFromRequest(options, multimodalSmokeFailureRequest(request, false))
	}
	return smokeFailureContextFromError(options, err)
}

// ResponsesSmokeFailureContext は ResponsesSmokeResult から共通 failure context を作る。
func ResponsesSmokeFailureContext(options SmokeFailureContextOptions, smoke ResponsesSmokeResult, err error) SmokeFailureContext {
	if request, ok := firstFailedResponsesSmokeRequest(smoke); ok {
		return smokeFailureContextFromRequest(options, responsesSmokeFailureRequest(request))
	}
	return smokeFailureContextFromError(options, err)
}

// RoutedResponsesSmokeFailureContext は RoutedResponsesSmokeResult から共通 failure context を作る。
func RoutedResponsesSmokeFailureContext(options SmokeFailureContextOptions, smoke RoutedResponsesSmokeResult, err error) SmokeFailureContext {
	if request, ok := firstFailedRoutedResponsesSmokeRequest(smoke); ok {
		return smokeFailureContextFromRequest(options, routedResponsesSmokeFailureRequest(request))
	}
	return smokeFailureContextFromError(options, err)
}

// KimiSmokeFailureContext は KimiSmokeResult から共通 failure context を作る。
func KimiSmokeFailureContext(options SmokeFailureContextOptions, smoke KimiSmokeResult, err error) SmokeFailureContext {
	if request, ok := firstFailedKimiSmokeRequest(smoke); ok {
		return smokeFailureContextFromRequest(options, kimiSmokeFailureRequest(request))
	}
	return smokeFailureContextFromError(options, err)
}

// InvocationSmokeFailureContext は InvocationSmokeResult から共通 failure context を作る。
func InvocationSmokeFailureContext(options SmokeFailureContextOptions, smoke InvocationSmokeResult, err error) SmokeFailureContext {
	if request, ok := firstFailedInvocationSmokeRequest(smoke); ok {
		return smokeFailureContextFromRequest(options, invocationSmokeFailureRequest(request))
	}
	return smokeFailureContextFromError(options, err)
}

type smokeFailureRequest struct {
	name      string
	route     string
	errText   string
	tool      bool
	image     bool
	thinking  bool
	webSearch bool
}

func smokeFailureContextFromRequest(options SmokeFailureContextOptions, request smokeFailureRequest) SmokeFailureContext {
	return SmokeFailureContext{
		SmokeFailureContextOptions: options,
		Detail:                     smokeFailureRequestDetail(request.name, request.route, request.errText),
		Feature:                    smokeFailureFeature(request.tool, request.image, request.thinking, request.webSearch),
	}
}

func smokeFailureContextFromError(options SmokeFailureContextOptions, err error) SmokeFailureContext {
	return SmokeFailureContext{SmokeFailureContextOptions: options, Detail: smokeFailureErrorDetail(err)}
}

func textToolSmokeFailureRequest(request TextToolSmokeRequestResult) smokeFailureRequest {
	return smokeFailureRequest{
		name:    request.Name,
		route:   request.Route,
		errText: request.Error,
		tool:    request.ToolPayload,
	}
}

func multimodalSmokeFailureRequest(request MultimodalSmokeRequestResult, thinking bool) smokeFailureRequest {
	return smokeFailureRequest{
		name:      request.Name,
		route:     request.Route,
		errText:   request.Error,
		tool:      request.ToolPayload,
		image:     request.ImagePayload,
		thinking:  thinking,
		webSearch: request.WebSearchPayload,
	}
}

func responsesSmokeFailureRequest(request ResponsesSmokeRequestResult) smokeFailureRequest {
	return smokeFailureRequest{
		name:    request.Name,
		errText: request.Error,
		tool:    request.ToolPayload,
	}
}

func routedResponsesSmokeFailureRequest(request RoutedResponsesSmokeRequestResult) smokeFailureRequest {
	return smokeFailureRequest{
		name:    request.Name,
		route:   request.Route,
		errText: request.Error,
		tool:    request.ToolPayload,
	}
}

func kimiSmokeFailureRequest(request KimiSmokeRequestResult) smokeFailureRequest {
	return smokeFailureRequest{
		name:      request.Name,
		errText:   request.Error,
		tool:      request.ToolPayload,
		image:     request.ImagePayload,
		webSearch: request.WebSearchPayload,
	}
}

func invocationSmokeFailureRequest(request InvocationSmokeRequestResult) smokeFailureRequest {
	return smokeFailureRequest{
		name:     request.Name,
		errText:  request.Error,
		tool:     request.ToolPayload,
		image:    request.ImagePayload,
		thinking: request.ThinkingEnabled,
	}
}

func firstFailedTextToolSmokeRequest(smoke TextToolSmokeResult) (TextToolSmokeRequestResult, bool) {
	return firstFailedSmokeRequest(smoke.Requests, func(request TextToolSmokeRequestResult) string { return request.Error })
}

func firstFailedMultimodalSmokeRequest(requests []MultimodalSmokeRequestResult) (MultimodalSmokeRequestResult, bool) {
	return firstFailedSmokeRequest(requests, func(request MultimodalSmokeRequestResult) string { return request.Error })
}

func firstFailedResponsesSmokeRequest(smoke ResponsesSmokeResult) (ResponsesSmokeRequestResult, bool) {
	return firstFailedSmokeRequest(smoke.Requests, func(request ResponsesSmokeRequestResult) string { return request.Error })
}

func firstFailedRoutedResponsesSmokeRequest(smoke RoutedResponsesSmokeResult) (RoutedResponsesSmokeRequestResult, bool) {
	return firstFailedSmokeRequest(smoke.Requests, func(request RoutedResponsesSmokeRequestResult) string { return request.Error })
}

func firstFailedKimiSmokeRequest(smoke KimiSmokeResult) (KimiSmokeRequestResult, bool) {
	return firstFailedSmokeRequest(smoke.Requests, func(request KimiSmokeRequestResult) string { return request.Error })
}

func firstFailedInvocationSmokeRequest(smoke InvocationSmokeResult) (InvocationSmokeRequestResult, bool) {
	return firstFailedSmokeRequest(smoke.Requests, func(request InvocationSmokeRequestResult) string { return request.Error })
}

func firstFailedSmokeRequest[T any](requests []T, errorText func(T) string) (T, bool) {
	for _, request := range requests {
		if strings.TrimSpace(errorText(request)) != "" {
			return request, true
		}
	}
	var zero T
	return zero, false
}

func smokeFailureFeature(tool, image, thinking, webSearch bool) SmokeFailureFeature {
	switch {
	case tool:
		return SmokeFailureFeatureFunctionCalling
	case image:
		return SmokeFailureFeatureImageInput
	case thinking:
		return SmokeFailureFeatureThinking
	case webSearch:
		return SmokeFailureFeatureWebSearch
	default:
		return ""
	}
}

func smokeFailureRequestDetail(name, route, errText string) string {
	parts := []string{}
	if strings.TrimSpace(name) != "" {
		parts = append(parts, "request="+strings.TrimSpace(name))
	}
	if strings.TrimSpace(route) != "" {
		parts = append(parts, "route="+strings.TrimSpace(route))
	}
	if strings.TrimSpace(errText) != "" {
		parts = append(parts, "error="+strings.TrimSpace(errText))
	}
	return strings.Join(parts, " ")
}

func smokeFailureErrorDetail(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func smokeFailureIsAuth(detail string) bool {
	return smokeFailureHasStatus(detail, 401, 403) ||
		smokeFailureContainsAny(detail,
			"unauthorized",
			"forbidden",
			"permission",
			"api key not valid",
			"invalid api key",
			"authentication",
			"authorization",
			"accessdenied",
			"unrecognizedclient",
			"expiredtoken",
			"invalid signature",
		)
}

func smokeFailureIsQuota(detail string) bool {
	return smokeFailureHasStatus(detail, 429, 503) ||
		smokeFailureContainsAny(detail,
			"rate limit",
			"rate_limit",
			"rate-limit",
			"too many requests",
			"too_many_requests",
			"quota",
			"resource_exhausted",
			"capacity",
			"service unavailable",
			"backend overloaded",
			"overloaded",
			"throttl",
			"provisionedthroughputexceeded",
			"servicequotaexceeded",
		)
}

func smokeFailureIsSpecificModelUnavailable(detail string) bool {
	if smokeFailureContainsAny(detail,
		"model not found",
		"model is not found",
		"model was not found",
		"model unavailable",
		"model is unavailable",
		"not found for api version",
		"not supported for this api version",
		"model name",
		"not a valid model",
		"model identifier is invalid",
		"deployment not found",
		"deployment does not exist",
		"deployment is not available",
	) {
		return true
	}
	return (strings.Contains(detail, "model") && strings.Contains(detail, "does not exist")) ||
		(strings.Contains(detail, "deployment") && smokeFailureContainsAny(detail, "does not exist", "not found"))
}

func smokeFailureIsGenericModelUnavailable(detail string, endpointOverride bool) bool {
	if endpointOverride {
		return false
	}
	return smokeFailureHasStatus(detail, 404) ||
		smokeFailureContainsAny(detail,
			"does not exist",
			"resource not found",
		)
}

func smokeFailureIsFeatureUnsupported(detail string, feature SmokeFailureFeature) bool {
	if feature == "" {
		return false
	}
	if smokeFailureContainsAny(detail, "unsupported", "not supported", "not accepted") {
		return true
	}
	switch feature {
	case SmokeFailureFeatureFunctionCalling:
		return smokeFailureContainsAny(detail, "tool smoke response did not include", "function_call", "function call", "functioncalling", "tool_config", "tool use")
	case SmokeFailureFeatureImageInput:
		return smokeFailureContainsAny(detail, "inline_data", "mime_type", "image input", "image data", "base64 image")
	case SmokeFailureFeatureThinking:
		return smokeFailureContainsAny(detail, "thinking", "extended thinking", "reasoning")
	case SmokeFailureFeatureWebSearch:
		return smokeFailureContainsAny(detail, "web search smoke response did not include", "web search", "google_search", "google search", "grounding", "$web_search")
	default:
		return false
	}
}

func smokeFailureIsEmptyResponse(detail string) bool {
	return smokeFailureContainsAny(detail, "no content", "empty response", "stream ended without", "response content is empty", "content is empty")
}

func smokeFailureIsEndpointMismatch(detail string, endpointOverride bool) bool {
	if !endpointOverride {
		return false
	}
	return smokeFailureHasStatus(detail, 404) ||
		smokeFailureContainsAny(detail,
			"failed to decode response",
			"invalid character",
			"empty response body",
			"streamgeneratecontent",
			"alt=sse",
			"generatecontent",
			"chat/completions",
			"messages endpoint",
			"endpoint mismatch",
			"route not found",
			"route does not exist",
			"resource not found",
		) ||
		smokeFailureIsEmptyResponse(detail)
}

func smokeFailureHasStatus(text string, statuses ...int) bool {
	for _, status := range statuses {
		if strings.Contains(text, fmt.Sprintf("status %d", status)) ||
			strings.Contains(text, fmt.Sprintf("api error (%d)", status)) ||
			strings.Contains(text, fmt.Sprintf("(%d)", status)) ||
			strings.Contains(text, fmt.Sprintf("http %d", status)) {
			return true
		}
	}
	return false
}

func smokeFailureContainsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func smokeFailureMessage(ctx SmokeFailureContext, kind SmokeFailureKind) string {
	provider := smokeFailureProvider(ctx.Provider)
	switch kind {
	case SmokeFailureKindAuth:
		return fmt.Sprintf("live %s smoke authentication or authorization failed", provider)
	case SmokeFailureKindQuota:
		return fmt.Sprintf("live %s smoke hit quota, rate limit, or capacity", provider)
	case SmokeFailureKindModelUnavailable:
		return fmt.Sprintf("live %s smoke model is unavailable", provider)
	case SmokeFailureKindEndpointMismatch:
		return fmt.Sprintf("live %s smoke endpoint does not match the selected request route", provider)
	case SmokeFailureKindFeatureUnsupported:
		return fmt.Sprintf("%s %s smoke was not accepted by the selected model or endpoint", provider, smokeFailureFeatureLabel(ctx.Feature))
	case SmokeFailureKindEmptyResponse:
		return fmt.Sprintf("live %s smoke response was empty", provider)
	default:
		return fmt.Sprintf("live %s smoke request failed", provider)
	}
}

func smokeFailureSuggestion(ctx SmokeFailureContext, kind SmokeFailureKind) string {
	provider := smokeFailureProvider(ctx.Provider)
	switch kind {
	case SmokeFailureKindAuth:
		if strings.TrimSpace(ctx.AuthEnv) != "" {
			return fmt.Sprintf("Check %s, credential permissions, and that the credential can access the selected %s model", ctx.AuthEnv, provider)
		}
		return fmt.Sprintf("Check credentials, permissions, and access to the selected %s model", provider)
	case SmokeFailureKindQuota:
		return fmt.Sprintf("Check %s quota, rate limits, and capacity for the selected model, honor Retry-After if present, or rerun later", provider)
	case SmokeFailureKindModelUnavailable:
		suggestion := fmt.Sprintf(
			"Use %s with an available %s %s; keep --catalog-model for alias token/pricing policy",
			smokeFailureModelFlag(ctx),
			provider,
			smokeFailureModelLabel(ctx),
		)
		if strings.TrimSpace(ctx.EndpointEnv) != "" {
			suggestion += fmt.Sprintf("; check %s if the %s exists", strings.TrimSpace(ctx.EndpointEnv), smokeFailureModelLabel(ctx))
		}
		return suggestion
	case SmokeFailureKindEndpointMismatch:
		if strings.TrimSpace(ctx.EndpointEnv) != "" {
			return fmt.Sprintf("Run --print-request and check %s or the configured proxy accepts the selected route", ctx.EndpointEnv)
		}
		return "Run --print-request and check that the configured endpoint accepts the selected route"
	case SmokeFailureKindFeatureUnsupported:
		return fmt.Sprintf("Use a model and endpoint with %s support, or rerun without the related smoke flag", smokeFailureFeatureLabel(ctx.Feature))
	case SmokeFailureKindEmptyResponse:
		return fmt.Sprintf("Retry, change %s if it repeats, or inspect --print-request for the request shape", smokeFailureModelFlag(ctx))
	default:
		suggestion := "Inspect the request-level smoke error and rerun with --print-request"
		if strings.TrimSpace(ctx.DebugEnv) != "" {
			suggestion += "; set " + strings.TrimSpace(ctx.DebugEnv) + "=1 when raw error details are needed"
		}
		return suggestion
	}
}

func smokeFailureModelFlag(ctx SmokeFailureContext) string {
	flag := strings.TrimSpace(ctx.ModelFlag)
	if flag == "" {
		return "--model"
	}
	return flag
}

func smokeFailureModelLabel(ctx SmokeFailureContext) string {
	label := strings.TrimSpace(ctx.ModelLabel)
	if label == "" {
		return "model"
	}
	return label
}

func smokeFailureProvider(provider string) string {
	if strings.TrimSpace(provider) == "" {
		return "provider"
	}
	return strings.TrimSpace(provider)
}

func smokeFailureFeatureLabel(feature SmokeFailureFeature) string {
	switch feature {
	case SmokeFailureFeatureFunctionCalling:
		return "function calling"
	case SmokeFailureFeatureImageInput:
		return "image input"
	case SmokeFailureFeatureThinking:
		return "thinking"
	case SmokeFailureFeatureWebSearch:
		return "web search"
	default:
		return "feature"
	}
}
