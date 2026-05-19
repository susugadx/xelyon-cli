package azure

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

const diagnosticPreviewRetentionResponseID = "${retention_initial.response_id}"

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions) {
	preview, err := buildDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "Azure OpenAI request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"Azure OpenAI request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultDiagnosticSmokeMaxOutputTokens
	}

	baseSmokeCfg := config.CloneConfig(cfg)
	baseSmokeCfg.Responses.Store = false
	baseSmokeCfg.Responses.PersistResponseID = false
	baseSmokeCfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: report.Deployment,
		CatalogModel: report.CatalogModel,
		ModelOverrides: map[string]config.ModelOverride{
			report.Deployment: {
				CatalogModel:    report.CatalogModel,
				MaxOutputTokens: maxOutputTokens,
			},
		},
	})

	provider := New(os.Getenv(apiKeyEnv))
	defer provider.ClearToolChoice()
	defer provider.ClearResponseID()

	preview := DiagnosticRequestPreview{}
	for _, request := range diagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			preview.Requests = append(preview.Requests, providerdiag.NewSkippedResponsesPreviewRequest(
				request,
				report.Route,
				"Azure OpenAI function calling payloads are disabled (AZURE_OPENAI_FUNCTION_CALLING=0)",
			))
			continue
		}

		requestCfg := diagnosticSmokeRequestConfig(baseSmokeCfg, request)
		requestCtx := newDiagnosticSmokeRequestContext(ctx, requestCfg, request, io.Discard)
		if request.ToolPayload {
			provider.SetToolChoice(diagnosticSmokeToolName)
		} else {
			provider.ClearToolChoice()
		}
		if request.Name == "retention_followup" {
			provider.SetResponseID(diagnosticPreviewRetentionResponseID)
		} else {
			provider.ClearResponseID()
		}

		previewRequest := buildDiagnosticRequestPreviewRequest(requestCtx, provider, report, request)
		preview.Requests = append(preview.Requests, previewRequest)
	}
	return preview, nil
}

func buildDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request diagnosticSmokeRequest,
) DiagnosticRequestPreviewRequest {
	body := provider.buildChatResponsesRequest(
		ctx,
		request.SystemPrompt,
		[]api.Message{{Role: "user", Content: request.UserContent}},
		report.Deployment,
	)
	return providerdiag.NewResponsesPreviewRequest(request, report.Route, providerdiag.RequestPreviewTransport{
		Method:             "POST",
		URL:                provider.responsesURL(),
		Headers:            diagnosticRequestPreviewHeaders(report.AuthMode),
		PreviousResponseID: body.PreviousResponseID,
		Body:               body,
	})
}

func diagnosticRequestPreviewHeaders(authMode string) map[string]string {
	headers := providerdiag.JSONHeaders()
	switch strings.TrimSpace(authMode) {
	case "api_key":
		headers["api-key"] = "<redacted>"
	case "entra_id", "entra_id_command":
		headers["Authorization"] = "Bearer <redacted>"
	}
	return headers
}
