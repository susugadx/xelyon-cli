package bedrock

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseBedrock_PrintRequestDoesNotRequireAWSCredentialsOrRunSmoke(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultModel,
		CatalogModel: defaultModel,
		PrintRequest: true,
		ToolSmoke:    true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for --print-request without AWS credentials: %#v", report.Checks)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil for --print-request", report.Smoke)
	}
	requireBedrockDiagnosticPreviewRequest(t, requireBedrockDiagnosticRequestPreview(t, report, 1), 0, bedrockDiagnosticToolRequestName)
	if hasBedrockDiagnosticCheckName(report, "auth") {
		t.Fatalf("auth check should be skipped for --print-request: %#v", report.Checks)
	}
}

func TestDiagnoseBedrock_PrintRequestBuildsClaudeTextToolImageAndThinkingBodies(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:        config.DefaultConfig(),
		Model:         defaultModel,
		CatalogModel:  defaultModel,
		PrintRequest:  true,
		TextSmoke:     true,
		ToolSmoke:     true,
		ImageSmoke:    true,
		ThinkingSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	preview := requireBedrockDiagnosticRequestPreview(t, report, 4)

	text := requireBedrockDiagnosticPreviewRequest(t, preview, 0, bedrockDiagnosticTextRequestName)
	if text.Route != string(bedrockRouteClaudeMessages) || text.Operation != bedrockDiagnosticOperationInvokeModelWithResponseStream {
		t.Fatalf("text preview = %#v, want Claude Messages invoke route", text)
	}
	if text.ModelID != defaultModel || !strings.Contains(text.URL, "/invoke-with-response-stream") {
		t.Fatalf("text preview target = model_id:%q url:%q, want Bedrock invoke target", text.ModelID, text.URL)
	}
	if text.Headers["Authorization"] != "<redacted: AWS SigV4>" {
		t.Fatalf("Authorization preview = %q, want redacted SigV4", text.Headers["Authorization"])
	}
	if _, ok := text.Body.(BedrockClaudeMessagesRequest); !ok {
		t.Fatalf("text body type = %T, want BedrockClaudeMessagesRequest", text.Body)
	}

	tool := requireBedrockDiagnosticPreviewRequest(t, preview, 1, bedrockDiagnosticToolRequestName)
	toolBody := requireBedrockDiagnosticPreviewBody[BedrockClaudeMessagesRequest](t, tool)
	if !tool.ToolPayload || len(toolBody.Tools) != 1 || toolBody.Tools[0].Name != diagnosticSmokeToolName {
		t.Fatalf("tool preview = %#v body=%#v, want diagnostic Bedrock tool", tool, toolBody)
	}

	image := requireBedrockDiagnosticPreviewRequest(t, preview, 2, bedrockDiagnosticImageRequestName)
	imageBody := requireBedrockDiagnosticPreviewBody[BedrockClaudeMultimodalRequest](t, image)
	if !image.ImagePayload || len(imageBody.Tools) != 0 || len(imageBody.Messages) != 1 {
		t.Fatalf("image preview = %#v body=%#v, want one multimodal message without tools", image, imageBody)
	}

	thinking := requireBedrockDiagnosticPreviewRequest(t, preview, 3, bedrockDiagnosticThinkingRequestName)
	thinkingBody := requireBedrockDiagnosticPreviewBody[BedrockClaudeMessagesRequest](t, thinking)
	if !thinking.ThinkingEnabled || thinkingBody.Thinking == nil {
		t.Fatalf("thinking preview = %#v body=%#v, want thinking request config", thinking, thinkingBody)
	}
}

