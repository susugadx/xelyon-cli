package kimi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnose_PrintRequestBuildsTextToolImageAndWebBodiesWithoutAPIKey(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatalf("print request should not send network requests")
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv(kimiFunctionCallingEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		Model:           "corp-kimi-model",
		CatalogModel:    "kimi-k2.6",
		PrintRequest:    true,
		ToolSmoke:       true,
		ImageSmoke:      true,
		WebSearchSmoke:  true,
		MaxOutputTokens: 8,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false without API key for print request: %#v", report.Checks)
	}
	if requestCount != 0 {
		t.Fatalf("requestCount = %d, want 0", requestCount)
	}
	if hasKimiDiagnosticCheckName(report, "auth") {
		t.Fatalf("auth check should be skipped for print request: %#v", report.Checks)
	}
	preview := requireKimiRequestPreview(t, report, 6)

	text := requireKimiPreviewRequest(t, preview, kimiDiagnosticSmokeCacheFirstName)
	if text.Name != kimiDiagnosticSmokeCacheFirstName || text.Route != DiagnosticRouteChatCompletions || text.Headers["Authorization"] != "Bearer <redacted>" {
		t.Fatalf("text preview = %#v, want redacted Chat Completions request", text)
	}
	textBody := kimiPreviewBodyMap(t, text.Body)
	if textBody["model"] != "corp-kimi-model" || textBody["max_completion_tokens"] != float64(8) {
		t.Fatalf("text body model/max = %#v/%#v, want corp model and max 8", textBody["model"], textBody["max_completion_tokens"])
	}
	if _, ok := textBody["tools"]; ok {
		t.Fatalf("text body tools = %#v, want absent", textBody["tools"])
	}
	if !strings.Contains(kimiRenderedPreviewBody(t, textBody["messages"]), "xelyon kimi doctor cache one") {
		t.Fatalf("text messages = %#v, want cache one prompt", textBody["messages"])
	}

	image := requireKimiPreviewRequest(t, preview, kimiDiagnosticSmokeImageName)
	if image.Name != kimiDiagnosticSmokeImageName || !image.ImagePayload || image.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("image preview = %#v, want image Chat Completions request", image)
	}
	imageBody := kimiPreviewBodyMap(t, image.Body)
	if !strings.Contains(kimiRenderedPreviewBody(t, imageBody["messages"]), "data:image/png;base64,"+kimiDiagnosticPNGBase64) {
		t.Fatalf("image body = %#v, want diagnostic data URL", imageBody)
	}
	if _, ok := imageBody["tools"]; ok {
		t.Fatalf("image body tools = %#v, want absent", imageBody["tools"])
	}

	web := requireKimiPreviewRequest(t, preview, kimiDiagnosticSmokeWebSearchName)
	if web.Name != kimiDiagnosticSmokeWebSearchName || !web.WebSearchPayload || web.Route != DiagnosticRouteChatCompletionsWebSearch {
		t.Fatalf("web preview = %#v, want web search request", web)
	}
	webBody := kimiPreviewBodyMap(t, web.Body)
	if !strings.Contains(kimiRenderedPreviewBody(t, webBody["tools"]), kimiWebSearchToolName) {
		t.Fatalf("web tools = %#v, want built-in web_search tool", webBody["tools"])
	}
	if _, ok := webBody["tool_choice"]; ok {
		t.Fatalf("web tool_choice = %#v, want absent", webBody["tool_choice"])
	}

	tool := requireKimiPreviewRequest(t, preview, kimiDiagnosticSmokeToolName)
	if tool.Name != kimiDiagnosticSmokeToolName || !tool.ToolPayload || tool.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("tool preview = %#v, want tool request", tool)
	}
	toolBody := kimiPreviewBodyMap(t, tool.Body)
	if !strings.Contains(kimiRenderedPreviewBody(t, toolBody["tools"]), diagnosticSmokeToolName) {
		t.Fatalf("tool body tools = %#v, want diagnostic tool", toolBody["tools"])
	}
	if !strings.Contains(kimiRenderedPreviewBody(t, toolBody["tool_choice"]), diagnosticSmokeToolName) {
		t.Fatalf("tool_choice = %#v, want forced diagnostic tool", toolBody["tool_choice"])
	}
}

func TestDiagnose_PrintRequestSkipsDisabledToolPreview(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "")
	t.Setenv(kimiAPIURLEnv, "")
	t.Setenv(kimiFunctionCallingEnv, "0")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		PrintRequest: true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for disabled tool preview skip: %#v", report.Checks)
	}
	preview := requireKimiRequestPreview(t, report, 4)
	tool := requireKimiPreviewRequest(t, preview, kimiDiagnosticSmokeToolName)
	if tool.Name != kimiDiagnosticSmokeToolName || !tool.Skipped || !tool.ToolPayload {
		t.Fatalf("tool preview = %#v, want skipped tool request", tool)
	}
}

func kimiPreviewBodyMap(t *testing.T, body any) map[string]any {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal preview body: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal preview body: %v\n%s", err, string(payload))
	}
	return out
}

func kimiRenderedPreviewBody(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal preview body value: %v", err)
	}
	return string(payload)
}

func requireKimiRequestPreview(t *testing.T, report DiagnosticReport, wantRequests int) DiagnosticRequestPreview {
	t.Helper()
	if report.RequestPreview == nil {
		t.Fatalf("RequestPreview = nil, want %d requests", wantRequests)
	}
	if len(report.RequestPreview.Requests) != wantRequests {
		t.Fatalf("RequestPreview = %#v, want %d requests", report.RequestPreview, wantRequests)
	}
	return *report.RequestPreview
}

func requireKimiPreviewRequest(t *testing.T, preview DiagnosticRequestPreview, name string) DiagnosticRequestPreviewRequest {
	t.Helper()
	var found *DiagnosticRequestPreviewRequest
	for i := range preview.Requests {
		if preview.Requests[i].Name != name {
			continue
		}
		if found != nil {
			t.Fatalf("RequestPreview has duplicate request name %q: %#v", name, preview.Requests)
		}
		found = &preview.Requests[i]
	}
	if found == nil {
		t.Fatalf("RequestPreview missing request %q: %#v", name, preview.Requests)
	}
	return *found
}
