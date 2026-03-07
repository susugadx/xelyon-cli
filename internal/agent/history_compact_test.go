package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// makeLargeContent は指定行数のテスト用コンテンツを生成
func makeLargeContent(lines int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		b.WriteString("line content here\n")
	}
	return b.String()
}

func TestCompactOldToolResults_TruncatesOldLargeResults(t *testing.T) {
	large := makeLargeContent(60)

	history := []api.Message{
		{Role: "user", Content: "turn 1"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "assistant", Content: "a2"},
		{Role: "tool", Content: large, ToolCallID: "c2", ToolName: "read_file"},
		{Role: "user", Content: "turn 3"},
		{Role: "assistant", Content: "a3"},
		{Role: "tool", Content: "small result", ToolCallID: "c3", ToolName: "str_replace"},
		{Role: "user", Content: "turn 4"},
		{Role: "assistant", Content: "a4"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// turn 1 の tool (index 2): old → truncated
	if !strings.Contains(result[2].Content, "truncated") {
		t.Errorf("expected turn 1 tool result to be truncated, got length %d", len(result[2].Content))
	}

	// turn 1 の tool は ToolCallID/ToolName を保持
	if result[2].ToolCallID != "c1" {
		t.Errorf("expected ToolCallID='c1', got '%s'", result[2].ToolCallID)
	}
	if result[2].ToolName != "search_code" {
		t.Errorf("expected ToolName='search_code', got '%s'", result[2].ToolName)
	}
}

func TestCompactOldToolResults_KeepsRecentResults(t *testing.T) {
	large := makeLargeContent(60)

	history := []api.Message{
		{Role: "user", Content: "old turn"},
		{Role: "assistant", Content: "a0"},
		{Role: "tool", Content: large, ToolCallID: "c0", ToolName: "search_code"},
		{Role: "user", Content: "turn 1"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "read_file"},
		{Role: "user", Content: "turn 2"},
		{Role: "assistant", Content: "a2"},
		{Role: "tool", Content: large, ToolCallID: "c2", ToolName: "search_code"},
		{Role: "user", Content: "turn 3"},
		{Role: "assistant", Content: "a3"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// 最新 3 ターン内のツール結果はそのまま
	if strings.Contains(result[5].Content, "truncated") {
		t.Error("turn 1 tool result should NOT be truncated (within keepTurns)")
	}
	if strings.Contains(result[8].Content, "truncated") {
		t.Error("turn 2 tool result should NOT be truncated (within keepTurns)")
	}

	// old turn の tool は truncated
	if !strings.Contains(result[2].Content, "truncated") {
		t.Error("old turn tool result should be truncated")
	}
}

func TestCompactOldToolResults_SmallResultsNotTruncated(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "old turn"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: "small result", ToolCallID: "c1", ToolName: "str_replace"},
		{Role: "user", Content: "turn 2"},
		{Role: "user", Content: "turn 3"},
		{Role: "user", Content: "turn 4"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// 50行以下の小さいツール結果は truncate されない
	if result[2].Content != "small result" {
		t.Errorf("expected small tool result unchanged, got: %s", result[2].Content)
	}
}

func TestCompactOldToolResults_OriginalUnchanged(t *testing.T) {
	large := makeLargeContent(60)
	original := large // コピーを保持

	history := []api.Message{
		{Role: "user", Content: "old turn"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "user", Content: "turn 3"},
		{Role: "user", Content: "turn 4"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// 元の history は変更されていないこと
	if history[1].Content != original {
		t.Error("original history was modified by CompactOldToolResults")
	}

	// result は truncated されていること
	if !strings.Contains(result[1].Content, "truncated") {
		t.Error("expected result to be truncated")
	}
}

func TestCompactOldToolResults_EmptyHistory(t *testing.T) {
	// nil
	result := CompactOldToolResults(nil, 3, 50, 20, 5)
	if result != nil {
		t.Errorf("expected nil for nil history, got length %d", len(result))
	}

	// empty
	result = CompactOldToolResults([]api.Message{}, 3, 50, 20, 5)
	if len(result) != 0 {
		t.Errorf("expected empty for empty history, got length %d", len(result))
	}
}

func TestCompactOldToolResults_NoToolResults(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "how are you"},
		{Role: "assistant", Content: "fine"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// panic しないことと、内容が保持されること
	if len(result) != len(history) {
		t.Errorf("expected %d messages, got %d", len(history), len(result))
	}
	for i, msg := range result {
		if msg.Content != history[i].Content {
			t.Errorf("message %d content mismatch", i)
		}
	}
}

func TestCompactOldToolResults_TurnCounting(t *testing.T) {
	large := makeLargeContent(60)

	// 5ターン
	history := []api.Message{
		{Role: "user", Content: "turn1"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "t1"},
		{Role: "user", Content: "turn2"},
		{Role: "assistant", Content: "a2"},
		{Role: "tool", Content: large, ToolCallID: "c2", ToolName: "t2"},
		{Role: "user", Content: "turn3"},
		{Role: "assistant", Content: "a3"},
		{Role: "tool", Content: large, ToolCallID: "c3", ToolName: "t3"},
		{Role: "user", Content: "turn4"},
		{Role: "assistant", Content: "a4"},
		{Role: "tool", Content: large, ToolCallID: "c4", ToolName: "t4"},
		{Role: "user", Content: "turn5"},
		{Role: "assistant", Content: "a5"},
		{Role: "tool", Content: large, ToolCallID: "c5", ToolName: "t5"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// keepTurns=3 → user3 (index 6) が boundary
	// turn1 tool (index 2): old → truncated
	if !strings.Contains(result[2].Content, "truncated") {
		t.Error("turn1 tool should be truncated")
	}
	// turn2 tool (index 5): old → truncated
	if !strings.Contains(result[5].Content, "truncated") {
		t.Error("turn2 tool should be truncated")
	}
	// turn3 tool (index 8): kept
	if strings.Contains(result[8].Content, "truncated") {
		t.Error("turn3 tool should NOT be truncated")
	}
	// turn4 tool (index 11): kept
	if strings.Contains(result[11].Content, "truncated") {
		t.Error("turn4 tool should NOT be truncated")
	}
	// turn5 tool (index 14): kept
	if strings.Contains(result[14].Content, "truncated") {
		t.Error("turn5 tool should NOT be truncated")
	}
}

func TestCompressErrorResult(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantResult string
		wantOk     bool
	}{
		{
			name:       "No matches found",
			content:    "No matches found",
			wantResult: "No matches found",
			wantOk:     true,
		},
		{
			name:       "No matches found with whitespace",
			content:    "  No matches found  \n",
			wantResult: "No matches found",
			wantOk:     true,
		},
		{
			name:       "Error: pattern not found",
			content:    "Error: pattern not found in file.go\nsome extra details\nmore details",
			wantResult: "Error: pattern not found in file.go",
			wantOk:     true,
		},
		{
			name:       "Error: old_str not found",
			content:    "Error: old_str not found in the file\nExpected:\nfunc foo()\nGot:\nfunc bar()",
			wantResult: "Error: old_str not found in the file",
			wantOk:     true,
		},
		{
			name:       "Error reading file multiline",
			content:    "Error reading file: /path/to/file.go\nopen /path/to/file.go: permission denied",
			wantResult: "Error reading file: /path/to/file.go",
			wantOk:     true,
		},
		{
			name:       "Error reading file single line",
			content:    "Error reading file: /path/to/file.go",
			wantResult: "Error reading file: /path/to/file.go",
			wantOk:     true,
		},
		{
			name:       "Generic Error: prefix",
			content:    "Error: something went wrong\nstack trace line 1\nstack trace line 2",
			wantResult: "Error: something went wrong",
			wantOk:     true,
		},
		{
			name:       "Normal content not compressed",
			content:    "package main\n\nfunc main() {}",
			wantResult: "",
			wantOk:     false,
		},
		{
			name:       "Content containing Error: but not at start",
			content:    "result contains Error: in the middle",
			wantResult: "",
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := compressErrorResult(tt.content)
			if ok != tt.wantOk {
				t.Errorf("compressErrorResult() ok = %v, want %v", ok, tt.wantOk)
			}
			if result != tt.wantResult {
				t.Errorf("compressErrorResult() = %q, want %q", result, tt.wantResult)
			}
		})
	}
}

func TestCompactOldToolResults_CompressesOldErrors(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "turn 1"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: "Error: pattern not found in main.go\nExpected: func foo()\nGot: func bar()", ToolCallID: "c1", ToolName: "str_replace"},
		{Role: "tool", Content: "No matches found", ToolCallID: "c2", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "user", Content: "turn 3"},
		{Role: "user", Content: "turn 4"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// Error: pattern not found は1行に圧縮
	if result[2].Content != "Error: pattern not found in main.go" {
		t.Errorf("expected compressed error, got %q", result[2].Content)
	}

	// No matches found はそのまま（既に短い）
	if result[3].Content != "No matches found" {
		t.Errorf("expected 'No matches found' unchanged, got %q", result[3].Content)
	}
}

func TestCompactOldToolResults_RecentErrorsNotCompressed(t *testing.T) {
	errorContent := "Error: pattern not found in main.go\nExpected: func foo()\nGot: func bar()"
	history := []api.Message{
		{Role: "user", Content: "turn 1"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: errorContent, ToolCallID: "c1", ToolName: "str_replace"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)

	// keepTurns 内なので圧縮されない
	if result[2].Content != errorContent {
		t.Errorf("recent error should not be compressed, got %q", result[2].Content)
	}
}

func TestCompactOldToolResults_TruncatedContentFormat(t *testing.T) {
	large := makeLargeContent(100)

	history := []api.Message{
		{Role: "user", Content: "old turn"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "user", Content: "turn 3"},
		{Role: "user", Content: "turn 4"},
	}

	result := CompactOldToolResults(history, 3, 50, 20, 5)
	content := result[1].Content

	// 先頭行が保持されていること
	if !strings.HasPrefix(content, "line content here\n") {
		t.Error("truncated content should start with head lines")
	}

	// truncation marker が含まれること
	if !strings.Contains(content, "... (truncated: was 100 lines") {
		t.Errorf("expected truncation marker with line count, got: %s", content)
	}

	// トークン推定が含まれること
	if !strings.Contains(content, "tokens)") {
		t.Error("expected token estimate in truncation marker")
	}

	// 末尾行が保持されていること（TrimRight で末尾改行が除去されている）
	if !strings.HasSuffix(content, "line content here") {
		t.Errorf("truncated content should end with tail lines, got suffix: %q",
			content[len(content)-40:])
	}

	// truncated は元より短い
	if len(content) >= len(large) {
		t.Error("truncated content should be shorter than original")
	}
}