func TestBedrockDiagnosticClaudePreviewBodiesMatchRuntimeBuilders(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	cfg := config.DefaultConfig()
	report := newBedrockDiagnosticPreviewReport(defaultModel, bedrockRouteClaudeMessages)
	options := DiagnosticOptions{
		TextSmoke:       true,
		ToolSmoke:       true,
		ImageSmoke:      true,
		ThinkingSmoke:   true,
		MaxOutputTokens: 8,
	}
	plan := buildBedrockDiagnosticRequestPlan(options)

	preview, err := buildBedrockDiagnosticRequestPreview(context.Background(), cfg, report, options, plan)
	if err != nil {
		t.Fatalf("buildBedrockDiagnosticRequestPreview() error = %v", err)
	}
	if len(preview.Requests) != len(plan.Requests) {
		t.Fatalf("preview requests = %d, want %d", len(preview.Requests), len(plan.Requests))
	}

	previewCfg := bedrockDiagnosticSmokeConfig(cfg, report, options.MaxOutputTokens)
	provider := &Provider{region: report.Region, runtimeConfig: previewCfg}
	for i, request := range plan.Requests {
		requestCfg := bedrockDiagnosticRequestConfig(previewCfg, request)
		requestCtx := newBedrockDiagnosticSmokeRequestContext(context.Background(), requestCfg, request, nil)
		req := provider.resolveBedrockRequestContext(requestCtx, report.Model)
		var wantBody any
		if request.ImagePayload {
			wantBody = provider.buildBedrockClaudeImageRequest(requestCtx, request.SystemPrompt, nil, request.UserContent, bedrockDiagnosticImage(), req)
		} else {
			wantBody = provider.buildBedrockClaudeMessagesRequest(requestCtx, request.SystemPrompt, bedrockDiagnosticUserMessages(request.UserContent), req)
		}
		if got, want := canonicalBedrockDiagnosticJSON(t, preview.Requests[i].Body), canonicalBedrockDiagnosticJSON(t, wantBody); got != want {
			t.Fatalf("%s preview body drifted from runtime builder:\ngot:  %s\nwant: %s", request.Name, got, want)
		}
	}
}

func TestDiagnoseBedrock_PrintRequestBuildsConverseBodiesAndSkippedRequests(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	const model = "amazon.nova-pro-v1:0"
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:        bedrockDiagnosticTestConfig(model, ""),
		Model:         model,
		PrintRequest:  true,
		TextSmoke:     true,
		ToolSmoke:     true,
		ImageSmoke:    true,
		ThinkingSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	preview := requireBedrockDiagnosticRequestPreview(t, report, 4)

	text := requireBedrockDiagnosticPreviewRequest(t, preview, 0, bedrockDiagnosticTextRequestName)
	if text.Route != string(bedrockRouteConverseStream) || text.Operation != bedrockDiagnosticOperationConverseStream {
		t.Fatalf("text preview = %#v, want ConverseStream text request", text)
	}
	if text.ModelID != model || !strings.Contains(text.URL, "/converse-stream") {
		t.Fatalf("text preview target = model_id:%q url:%q, want Bedrock Converse target", text.ModelID, text.URL)
	}
	textBody := canonicalBedrockDiagnosticJSON(t, text.Body)
	if !strings.Contains(textBody, `"messages"`) || !strings.Contains(textBody, `"inferenceConfig"`) {
		t.Fatalf("text body = %s, want Converse messages and inferenceConfig", textBody)
	}

	tool := requireBedrockDiagnosticPreviewRequest(t, preview, 1, bedrockDiagnosticToolRequestName)
	toolBody := canonicalBedrockDiagnosticJSON(t, tool.Body)
	if !tool.ToolPayload || !strings.Contains(toolBody, `"toolConfig"`) || !strings.Contains(toolBody, diagnosticSmokeToolName) {
		t.Fatalf("tool preview = %#v body=%s, want diagnostic Converse toolConfig", tool, toolBody)
	}

	for _, request := range preview.Requests[2:] {
		if !request.Skipped || !strings.Contains(request.SkipReason, "ConverseStream route does not support image or thinking smoke") {
			t.Fatalf("request = %#v, want skipped unsupported Converse image/thinking request", request)
		}
	}
}

