package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func makeLargeContent(lines int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		b.WriteString("line content here\n")
	}
	return b.String()
}

func TestCompactOldToolResults_TruncatesReadLargeResults(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "assistant", Content: "done"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if !strings.Contains(result[2].Content, "truncated") {
		t.Fatalf("expected tool result to be truncated, got %q", result[2].Content)
	}
	if result[2].ToolCallID != "c1" {
		t.Fatalf("expected ToolCallID to be preserved, got %q", result[2].ToolCallID)
	}
	if result[2].ToolName != "search_code" {
		t.Fatalf("expected ToolName to be preserved, got %q", result[2].ToolName)
	}
}

func TestCompactOldToolResults_SingleTurn_CompressesReadTools(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "single turn"},
		{Role: "assistant", Content: "step 1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "read_file"},
		{Role: "assistant", Content: "step 2"},
		{Role: "tool", Content: large, ToolCallID: "c2", ToolName: "read_file"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if !strings.Contains(result[2].Content, "truncated") {
		t.Fatal("expected read tool result before the last assistant to be truncated")
	}
	if strings.Contains(result[4].Content, "truncated") {
		t.Fatal("expected unread tool result after the last assistant to be preserved")
	}
}

func TestCompactOldToolResults_NoAssistant_NoCompression(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "single turn"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "read_file"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if result[1].Content != large {
		t.Fatalf("expected no compression without assistant, got %q", result[1].Content)
	}
}

func TestCompactOldToolResults_SmallResultsNotTruncated(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: "small result", ToolCallID: "c1", ToolName: "str_replace"},
		{Role: "assistant", Content: "done"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if result[2].Content != "small result" {
		t.Fatalf("expected small tool result to stay unchanged, got %q", result[2].Content)
	}
}

func TestCompactOldToolResults_OriginalUnchanged(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "assistant", Content: "done"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if history[2].Content != large {
		t.Fatal("original history was modified")
	}
	if !strings.Contains(result[2].Content, "truncated") {
		t.Fatal("expected copied result to be truncated")
	}
}

func TestCompactOldToolResults_EmptyHistory(t *testing.T) {
	result, _ := CompactOldToolResults(nil, 50, 20, 5)
	if result != nil {
		t.Fatalf("expected nil result, got len=%d", len(result))
	}

	result, _ = CompactOldToolResults([]api.Message{}, 50, 20, 5)
	if len(result) != 0 {
		t.Fatalf("expected empty result, got len=%d", len(result))
	}
}

func TestCompactOldToolResults_NoToolResults(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "how are you"},
		{Role: "assistant", Content: "fine"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if len(result) != len(history) {
		t.Fatalf("expected %d messages, got %d", len(history), len(result))
	}
	for i, msg := range result {
		if msg.Content != history[i].Content {
			t.Fatalf("message %d content mismatch", i)
		}
	}
}

func TestCompactOldToolResults_UniformTruncation(t *testing.T) {
	// 全 tool result が同じ head/tail パラメータで truncation される
	history := []api.Message{{Role: "user", Content: "task"}}
	content := makeLargeContent(100)
	for i := 1; i <= 16; i++ {
		history = append(history,
			api.Message{Role: "assistant", Content: fmt.Sprintf("assistant-%d", i)},
			api.Message{Role: "tool", Content: content, ToolCallID: fmt.Sprintf("c%d", i), ToolName: "str_replace"},
		)
	}
	history = append(history, api.Message{Role: "assistant", Content: "done"})

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	oldest := result[2].Content
	recent := result[32].Content

	if !strings.Contains(oldest, "truncated") || !strings.Contains(recent, "truncated") {
		t.Fatal("expected all tool results to be truncated")
	}
	if len(oldest) != len(recent) {
		t.Fatalf("expected uniform truncation: oldest=%d recent=%d", len(oldest), len(recent))
	}
}

func TestCompressErrorResult(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantResult string
		wantOK     bool
	}{
		{
			name:       "No matches found",
			content:    "No matches found",
			wantResult: "No matches found",
			wantOK:     true,
		},
		{
			name:       "No matches found with whitespace",
			content:    "  No matches found  \n",
			wantResult: "No matches found",
			wantOK:     true,
		},
		{
			name:       "Error: pattern not found",
			content:    "Error: pattern not found in file.go\nsome extra details\nmore details",
			wantResult: "Error: pattern not found in file.go",
			wantOK:     true,
		},
		{
			name:       "Error: old_str not found",
			content:    "Error: old_str not found in the file\nExpected:\nfunc foo()\nGot:\nfunc bar()",
			wantResult: "Error: old_str not found in the file",
			wantOK:     true,
		},
		{
			name:       "Error reading file multiline",
			content:    "Error reading file: /path/to/file.go\nopen /path/to/file.go: permission denied",
			wantResult: "Error reading file: /path/to/file.go",
			wantOK:     true,
		},
		{
			name:       "Error reading file single line",
			content:    "Error reading file: /path/to/file.go",
			wantResult: "Error reading file: /path/to/file.go",
			wantOK:     true,
		},
		{
			name:       "Generic Error: prefix",
			content:    "Error: something went wrong\nstack trace line 1\nstack trace line 2",
			wantResult: "Error: something went wrong",
			wantOK:     true,
		},
		{
			name:       "Normal content not compressed",
			content:    "package main\n\nfunc main() {}",
			wantResult: "",
			wantOK:     false,
		},
		{
			name:       "Content containing Error: but not at start",
			content:    "result contains Error: in the middle",
			wantResult: "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := compressErrorResult(tt.content)
			if ok != tt.wantOK {
				t.Fatalf("compressErrorResult() ok = %v, want %v", ok, tt.wantOK)
			}
			if result != tt.wantResult {
				t.Fatalf("compressErrorResult() = %q, want %q", result, tt.wantResult)
			}
		})
	}
}

