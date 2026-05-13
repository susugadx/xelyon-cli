package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseGemini_PrintRequestBuildsTextToolImageAndWebBodies(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatalf("print request should not send network traffic")
	}))
	defer server.Close()

	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, server.URL)
	t.Setenv(geminiFunctionCallingEnv, "1")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultGeminiDiagnosticModel,
		CatalogModel:   defaultGeminiDiagnosticModel,
		PrintRequest:   true,
		TextSmoke:      true,
		ToolSmoke:      true,
		ImageSmoke:     true,
		WebSearchSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if requestCount != 0 {
		t.Fatalf("requestCount = %d, want no network", requestCount)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 4 {
		t.Fatalf("RequestPreview = %#v, want text/tool/image/web requests", report.RequestPreview)
	}

	text := report.RequestPreview.Requests[0]
	if text.Name != "text" || text.Route != DiagnosticRouteStreamGenerateContentSSE || text.Headers["x-goog-api-key"] != "<redacted>" {
		t.Fatalf("text preview = %#v, want stream route and redacted header", text)
	}
	if _, ok := text.Body.(GeminiRequest); !ok {
		t.Fatalf("text body type = %T, want GeminiRequest", text.Body)
	}

	tool := report.RequestPreview.Requests[1]
	toolBody, ok := tool.Body.(GeminiRequestWithTools)
	if !ok {
		t.Fatalf("tool body type = %T, want GeminiRequestWithTools", tool.Body)
	}
	if tool.Route != DiagnosticRouteStreamGenerateContentSSE || toolBody.ToolConfig == nil || toolBody.ToolConfig.FunctionCallingConfig.Mode != "ANY" {
		t.Fatalf("tool preview = %#v body=%#v, want stream route with ANY mode", tool, toolBody)
	}
	if len(toolBody.Tools) != 1 || len(toolBody.Tools[0].FunctionDeclarations) != 1 || toolBody.Tools[0].FunctionDeclarations[0].Name != geminiDiagnosticToolName {
		t.Fatalf("tool declarations = %#v, want only diagnostic tool", toolBody.Tools)
	}

	image := report.RequestPreview.Requests[2]
	imageBody, ok := image.Body.(GeminiMultimodalRequest)
	if !ok {
		t.Fatalf("image body type = %T, want GeminiMultimodalRequest", image.Body)
	}
	if image.Route != DiagnosticRouteStreamGenerateContentSSE || len(imageBody.Tools) != 0 {
		t.Fatalf("image preview = %#v body=%#v, want stream route without tools", image, imageBody)
	}

	web := report.RequestPreview.Requests[3]
	webBody, ok := web.Body.(webSearchRequest)
	if !ok {
		t.Fatalf("web body type = %T, want webSearchRequest", web.Body)
	}
	if web.Route != DiagnosticRouteGenerateContent || len(webBody.Tools) != 1 || webBody.Tools[0].GoogleSearch == nil {
		t.Fatalf("web preview = %#v body=%#v, want generateContent google_search body", web, webBody)
	}
}
