package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOpenRouter_PrintRequestDoesNotRequireAPIKeyOrSendNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "anthropic/claude-sonnet-4.6",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want print request without API key to pass: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckAbsent(t, report, "auth")
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil for --print-request", report.Smoke)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one tool request", report.RequestPreview)
	}
	preview := report.RequestPreview.Requests[0]
	if preview.Name != "tool" || !preview.ToolPayload || preview.Route != DiagnosticRouteAnthropicMessages || preview.URL != server.URL+"/v1/messages" {
		t.Fatalf("preview = %#v, want tool request to Anthropic Skin endpoint", preview)
	}
	if preview.Headers["Authorization"] != "Bearer <redacted>" || preview.Headers["X-Title"] != "XELYON CLI" {
		t.Fatalf("headers = %#v, want redacted OpenRouter headers", preview.Headers)
	}
	body := decodeOpenRouterDiagnosticPreviewBodyForTest(t, preview.Body)
	if body["model"] != "anthropic/claude-sonnet-4.6" || body["anthropic_version"] == "" || body["max_tokens"] != float64(64) {
		t.Fatalf("preview body = %#v, want Anthropic Skin diagnostic body", body)
	}
	tools, ok := body["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want diagnostic Claude tool", body["tools"])
	}
	toolChoice, ok := body["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %T, want forced Anthropic Skin tool choice", body["tool_choice"])
	}
	if toolChoice["type"] != "tool" || toolChoice["name"] != openRouterDiagnosticSmokeToolName {
		t.Fatalf("tool_choice = %#v, want diagnostic tool choice", toolChoice)
	}
}
