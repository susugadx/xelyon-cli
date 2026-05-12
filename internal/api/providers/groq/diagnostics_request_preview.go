package groq

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildGroqDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "Groq request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"Groq request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildGroqDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultGroqDiagnosticSmokeMaxOutputTokens
	}

	previewCfg := groqDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	provider := New(os.Getenv(groqAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	preview := DiagnosticRequestPreview{}
	for _, request := range groqDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, DiagnosticRequestPreviewRequest{
				Name:        request.Name,
				Skipped:     true,
				SkipReason:  "Groq function calling payloads are disabled (GROQ_FUNCTION_CALLING=0)",
				ToolPayload: true,
				Route:       report.Route,
			})
			continue
		}

		requestCtx := newGroqDiagnosticSmokeRequestContext(ctx, previewCfg, request, io.Discard)
		if request.ToolPayload {
			provider.SetToolChoice(groqDiagnosticSmokeToolName)
		} else {
			provider.ClearToolChoice()
		}
		preview.Requests = append(preview.Requests, buildGroqDiagnosticRequestPreviewRequest(requestCtx, provider, report, request))
	}
	return preview, nil
}

func buildGroqDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request groqDiagnosticSmokeRequest,
) DiagnosticRequestPreviewRequest {
	return DiagnosticRequestPreviewRequest{
		Name:        request.Name,
		ToolPayload: request.ToolPayload,
		Route:       report.Route,
		Method:      "POST",
		URL:         report.APIURL,
		Headers:     providerdiag.RedactedBearerHeaders(),
		Body: provider.buildChatCompletionsRequest(
			ctx,
			request.SystemPrompt,
			[]api.Message{{Role: "user", Content: request.UserContent}},
			report.Model,
		),
	}
}
