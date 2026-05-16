package claude

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildClaudeDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "Claude request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"Claude request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildClaudeDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := claudeDiagnosticRequestMaxOutputTokens(options)

	previewCfg := claudeDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	provider := New("<redacted>")
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	preview := DiagnosticRequestPreview{}
	plan := buildClaudeDiagnosticRequestPlan(options, report.FunctionCallingEnabled)
	for _, request := range plan.Requests {
		if request.skipped(report.FunctionCallingEnabled) {
			preview.Requests = append(preview.Requests, newClaudeDiagnosticSkippedToolPreviewRequest(request))
			continue
		}
		requestCtx := newClaudeDiagnosticRequestContext(ctx, previewCfg, request, io.Discard)
		previewRequest, err := buildClaudeDiagnosticRequestPreviewRequest(requestCtx, provider, report, request)
		if err != nil {
			return preview, err
		}
		preview.Requests = append(preview.Requests, previewRequest)
	}
	return preview, nil
}

func buildClaudeDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request claudeDiagnosticRequest,
) (DiagnosticRequestPreviewRequest, error) {
	body, contextManagement := buildClaudeDiagnosticRequestBody(ctx, provider, report.Model, request)
	headers := provider.anthropicHeaders(ctx, report.Model, contextManagement)
	if request.WebSearchPayload {
		headers = provider.anthropicHeaders(ctx, report.Model, nil, webSearchBetaHeader)
	}
	result := request.previewBase()
	result.Method = "POST"
	result.URL = report.APIURL
	result.Headers = redactedClaudeHeaders(headers)
	result.Body = body
	return result, nil
}

func buildClaudeDiagnosticRequestBody(ctx context.Context, provider *Provider, model string, request claudeDiagnosticRequest) (any, *ContextManagement) {
	applyClaudeDiagnosticToolChoice(provider, request)

	switch {
	case request.ImagePayload:
		built := provider.buildMultimodalRequest(ctx, request.SystemPrompt, nil, request.UserContent, claudeDiagnosticImage(), model)
		return built.Request, built.Request.ContextManagement
	case request.WebSearchPayload:
		built := provider.buildWebSearchRequest(ctx, request.UserContent, model)
		return built.Request, nil
	default:
		built := provider.buildMessagesRequest(
			ctx,
			request.SystemPrompt,
			[]api.Message{{Role: "user", Content: request.UserContent}},
			model,
		)
		return built.Request, built.Request.ContextManagement
	}
}

func redactedClaudeHeaders(headers http.Header) map[string]string {
	result := map[string]string{
		"Content-Type":      headers.Get("Content-Type"),
		"x-api-key":         "<redacted>",
		"anthropic-version": headers.Get("anthropic-version"),
	}
	if beta := headers.Get("anthropic-beta"); beta != "" {
		result["anthropic-beta"] = beta
	}
	return result
}
