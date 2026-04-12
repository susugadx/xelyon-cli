package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestChatWithTools_DefaultsModelWhenEmpty(t *testing.T) {
	var reqBody Request
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Content: []Content{{Type: "text", Text: "ok"}},
		})
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	if _, err := p.ChatWithTools(context.Background(), "System", []api.Message{{Role: "user", Content: "Hello"}}, ""); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if reqBody.Model != "claude-sonnet-4-6" {
		t.Fatalf("Model = %q, want %q", reqBody.Model, "claude-sonnet-4-6")
	}
}

func TestChatWithTools_DebugHistoryDump(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Content: []Content{{Type: "text", Text: "ok"}},
		})
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)
	t.Setenv("XELYON_DEBUG_CLAUDE", "1")

	var out bytes.Buffer
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(bytes.NewReader(nil), &out, &out))
	history := []api.Message{
		{Role: "user", Content: "Read test.txt"},
		{
			Role: "assistant",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"test.txt"}`,
					},
				},
			},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "file content"},
	}

	p := New("test-key")
	if _, err := p.ChatWithTools(ctx, "System", history, "claude-sonnet-4-6"); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	logs := out.String()
	for _, fragment := range []string{
		"[DEBUG Claude] === History (3 messages) ===",
		"history[1] role=assistant tool_calls=[call_1]",
		"history[2] role=tool tool_call_id=call_1",
		"[DEBUG Claude] === Converted",
		"tool_use:call_1",
		"tool_result:call_1",
	} {
		if !strings.Contains(logs, fragment) {
			t.Fatalf("debug logs missing %q:\n%s", fragment, logs)
		}
	}
}