func TestBedrockDiagnosticConversePreviewBodyMatchesRuntimeBuilder(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	const model = "amazon.nova-pro-v1:0"
	cfg := bedrockDiagnosticTestConfig(model, "")
	report := newBedrockDiagnosticPreviewReport(model, bedrockRouteConverseStream)
	options := DiagnosticOptions{ToolSmoke: true, MaxOutputTokens: 8}
	plan := buildBedrockDiagnosticRequestPlan(options)

	preview, err := buildBedrockDiagnosticRequestPreview(context.Background(), cfg, report, options, plan)
	if err != nil {
		t.Fatalf("buildBedrockDiagnosticRequestPreview() error = %v", err)
	}
	if len(preview.Requests) != 1 {
		t.Fatalf("preview requests = %#v, want one tool request", preview.Requests)
	}

	previewCfg := bedrockDiagnosticSmokeConfig(cfg, report, options.MaxOutputTokens)
	provider := &Provider{region: report.Region, runtimeConfig: previewCfg}
	request := plan.Requests[0]
	requestCfg := bedrockDiagnosticRequestConfig(previewCfg, request)
	requestCtx := newBedrockDiagnosticSmokeRequestContext(context.Background(), requestCfg, request, nil)
	req := provider.resolveBedrockRequestContext(requestCtx, report.Model)
	input, err := provider.buildConverseStreamInput(requestCtx, request.SystemPrompt, bedrockDiagnosticUserMessages(request.UserContent), req)
	if err != nil {
		t.Fatalf("buildConverseStreamInput() error = %v", err)
	}
	wantBody, err := normalizeBedrockConverseStreamPreviewBody(input)
	if err != nil {
		t.Fatalf("normalizeBedrockConverseStreamPreviewBody() error = %v", err)
	}
	if got, want := canonicalBedrockDiagnosticJSON(t, preview.Requests[0].Body), canonicalBedrockDiagnosticJSON(t, wantBody); got != want {
		t.Fatalf("Converse preview body drifted from runtime builder:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestDiagnoseBedrock_PrintRequestRecordsSkippedToolWhenFunctionCallingDisabled(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultModel,
		CatalogModel: defaultModel,
		PrintRequest: true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for skipped tool preview: %#v", report.Checks)
	}
	tool := requireBedrockDiagnosticPreviewRequest(t, requireBedrockDiagnosticRequestPreview(t, report, 1), 0, bedrockDiagnosticToolRequestName)
	if !tool.ToolPayload || !tool.Skipped || !strings.Contains(tool.SkipReason, "BEDROCK_FUNCTION_CALLING=0") {
		t.Fatalf("tool preview request = %#v, want skipped tool entry", tool)
	}
}

func newBedrockDiagnosticPreviewReport(model string, route bedrockRoute) DiagnosticReport {
	return DiagnosticReport{
		Region:                 "us-east-1",
		Model:                  model,
		CatalogModel:           model,
		Route:                  string(route),
		FunctionCallingEnabled: true,
	}
}

func requireBedrockDiagnosticRequestPreview(t *testing.T, report DiagnosticReport, wantRequests int) DiagnosticRequestPreview {
	t.Helper()
	if report.RequestPreview == nil {
		t.Fatalf("RequestPreview = nil, want %d requests", wantRequests)
	}
	if len(report.RequestPreview.Requests) != wantRequests {
		t.Fatalf("RequestPreview = %#v, want %d requests", report.RequestPreview, wantRequests)
	}
	return *report.RequestPreview
}

func requireBedrockDiagnosticPreviewRequest(t *testing.T, preview DiagnosticRequestPreview, index int, name string) DiagnosticRequestPreviewRequest {
	t.Helper()
	if index < 0 || index >= len(preview.Requests) {
		t.Fatalf("preview request index %d out of range: %#v", index, preview.Requests)
	}
	request := preview.Requests[index]
	if request.Name != name {
		t.Fatalf("preview request[%d].Name = %q, want %q; request=%#v", index, request.Name, name, request)
	}
	return request
}

func requireBedrockDiagnosticPreviewBody[T any](t *testing.T, request DiagnosticRequestPreviewRequest) T {
	t.Helper()
	body, ok := request.Body.(T)
	if !ok {
		var zero T
		t.Fatalf("%s body type = %T, want %T", request.Name, request.Body, zero)
	}
	return body
}

func bedrockDiagnosticUserMessages(content string) []api.Message {
	return []api.Message{{Role: "user", Content: content}}
}

func canonicalBedrockDiagnosticJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	return string(payload)
}
