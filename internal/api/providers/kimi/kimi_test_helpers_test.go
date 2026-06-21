package kimi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func newKimiTestContext(t *testing.T, thinking bool) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = "high"

	var out bytes.Buffer
	var errOut bytes.Buffer
	runtime := uiruntime.NewRuntime(strings.NewReader(""), &out, &errOut)
	ctx := uiruntime.WithRuntime(context.Background(), runtime)
	ctx = config.WithContext(ctx, cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return ctx, &out, &errOut
}

func mockKimiAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func newKimiReadFileToolProviderForTest() *Provider {
	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "read_file",
		Description: "read",
		Parameters:  map[string]any{"type": "object"},
	}})
	p.SetToolChoice("read_file")
	return p
}

func assertKimiToolPayloadOmitted(t *testing.T, body map[string]any) {
	t.Helper()
	if _, ok := body["tools"]; ok {
		t.Fatalf("tools = %#v, want absent", body["tools"])
	}
	if _, ok := body["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want absent", body["tool_choice"])
	}
}

func kimiStreamingHandler(chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		for _, chunk := range chunks {
			_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}
}

func newKimiSSEResponse(chunks ...string) *http.Response {
	var body strings.Builder
	for _, chunk := range chunks {
		body.WriteString("data: ")
		body.WriteString(chunk)
		body.WriteString("\n\n")
	}
	body.WriteString("data: [DONE]\n\n")
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body.String())),
	}
}

const kimiTestPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="
