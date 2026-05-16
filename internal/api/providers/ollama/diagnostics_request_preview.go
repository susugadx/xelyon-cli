package ollama

import (
	"context"
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildOllamaDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "Ollama request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"Ollama request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildOllamaDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultOllamaDiagnosticSmokeMaxOutputTokens
	}

	previewCfg := ollamaDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	provider := New(report.APIURL)
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	preview := DiagnosticRequestPreview{}
	for _, request := range ollamaDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, newOllamaDiagnosticSkippedToolPreviewRequest(request, report.Route))
			continue
		}

		requestCtx := newOllamaDiagnosticSmokeRequestContext(ctx, previewCfg, request, io.Discard)
		if request.ToolPayload {
			provider.SetToolChoice(ollamaDiagnosticSmokeToolName)
		} else {
			provider.ClearToolChoice()
		}
		preview.Requests = append(preview.Requests, buildOllamaDiagnosticRequestPreviewRequest(requestCtx, provider, report, request))
	}
	return preview, nil
}

func buildOllamaDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request ollamaDiagnosticSmokeRequest,
) DiagnosticRequestPreviewRequest {
	build := provider.buildChatRequest(
		ctx,
		request.SystemPrompt,
		[]api.Message{{Role: "user", Content: request.UserContent}},
		report.Model,
	)
	return DiagnosticRequestPreviewRequest{
		Name:        request.Name,
		ToolPayload: request.ToolPayload,
		Route:       report.Route,
		Method:      "POST",
		URL:         build.URL,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Body:        build.Request,
	}
}
