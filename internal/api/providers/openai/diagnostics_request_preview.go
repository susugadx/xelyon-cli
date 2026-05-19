package openai

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const openAIDiagnosticPreviewRetentionResponseID = "${retention_initial.response_id}"

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildOpenAIDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "OpenAI request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"OpenAI request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildOpenAIDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultOpenAIDiagnosticSmokeMaxOutputToks
	}

	baseSmokeCfg := openAIDiagnosticConfigWithModelPolicy(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	baseSmokeCfg.Responses.Store = false
	baseSmokeCfg.Responses.PersistResponseID = false

	provider := New(os.Getenv(openAIAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()
	defer provider.ClearResponseID()

	preview := DiagnosticRequestPreview{}
	for _, request := range openAIDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, providerdiag.NewSkippedRoutedResponsesPreviewRequest(
				request,
				report.Route,
				"OpenAI function calling payloads are disabled (OPENAI_FUNCTION_CALLING=0)",
			))
			continue
		}
		if request.RetentionPayload && report.Route == DiagnosticRouteChatCompletions {
			return preview, fmt.Errorf("OpenAI Responses retention request preview is not supported on the Chat Completions route")
		}

		requestCfg := openAIDiagnosticSmokeRequestConfig(baseSmokeCfg, request)
		requestCtx := newOpenAIDiagnosticSmokeRequestContext(ctx, requestCfg, request, io.Discard)
		if request.ToolPayload {
			provider.SetToolChoice(openAIDiagnosticSmokeToolName)
		} else {
			provider.ClearToolChoice()
		}
		if request.Name == "retention_followup" {
			provider.SetResponseID(openAIDiagnosticPreviewRetentionResponseID)
		} else {
			provider.ClearResponseID()
		}

		previewRequest := buildOpenAIDiagnosticRequestPreviewRequest(requestCtx, provider, report, request)
		preview.Requests = append(preview.Requests, previewRequest)
	}
	return preview, nil
}

func buildOpenAIDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request openAIDiagnosticSmokeRequest,
) DiagnosticRequestPreviewRequest {
	history := []api.Message{{Role: "user", Content: request.UserContent}}

	if report.Route == DiagnosticRouteChatCompletions {
		body := provider.buildChatCompletionsRequest(ctx, request.SystemPrompt, history, report.Model)
		return providerdiag.NewRoutedResponsesPreviewRequest(request, report.Route, providerdiag.RequestPreviewTransport{
			Method:  "POST",
			URL:     report.APIURL,
			Headers: openAIDiagnosticRequestPreviewHeaders(),
			Body:    body,
		})
	}

	body := provider.buildChatResponsesRequest(ctx, request.SystemPrompt, history, report.Model)
	return providerdiag.NewRoutedResponsesPreviewRequest(request, report.Route, providerdiag.RequestPreviewTransport{
		Method:             "POST",
		URL:                report.ResponsesURL,
		Headers:            openAIDiagnosticRequestPreviewHeaders(),
		PreviousResponseID: body.PreviousResponseID,
		Body:               body,
	})
}

func openAIDiagnosticRequestPreviewHeaders() map[string]string {
	headers := providerdiag.JSONHeaders()
	if strings.TrimSpace(os.Getenv(openAIAPIKeyEnv)) != "" {
		headers["Authorization"] = "Bearer <redacted>"
	}
	return headers
}
