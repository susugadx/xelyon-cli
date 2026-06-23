package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestBuildLoopAbortHistoryMessage_FCTrigger(t *testing.T) {
	tc := &tools.ToolCall{ID: "call1", Tool: "read_file"}
	msg, ok := toolruntime.BuildLoopAbortHistoryMessage(tc, 2, 2, 3)
	if !ok {
		t.Fatal("expected ok=true for FC trigger")
	}
	if msg.Role != "tool" {
		t.Fatalf("Role = %q, want tool", msg.Role)
	}
	if msg.ToolCallID != "call1" {
		t.Fatalf("ToolCallID = %q, want call1", msg.ToolCallID)
	}
	want := "Tool loop detected: read_file was called 3 times. Execution stopped to prevent an infinite loop."
	if msg.Content != want {
		t.Fatalf("Content = %q, want %q", msg.Content, want)
	}
}

func TestBuildLoopAbortHistoryMessage_TextTrigger(t *testing.T) {
	tc := &tools.ToolCall{Tool: "read_file"}
	msg, ok := toolruntime.BuildLoopAbortHistoryMessage(tc, 2, 2, 4)
	if !ok {
		t.Fatal("expected ok=true for text trigger")
	}
	if msg.Role != "user" {
		t.Fatalf("Role = %q, want user", msg.Role)
	}
	want := "Tool loop detected: the same tool call was repeated 4 times. Try a different approach or ask the user for clarification."
	if msg.Content != want {
		t.Fatalf("Content = %q, want %q", msg.Content, want)
	}
}

func TestBuildLoopAbortHistoryMessage_FCSubsequent(t *testing.T) {
	tc := &tools.ToolCall{ID: "call2", Tool: "search_code"}
	msg, ok := toolruntime.BuildLoopAbortHistoryMessage(tc, 3, 1, 3)
	if !ok {
		t.Fatal("expected ok=true for FC subsequent")
	}
	if msg.Role != "tool" {
		t.Fatalf("Role = %q, want tool", msg.Role)
	}
	if msg.ToolCallID != "call2" {
		t.Fatalf("ToolCallID = %q, want call2", msg.ToolCallID)
	}
	if msg.Content != "Skipped because a previous tool call in this batch triggered loop detection." {
		t.Fatalf("Content = %q, want skipped message", msg.Content)
	}
}

func TestBuildLoopAbortHistoryMessage_TextSubsequent_NoMessage(t *testing.T) {
	tc := &tools.ToolCall{Tool: "search_code"}
	_, ok := toolruntime.BuildLoopAbortHistoryMessage(tc, 3, 1, 3)
	if ok {
		t.Fatal("expected ok=false for text-based subsequent tool")
	}
}