func TestCompactOldToolResults_CompressesReadErrors(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: "Error: pattern not found in main.go\nExpected: func foo()\nGot: func bar()", ToolCallID: "c1", ToolName: "str_replace"},
		{Role: "tool", Content: "No matches found", ToolCallID: "c2", ToolName: "search_code"},
		{Role: "assistant", Content: "done"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if result[2].Content != "Error: pattern not found in main.go" {
		t.Fatalf("expected compressed error, got %q", result[2].Content)
	}
	if result[3].Content != "No matches found" {
		t.Fatalf("expected 'No matches found' unchanged, got %q", result[3].Content)
	}
}

func TestCompactOldToolResults_UnreadErrorsNotCompressed(t *testing.T) {
	errorContent := "Error: pattern not found in main.go\nExpected: func foo()\nGot: func bar()"
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: errorContent, ToolCallID: "c1", ToolName: "str_replace"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if result[2].Content != errorContent {
		t.Fatalf("expected unread error to be preserved, got %q", result[2].Content)
	}
}

func TestCompressFailedPair_Basic(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "fix it"},
		{Role: "assistant", Content: "trying", ToolCalls: []api.OpenAIToolCall{
			{ID: "c1", Function: api.OpenAIToolCallFunction{Name: "str_replace"}},
		}},
		{Role: "tool", Content: "Error: old_str not found in file.go\nExpected: func foo()\nGot: func bar()", ToolCallID: "c1", ToolName: "str_replace"},
		{Role: "assistant", Content: "retrying", ToolCalls: []api.OpenAIToolCall{
			{ID: "c2", Function: api.OpenAIToolCallFunction{Name: "str_replace"}},
		}},
		{Role: "tool", Content: "OK", ToolCallID: "c2", ToolName: "str_replace"},
	}

	compressed, ok := compressFailedPair(history, 2)
	if !ok {
		t.Fatal("expected failed pair to be detected")
	}
	if !strings.HasPrefix(compressed.Content, "[Failed: str_replace") {
		t.Fatalf("expected failed summary, got %q", compressed.Content)
	}
	if !strings.Contains(compressed.Content, "Error: old_str not found") {
		t.Fatalf("expected first error line in summary, got %q", compressed.Content)
	}
	if compressed.ToolCallID != "c1" {
		t.Fatalf("expected ToolCallID preserved, got %q", compressed.ToolCallID)
	}
}

func TestCompressFailedPair_DifferentTool(t *testing.T) {
	history := []api.Message{
		{Role: "tool", Content: "Error: file not found", ToolCallID: "c1", ToolName: "read_file"},
		{Role: "assistant", Content: "trying write", ToolCalls: []api.OpenAIToolCall{
			{ID: "c2", Function: api.OpenAIToolCallFunction{Name: "write_file"}},
		}},
	}

	_, ok := compressFailedPair(history, 0)
	if ok {
		t.Fatal("should not detect failed pair when retry uses a different tool")
	}
}

func TestCompressFailedPair_NoAssistantAfter(t *testing.T) {
	history := []api.Message{
		{Role: "tool", Content: "Error: something", ToolCallID: "c1", ToolName: "bash"},
		{Role: "user", Content: "next question"},
	}

	_, ok := compressFailedPair(history, 0)
	if ok {
		t.Fatal("should not detect failed pair when user appears first")
	}
}

