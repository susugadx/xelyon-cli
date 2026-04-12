package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newDeepSeekTestContext(t *testing.T, thinking bool) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = "high"
	var out bytes.Buffer
	var errOut bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader(""), &out, &errOut)
	ctx := ui.WithRuntime(context.Background(), runtime)
	ctx = config.WithContext(ctx, cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return ctx, &out, &errOut
}

func newDeepSeekSSEResponse(chunks ...string) *http.Response {
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

func TestProvider_IsFunctionCallingEnabled_RespectsEnv(t *testing.T) {
	t.Setenv("DEEPSEEK_FUNCTION_CALLING", "0")
	if New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() should be false when DEEPSEEK_FUNCTION_CALLING=0")
	}

	t.Setenv("DEEPSEEK_FUNCTION_CALLING", "1")
	if !New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() should be true when DEEPSEEK_FUNCTION_CALLING=1")
	}
}

func TestProvider_ChatWithTools_SelectsReasonerModelByThinkingState(t *testing.T) {
	tests := []struct {
		name      string
		thinking  bool
		model     string
		wantModel string
	}{
		{
			name:      "thinking on upgrades default model",
			thinking:  true,
			model:     "",
			wantModel: "deepseek-reasoner",
		},
		{
			name:      "thinking off downgrades explicit reasoner",
			thinking:  false,
			model:     "deepseek-reasoner",
			wantModel: "deepseek-chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured api.ChatRequest
			server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
			})
			t.Setenv("DEEPSEEK_API_URL", server.URL)

			p := New("test-key")
			ctx, _, _ := newDeepSeekTestContext(t, tt.thinking)
			_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, tt.model)
			if err != nil {
				t.Fatalf("ChatWithTools() error = %v", err)
			}
			if captured.Model != tt.wantModel {
				t.Fatalf("request model = %q, want %q", captured.Model, tt.wantModel)
			}
		})
	}
}

func TestProvider_ChatWithTools_ForcedToolChoiceIncludesCustomMCPTool(t *testing.T) {
	var captured api.ChatRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "custom_lookup",
		Description: "custom lookup",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		},
	}})
	p.SetToolChoice("custom_lookup")

	ctx, _, _ := newDeepSeekTestContext(t, false)
	_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Lookup it"}}, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	foundCustomTool := false
	for _, tool := range captured.Tools {
		if tool.Function != nil && tool.Function.Name == "custom_lookup" {
			foundCustomTool = true
			break
		}
	}
	if !foundCustomTool {
		t.Fatalf("request tools should include custom_lookup: %+v", captured.Tools)
	}

	toolChoice, ok := captured.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("toolChoice type = %T, want map[string]any", captured.ToolChoice)
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != "custom_lookup" {
		t.Fatalf("toolChoice function = %#v, want custom_lookup", toolChoice["function"])
	}
}

func TestHandleStreamingResponse_PersistsReasoningAndUsage(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	ctx, out, _ := newDeepSeekTestContext(t, true)
	resp := newDeepSeekSSEResponse(
		`{"choices":[{"delta":{"reasoning_content":"Think"}}]}`,
		`{"choices":[{"delta":{"reasoning_content":" deeply"}}]}`,
		`{"choices":[{"delta":{"content":"Answer"}}]}`,
		`{"choices":[],"usage":{"prompt_tokens":11,"completion_tokens":7,"prompt_cache_hit_tokens":3}}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	)
	defer resp.Body.Close()

	got, err := p.handleStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if got != "Answer" {
		t.Fatalf("handleStreamingResponse() = %q, want %q", got, "Answer")
	}
	if p.LastReasoningContent() != "Think deeply" {
		t.Fatalf("LastReasoningContent() = %q, want %q", p.LastReasoningContent(), "Think deeply")
	}
	if gotUsage.InputTokens != 11 || gotUsage.OutputTokens != 7 || gotUsage.CachedInputTokens != 3 {
		t.Fatalf("usage = %+v, want input=11 output=7 cached=3", gotUsage)
	}
	if !strings.Contains(out.String(), "Think deeply") {
		t.Fatalf("output should include reasoning content, got %q", out.String())
	}
}
