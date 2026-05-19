package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildOpenRouterDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "OpenRouter request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"OpenRouter request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildOpenRouterDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultOpenRouterDiagnosticSmokeMaxOutputTokens
	}

	previewCfg := openRouterDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	provider := New(os.Getenv(openRouterAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	preview := DiagnosticRequestPreview{}
	for _, request := range openRouterDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, providerdiag.NewSkippedTextToolPreviewRequest(request, report.Route, openRouterDiagnosticToolSmokeSkipReason))
			continue
		}

		requestCtx := newOpenRouterDiagnosticSmokeRequestContext(ctx, previewCfg, request, io.Discard)
		applyOpenRouterDiagnosticToolChoice(provider, request)
		previewRequest, err := buildOpenRouterDiagnosticRequestPreviewRequest(requestCtx, provider, report, request)
		preview.Requests = append(preview.Requests, previewRequest)
		if err != nil {
			return preview, err
		}
	}
	return preview, nil
}

func buildOpenRouterDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request openRouterDiagnosticSmokeRequest,
) (DiagnosticRequestPreviewRequest, error) {
	payload, err := buildOpenRouterDiagnosticPreviewPayload(ctx, provider, report, request)
	if err != nil {
		return DiagnosticRequestPreviewRequest{}, err
	}
	body, err := decodeOpenRouterDiagnosticPreviewBody(payload)
	if err != nil {
		return DiagnosticRequestPreviewRequest{}, err
	}

	return providerdiag.NewTextToolPreviewRequest(request, report.Route, providerdiag.RequestPreviewTransport{
		Method:  "POST",
		URL:     report.APIURL,
		Headers: openRouterDiagnosticPreviewHeaders(),
		Body:    body,
	}), nil
}

func buildOpenRouterDiagnosticPreviewPayload(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request openRouterDiagnosticSmokeRequest,
) ([]byte, error) {
	history := []api.Message{{Role: "user", Content: request.UserContent}}
	switch report.Route {
	case DiagnosticRouteAnthropicMessages:
		return provider.buildClaudeChatPayload(ctx, request.SystemPrompt, history, "", report.Model, nil)
	default:
		return provider.buildOpenAITextChatPayload(ctx, request.SystemPrompt, history, report.Model)
	}
}

func decodeOpenRouterDiagnosticPreviewBody(payload []byte) (any, error) {
	var body any
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func openRouterDiagnosticPreviewHeaders() map[string]string {
	headers := providerdiag.RedactedBearerHeaders()
	headers["HTTP-Referer"] = "https://github.com/susugadx/xelyon-cli"
	headers["X-Title"] = "XELYON CLI"
	return headers
}
