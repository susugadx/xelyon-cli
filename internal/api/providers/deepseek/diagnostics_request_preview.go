package deepseek

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildDeepSeekDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "DeepSeek request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"DeepSeek request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildDeepSeekDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultDeepSeekDiagnosticSmokeMaxOutputTokens
	}

	previewCfg := deepSeekDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	provider := New(os.Getenv(deepSeekAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	preview := DiagnosticRequestPreview{}
	for _, request := range deepSeekDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, DiagnosticRequestPreviewRequest{
				Name:        request.Name,
				Skipped:     true,
				SkipReason:  fmt.Sprintf("DeepSeek function calling payloads are disabled (%s=0)", deepSeekFunctionCallingEnv),
				ToolPayload: true,
				Route:       report.Route,
			})
			continue
		}

		requestCtx := newDeepSeekDiagnosticSmokeRequestContext(ctx, previewCfg, request, io.Discard)
		if request.ToolPayload {
			provider.SetToolChoice(deepSeekDiagnosticSmokeToolName)
		} else {
			provider.ClearToolChoice()
		}
		preview.Requests = append(preview.Requests, buildDeepSeekDiagnosticRequestPreviewRequest(requestCtx, provider, report, request))
	}
	return preview, nil
}

func buildDeepSeekDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request deepSeekDiagnosticSmokeRequest,
) DiagnosticRequestPreviewRequest {
	body, _ := provider.buildChatCompletionsRequest(
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
		URL:         report.APIURL,
		Headers:     deepSeekDiagnosticRequestPreviewHeaders(),
		Body:        body,
	}
}

func deepSeekDiagnosticRequestPreviewHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer <redacted>",
	}
}
