package gemini

import (
	"context"
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildGeminiDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "Gemini request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"Gemini request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildGeminiDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultGeminiDiagnosticSmokeMaxOutputTokens
	}

	previewCfg := geminiDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	provider := New("diagnostic-key")
	provider.SetMCPTools(nil)

	preview := DiagnosticRequestPreview{}
	for _, request := range geminiDiagnosticRequests(options) {
		requestCtx := newGeminiDiagnosticRequestContext(ctx, previewCfg, request, io.Discard)
		preview.Requests = append(preview.Requests, buildGeminiDiagnosticRequestPreviewRequest(requestCtx, provider, report, request))
	}
	return preview, nil
}

func buildGeminiDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request geminiDiagnosticRequest,
) DiagnosticRequestPreviewRequest {
	cfg := config.FromContext(ctx)
	route := geminiDiagnosticRequestRoute(request)
	body := buildGeminiDiagnosticRequestBody(ctx, provider, report, request, cfg)
	return DiagnosticRequestPreviewRequest{
		Name:             request.Name,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		WebSearchPayload: request.WebSearchPayload,
		Route:            route,
		Method:           "POST",
		URL:              geminiDiagnosticRequestURL(report.Model, request),
		Headers:          redactedGeminiHeaders(),
		Body:             body,
	}
}

func buildGeminiDiagnosticRequestBody(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request geminiDiagnosticRequest,
	cfg *config.Config,
) any {
	switch {
	case request.ToolPayload:
		toolDefs := GetCombinedToolDefinitionsWithContext(ctx, nil)
		toolCfg := newGeminiToolConfig(geminiFunctionCallingMode(ctx))
		return buildGeminiFunctionCallingRequest(
			ctx,
			request.SystemPrompt,
			[]api.Message{{Role: "user", Content: request.UserContent}},
			report.Model,
			"",
			toolDefs,
			toolCfg,
			cfg,
		)
	case request.ImagePayload:
		return buildGeminiMultimodalRequest(
			ctx,
			request.SystemPrompt,
			nil,
			request.UserContent,
			geminiDiagnosticImage(),
			report.Model,
			nil,
			provider.IsFunctionCallingEnabled(),
			cfg,
		)
	case request.WebSearchPayload:
		return buildGeminiWebSearchRequest(ctx, request.UserContent, report.Model, cfg)
	default:
		return buildGeminiTextRequest(
			ctx,
			request.SystemPrompt,
			[]api.Message{{Role: "user", Content: request.UserContent}},
			report.Model,
			"",
			cfg,
		)
	}
}

func redactedGeminiHeaders() map[string]string {
	return map[string]string{
		"Content-Type":   "application/json",
		"x-goog-api-key": "<redacted>",
	}
}
