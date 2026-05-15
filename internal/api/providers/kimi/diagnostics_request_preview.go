package kimi

import (
	"context"
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildKimiDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "Kimi request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"Kimi request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildKimiDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultKimiDiagnosticSmokeMaxOutputTokens
	}

	previewCfg := kimiDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	provider := New("diagnostic-key")
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	preview := DiagnosticRequestPreview{}
	plan := buildKimiDiagnosticRequestPlan(options)
	for _, request := range plan.Requests {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, newKimiDiagnosticSkippedToolPreviewRequest(request))
			continue
		}

		requestCtx := newKimiDiagnosticSmokeRequestContext(ctx, previewCfg, provider, request, io.Discard)
		previewRequest, err := buildKimiDiagnosticRequestPreviewRequest(requestCtx, provider, report, request)
		if err != nil {
			return preview, err
		}
		preview.Requests = append(preview.Requests, previewRequest)
	}
	return preview, nil
}

func buildKimiDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request kimiDiagnosticSmokeRequest,
) (DiagnosticRequestPreviewRequest, error) {
	body, err := buildKimiDiagnosticRequestBody(ctx, provider, report.Model, request)
	if err != nil {
		return DiagnosticRequestPreviewRequest{}, err
	}
	return DiagnosticRequestPreviewRequest{
		Name:             request.Name,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		WebSearchPayload: request.WebSearchPayload,
		Route:            kimiDiagnosticRequestRoute(request),
		Method:           "POST",
		URL:              report.APIURL,
		Headers:          providerdiag.RedactedBearerHeaders(),
		Body:             body,
	}, nil
}

func buildKimiDiagnosticRequestBody(ctx context.Context, provider *Provider, model string, request kimiDiagnosticSmokeRequest) (any, error) {
	switch {
	case request.ImagePayload:
		built, err := provider.buildImageChatCompletionsRequest(ctx, request.SystemPrompt, nil, request.UserContent, kimiDiagnosticImage(), model)
		if err != nil {
			return nil, err
		}
		return built.Request, nil
	case request.WebSearchPayload:
		return buildKimiWebSearchRequest(ctx, initialKimiWebSearchMessages(request.UserContent), model, "kimi"), nil
	default:
		return provider.buildChatCompletionsRequest(
			ctx,
			request.SystemPrompt,
			[]api.Message{{Role: "user", Content: request.UserContent}},
			model,
		).Request, nil
	}
}