func TestCompressFailedPair_NotError(t *testing.T) {
	history := []api.Message{
		{Role: "tool", Content: "Success: done", ToolCallID: "c1", ToolName: "bash"},
		{Role: "assistant", Content: "ok", ToolCalls: []api.OpenAIToolCall{
			{ID: "c2", Function: api.OpenAIToolCallFunction{Name: "bash"}},
		}},
	}

	_, ok := compressFailedPair(history, 0)
	if ok {
		t.Fatal("should not detect failed pair for non-error results")
	}
}

func TestCompactOldToolResults_FailedPairIntegration(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "trying", ToolCalls: []api.OpenAIToolCall{
			{ID: "c1", Function: api.OpenAIToolCallFunction{Name: "str_replace"}},
		}},
		{Role: "tool", Content: "Error: old_str not found in main.go\nline2\nline3", ToolCallID: "c1", ToolName: "str_replace"},
		{Role: "assistant", Content: "retry", ToolCalls: []api.OpenAIToolCall{
			{ID: "c2", Function: api.OpenAIToolCallFunction{Name: "str_replace"}},
		}},
		{Role: "tool", Content: large, ToolCallID: "c2", ToolName: "str_replace"},
		{Role: "assistant", Content: "done"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if !strings.HasPrefix(result[2].Content, "[Failed: str_replace") {
		t.Fatalf("expected failed pair to be compressed, got %q", result[2].Content)
	}
	if !strings.Contains(result[4].Content, "truncated") {
		t.Fatalf("expected retried tool result to be truncated, got %q", result[4].Content)
	}
}

func TestCompactOldToolResults_TruncatedContentFormat(t *testing.T) {
	large := makeLargeContent(100)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "assistant", Content: "done"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)
	content := result[2].Content

	if !strings.HasPrefix(content, "line content here\n") {
		t.Fatal("expected head lines to be preserved")
	}
	if !strings.Contains(content, "... (truncated: was 100 lines") {
		t.Fatalf("expected truncation marker, got %q", content)
	}
	if !strings.Contains(content, "tokens)") {
		t.Fatal("expected token estimate in truncation marker")
	}
	if !strings.HasSuffix(content, "line content here") {
		t.Fatalf("expected tail lines to be preserved, got suffix %q", content[len(content)-40:])
	}
	if len(content) >= len(large) {
		t.Fatal("expected truncated content to be shorter than original")
	}
}

func TestCompactOldToolResults_CompactionMetrics(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "trying", ToolCalls: []api.OpenAIToolCall{
			{ID: "c1", Function: api.OpenAIToolCallFunction{Name: "str_replace"}},
		}},
		{Role: "tool", Content: "Error: old_str not found in main.go\nline2\nline3", ToolCallID: "c1", ToolName: "str_replace"},
		{Role: "assistant", Content: "retry", ToolCalls: []api.OpenAIToolCall{
			{ID: "c2", Function: api.OpenAIToolCallFunction{Name: "str_replace"}},
		}},
		{Role: "tool", Content: large, ToolCallID: "c2", ToolName: "str_replace"},
		{Role: "tool", Content: "Error: pattern not found in file.go\nmore detail", ToolCallID: "c3", ToolName: "str_replace"},
		{Role: "assistant", Content: "done"},
	}

	_, metrics := CompactOldToolResults(history, 50, 20, 5)

	if metrics.FailedPairCompressions != 1 {
		t.Fatalf("FailedPairCompressions = %d, want 1", metrics.FailedPairCompressions)
	}
	if metrics.ErrorCompressions != 1 {
		t.Fatalf("ErrorCompressions = %d, want 1", metrics.ErrorCompressions)
	}
}

func TestIsCompactedToolResult_SkipsTruncate(t *testing.T) {
	msg := api.Message{
		Role:     "tool",
		ToolName: "read_file",
		Content:  "[Cleared read_file for internal/agent/foo.go: 30 lines. Re-run read_file if needed.]\n" + strings.Repeat("extra\n", 20),
	}

	truncated, metrics := truncateToolResult(msg, 5, 2, 1)
	if truncated.Content != msg.Content {
		t.Fatalf("already cleared content should skip truncate, got %q", truncated.Content)
	}
	if metrics != (CompactionMetrics{}) {
		t.Fatalf("unexpected metrics for already cleared content: %+v", metrics)
	}
}
