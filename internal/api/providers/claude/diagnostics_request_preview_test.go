package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseClaude_PrintRequestBuildsTextToolImageThinkingAndWebBodies(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatalf("print request should not send network traffic")
	}))
	defer server.Close()

	proxyURL := server.URL + "/proxy"
	setClaudeDiagnosticTestEnv(t, proxyURL, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultClaudeModel,
		CatalogModel:   defaultClaudeModel,
		PrintRequest:   true,
		TextSmoke:      true,
		ToolSmoke:      true,
		ImageSmoke:     true,
		ThinkingSmoke:  true,
		WebSearchSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if requestCount != 0 {
		t.Fatalf("requestCount = %d, want no network", requestCount)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 5 {
		t.Fatalf("RequestPreview = %#v, want text/tool/image/thinking/web requests", report.RequestPreview)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	requireClaudePreviewRequestsUseURL(t, report, proxyURL)

	text := report.RequestPreview.Requests[0]
	if text.Name != "text" || text.Route != DiagnosticRouteClaudeMessages || text.Headers["x-api-key"] != "<redacted>" {
		t.Fatalf("text preview = %#v, want Messages route and redacted header", text)
	}
	if _, ok := text.Body.(Request); !ok {
		t.Fatalf("text body type = %T, want Request", text.Body)
	}

	tool := report.RequestPreview.Requests[1]
	toolBody, ok := tool.Body.(Request)
	if !ok {
		t.Fatalf("tool body type = %T, want Request", tool.Body)
	}
	if tool.Route != DiagnosticRouteClaudeMessages || len(toolBody.Tools) != 1 || toolBody.Tools[0].Name != claudeDiagnosticToolName {
		t.Fatalf("tool preview = %#v body=%#v, want diagnostic Claude tool", tool, toolBody)
	}
	requireClaudeToolChoice(t, toolBody.ToolChoice, claudeDiagnosticToolName)

	image := report.RequestPreview.Requests[2]
	imageBody, ok := image.Body.(MultimodalRequest)
	if !ok {
		t.Fatalf("image body type = %T, want MultimodalRequest", image.Body)
	}
	if image.Route != DiagnosticRouteClaudeMessages || len(imageBody.Tools) != 0 || len(imageBody.Messages) != 1 {
		t.Fatalf("image preview = %#v body=%#v, want Messages route without tools", image, imageBody)
	}
	imageMessage, ok := imageBody.Messages[0].(MultimodalMessage)
	if !ok || len(imageMessage.Content) != 2 || imageMessage.Content[0].Source == nil || imageMessage.Content[0].Source.Data != claudeDiagnosticPNGBase64 {
		t.Fatalf("image message = %#v, want diagnostic image block", imageBody.Messages[0])
	}

	thinking := report.RequestPreview.Requests[3]
	thinkingBody, ok := thinking.Body.(Request)
	if !ok {
		t.Fatalf("thinking body type = %T, want Request", thinking.Body)
	}
	if thinking.Route != DiagnosticRouteClaudeMessages || thinkingBody.Thinking == nil {
		t.Fatalf("thinking preview = %#v body=%#v, want thinking request config", thinking, thinkingBody)
	}

	web := report.RequestPreview.Requests[4]
	webBody, ok := web.Body.(webSearchRequest)
	if !ok {
		t.Fatalf("web body type = %T, want webSearchRequest", web.Body)
	}
	if web.Route != DiagnosticRouteClaudeWebSearch || len(webBody.Tools) != 1 || webBody.Tools[0].Type != "web_search_20250305" {
		t.Fatalf("web preview = %#v body=%#v, want Claude web search tool", web, webBody)
	}
	if !strings.Contains(web.Headers["anthropic-beta"], webSearchBetaHeader) {
		t.Fatalf("anthropic-beta = %q, want web search beta", web.Headers["anthropic-beta"])
	}
}

func TestClaudeDiagnosticRequestPreviewBodiesMatchRuntimeBuilders(t *testing.T) {
	cfg := config.DefaultConfig()
	report := DiagnosticReport{
		APIURL:                 "https://api.anthropic.com/v1/messages",
		Model:                  defaultClaudeModel,
		CatalogModel:           defaultClaudeModel,
		FunctionCallingEnabled: true,
	}
	options := DiagnosticOptions{
		TextSmoke:       true,
		ToolSmoke:       true,
		ImageSmoke:      true,
		ThinkingSmoke:   true,
		WebSearchSmoke:  true,
		MaxOutputTokens: 8,
	}

	preview, err := buildClaudeDiagnosticRequestPreview(context.Background(), cfg, report, options)
	if err != nil {
		t.Fatalf("buildClaudeDiagnosticRequestPreview() error = %v", err)
	}
	requests := buildClaudeDiagnosticRequestPlan(options, true).Requests
	if len(preview.Requests) != len(requests) {
		t.Fatalf("preview requests = %d, want %d", len(preview.Requests), len(requests))
	}

	previewCfg := claudeDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, options.MaxOutputTokens)
	provider := New("diagnostic-key")
	provider.SetMCPTools(nil)
	for i, request := range requests {
		requestCtx := newClaudeDiagnosticRequestContext(context.Background(), previewCfg, request, nil)
		wantBody, _ := buildClaudeDiagnosticRequestBody(requestCtx, provider, report, request)
		if got, want := canonicalClaudeDiagnosticJSON(t, preview.Requests[i].Body), canonicalClaudeDiagnosticJSON(t, wantBody); got != want {
			t.Fatalf("%s preview body drifted from diagnostic body builder:\ngot:  %s\nwant: %s", request.Name, got, want)
		}
		if got, want := preview.Requests[i].Route, claudeDiagnosticRequestRoute(request); got != want {
			t.Fatalf("%s route = %q, want %q", request.Name, got, want)
		}
	}
}

func TestDiagnoseClaude_PrintRequestFableToolSmokeUsesAutoToolChoice(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "claude-fable-5",
		CatalogModel: "claude-fable-5",
		PrintRequest: true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want only tool request", report.RequestPreview)
	}
	body, ok := report.RequestPreview.Requests[0].Body.(Request)
	if !ok {
		t.Fatalf("tool body type = %T, want Request", report.RequestPreview.Requests[0].Body)
	}
	if len(body.Tools) != 1 || body.Tools[0].Name != claudeDiagnosticToolName {
		t.Fatalf("Tools = %#v, want diagnostic tool payload", body.Tools)
	}
	if body.ToolChoice != nil {
		t.Fatalf("ToolChoice = %#v, want omitted for always-on thinking model", body.ToolChoice)
	}
}

