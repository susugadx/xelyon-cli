package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestShouldThrottleWrite_WriteTool(t *testing.T) {
	a := &Agent{ProviderName: "Any"}
	tc := &tools.ToolCall{Tool: "str_replace", Args: map[string]string{"path": "a.go"}}
	if !a.shouldThrottleWrite(tc) {
		t.Error("shouldThrottleWrite should return true for write tool")
	}
}

func TestShouldThrottleWrite_ReadTool(t *testing.T) {
	a := &Agent{ProviderName: "Any"}
	tc := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "a.go"}}
	if a.shouldThrottleWrite(tc) {
		t.Error("shouldThrottleWrite should return false for read tool")
	}
}

func TestShouldThrottleWrite_Bash(t *testing.T) {
	a := &Agent{ProviderName: "Any"}
	tc := &tools.ToolCall{Tool: "bash", Args: map[string]string{"command": "ls"}}
	if a.shouldThrottleWrite(tc) {
		t.Error("shouldThrottleWrite should return false for bash")
	}
}

func TestAutoReadBack_AppendsToHistory(t *testing.T) {
	a := &Agent{
		ProviderName: "Any",
		History: []api.Message{
			{Role: "tool", Content: "Successfully replaced"},
		},
	}

	// autoReadBack は実際にファイルを読むので、存在しないパスの場合は Error: で追記される
	tc := &tools.ToolCall{Tool: "str_replace", Args: map[string]string{"path": "/nonexistent/test.go"}}
	a.autoReadBack(tc)

	if len(a.History) != 1 {
		t.Fatalf("History length should be 1, got %d", len(a.History))
	}

	last := a.History[len(a.History)-1]
	if last.Content == "Successfully replaced" {
		t.Error("autoReadBack should have appended content to the last history entry")
	}
}

func TestAutoReadBack_EmptyHistory(t *testing.T) {
	a := &Agent{
		ProviderName: "Any",
		History:      []api.Message{},
	}

	tc := &tools.ToolCall{Tool: "str_replace", Args: map[string]string{"path": "test.go"}}

	// Should not panic
	a.autoReadBack(tc)

	if len(a.History) != 0 {
		t.Errorf("History length should be 0, got %d", len(a.History))
	}
}

func TestAutoReadBack_SkipsDeleteFile(t *testing.T) {
	a := &Agent{
		ProviderName: "Any",
		History: []api.Message{
			{Role: "tool", Content: "File deleted"},
		},
	}

	tc := &tools.ToolCall{Tool: "delete_file", Args: map[string]string{"path": "test.go"}}
	a.autoReadBack(tc)

	last := a.History[len(a.History)-1]
	if last.Content != "File deleted" {
		t.Error("autoReadBack should not modify history for delete_file")
	}
}

func TestAutoReadBack_SkipsMoveFile(t *testing.T) {
	a := &Agent{
		ProviderName: "Any",
		History: []api.Message{
			{Role: "tool", Content: "File moved"},
		},
	}

	tc := &tools.ToolCall{Tool: "move_file", Args: map[string]string{"src": "a.go", "dest": "b.go"}}
	a.autoReadBack(tc)

	last := a.History[len(a.History)-1]
	if last.Content != "File moved" {
		t.Error("autoReadBack should not modify history for move_file")
	}
}

func TestAutoReadBack_SkipsGrepReplace(t *testing.T) {
	a := &Agent{
		ProviderName: "Any",
		History: []api.Message{
			{Role: "tool", Content: "Replaced in 3 files"},
		},
	}

	tc := &tools.ToolCall{Tool: "grep_replace", Args: map[string]string{"path": ".", "pattern": "foo"}}
	a.autoReadBack(tc)

	last := a.History[len(a.History)-1]
	if last.Content != "Replaced in 3 files" {
		t.Error("autoReadBack should not modify history for grep_replace")
	}
}

func TestInjectWriteThrottleMessage_OnlyWrites(t *testing.T) {
	a := &Agent{
		History: []api.Message{
			{Role: "tool", Content: "Success"},
		},
	}

	a.injectWriteThrottleMessage(3, 0)

	last := a.History[len(a.History)-1]
	expected := "3 write operation(s) were queued but not executed"
	if !strings.Contains(last.Content, expected) {
		t.Errorf("throttle message should contain %q, got: %s", expected, last.Content)
	}
}

func TestInjectWriteThrottleMessage_WithCommands(t *testing.T) {
	a := &Agent{
		History: []api.Message{
			{Role: "tool", Content: "Success"},
		},
	}

	a.injectWriteThrottleMessage(2, 1)

	last := a.History[len(a.History)-1]
	if last.Content == "Success" {
		t.Error("injectWriteThrottleMessage should have appended to the last history entry")
	}

	expected := "2 write operation(s) and 1 command(s) were queued but not executed"
	if !strings.Contains(last.Content, expected) {
		t.Errorf("throttle message should contain %q, got: %s", expected, last.Content)
	}
}

func TestExtractPathFromToolCall(t *testing.T) {
	tests := []struct {
		name string
		tc   *tools.ToolCall
		want string
	}{
		{"path arg", &tools.ToolCall{Args: map[string]string{"path": "a.go"}}, "a.go"},
		{"dest arg", &tools.ToolCall{Args: map[string]string{"dest": "b.go"}}, "b.go"},
		{"both args", &tools.ToolCall{Args: map[string]string{"path": "a.go", "dest": "b.go"}}, "a.go"},
		{"no args", &tools.ToolCall{Args: map[string]string{}}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathFromToolCall(tt.tc)
			if got != tt.want {
				t.Errorf("extractPathFromToolCall() = %q, want %q", got, tt.want)
			}
		})
	}
}
