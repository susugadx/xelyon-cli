package toolruntime

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestArgsToJSON(t *testing.T) {
	got := ArgsToJSON(map[string]any{"path": "main.go"})
	if got != `{"path":"main.go"}` {
		t.Fatalf("ArgsToJSON() = %q", got)
	}
}

func TestBuildToolResultMessage_FunctionCalling(t *testing.T) {
	msg := BuildToolResultMessage(&tools.ToolCall{ID: "call-1", Tool: "read_file"}, "ok", "text")
	if msg.Role != "tool" || msg.ToolCallID != "call-1" || msg.Content != "ok" {
		t.Fatalf("message = %#v", msg)
	}
}

func TestFormatTextToolResultContent(t *testing.T) {
	got := FormatTextToolResultContent("search_code", "matches")
	want := "[Tool Result for search_code]\nmatches"
	if got != want {
		t.Fatalf("FormatTextToolResultContent() = %q, want %q", got, want)
	}
}

func TestBuildToolResultMessage_TextModeAndNil(t *testing.T) {
	msg := BuildToolResultMessage(&tools.ToolCall{Tool: "read_file"}, "function", "text")
	if msg.Role != "user" || msg.Content != "text" || msg.ToolCallID != "" || msg.ToolName != "" {
		t.Fatalf("text-mode message = %#v, want user text message", msg)
	}

	if msg := BuildToolResultMessage(nil, "function", "text"); msg.Role != "" || msg.Content != "" {
		t.Fatalf("nil tool call message = %#v, want zero value", msg)
	}
}
