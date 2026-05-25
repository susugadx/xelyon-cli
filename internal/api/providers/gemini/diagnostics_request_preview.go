package gemini

import (
	"context"
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
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
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, providerdiag.NewSkippedMultimodalPreviewRequest(
				request.multimodalSmokeRequest(),
				"Gemini function calling is not supported for the resolved catalog_model",
			))
			continue
		}
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
	body := buildGeminiDiagnosticRequestBody(ctx, provider, report, request, cfg)
	return providerdiag.NewMultimodalRequestPreview(request.multimodalSmokeRequest(), providerdiag.RequestPreviewTransport{
		Method:  "POST",
		URL:     geminiDiagnosticRequestURL(report.Model, request),
		Headers: redactedGeminiHeaders(),
		Body:    body,
	})
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
			report.FunctionCallingEnabled,
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
	return providerdiag.RedactedAPIKeyHeaders("x-goog-api-key")
}