func TestDiagnoseClaude_PrintRequestThinkingToolSmokeUsesAutoToolChoice(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "", "")
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		PrintRequest: true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	body, ok := report.RequestPreview.Requests[0].Body.(Request)
	if !ok {
		t.Fatalf("tool body type = %T, want Request", report.RequestPreview.Requests[0].Body)
	}
	if body.Thinking == nil || body.Thinking.Type != "adaptive" {
		t.Fatalf("Thinking = %#v, want adaptive thinking request", body.Thinking)
	}
	if body.ToolChoice != nil {
		t.Fatalf("ToolChoice = %#v, want omitted when thinking is enabled", body.ToolChoice)
	}
}

func TestDiagnoseClaude_PrintRequestRecordsSkippedToolWhenFunctionCallingDisabled(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "", "")
	t.Setenv(claudeFunctionCallEnv, "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		PrintRequest: true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for skipped tool preview: %#v", report.Checks)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 2 {
		t.Fatalf("RequestPreview = %#v, want text fallback and skipped tool", report.RequestPreview)
	}
	if report.RequestPreview.Requests[0].Name != "text" || report.RequestPreview.Requests[0].Skipped {
		t.Fatalf("first preview request = %#v, want text fallback", report.RequestPreview.Requests[0])
	}
	tool := report.RequestPreview.Requests[1]
	if tool.Name != "tool" || !tool.ToolPayload || !tool.Skipped || !strings.Contains(tool.SkipReason, claudeFunctionCallEnv) {
		t.Fatalf("tool preview request = %#v, want skipped tool entry", tool)
	}
}

func canonicalClaudeDiagnosticJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	return string(payload)
}
