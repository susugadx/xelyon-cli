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
