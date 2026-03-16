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

func makeSingleTurnToolHistory(toolCount, lines int, toolName string) []api.Message {
	history := []api.Message{{Role: "user", Content: "task"}}
	content := makeLargeContent(lines)

	for i := 1; i <= toolCount; i++ {
		history = append(history,
			api.Message{Role: "assistant", Content: fmt.Sprintf("assistant-%d", i)},
			api.Message{
				Role:       "tool",
				Content:    content,
				ToolCallID: fmt.Sprintf("c%d", i),
				ToolName:   toolName,
			},
		)
	}

	history = append(history, api.Message{Role: "assistant", Content: "done"})
	return history
}

func makeSearchCodeContent() string {
	return "Found 7 matches in 2 files:\n" + strings.Repeat("  internal/agent/foo.go — 1 match\n", 12)
}

func makeListDirContent() string {
	return strings.Join([]string{
		"📂 /tmp/project",
		"summary: depth=2, dirs=3, files=5",
		"dirs: cmd/, internal/, docs/",
		"files: README.md (120 bytes), go.mod (80 bytes), go.sum (60 bytes), Makefile (40 bytes), AGENTS.md (20 bytes)",
		"subtrees: 2 shown (+1 more)",
		"- internal -> dirs=2, files=8",
		"  dirs: agent/, tools/",
		"  files: compact.go (100 bytes), stats.go (80 bytes), history.go (70 bytes), extra.go (60 bytes), more.go (50 bytes), keep.go (40 bytes)",
	}, "\n")
}

func makeInspectContent(symbol, path string) string {
	return fmt.Sprintf("── func %s (L10-L40) in %s ──\n%s", symbol, path, strings.Repeat("10: body line\n", 30))
}

func makeWebSearchContent() string {
	return strings.Join([]string{
		"Summary:",
		"- Point one with enough detail to make the result meaningfully long.",
		"- Point two with additional explanation for the search summary.",
		"- Point three describing the context around the search results.",
		"- Point four highlighting another relevant detail from the web.",
		"- Point five covering the remaining background information.",
		"Sources:",
		"1. Example One",
		"   URL: https://example.com/one",
		"",
		"2. Example Two",
		"   URL: https://example.com/two",
	}, "\n")
}

func makePhase0History(toolName, toolCallID, argsJSON, content string, age int) []api.Message {
	if age < 1 {
		age = 1
	}

	history := []api.Message{{Role: "user", Content: "task"}}
	targetAssistant := api.Message{Role: "assistant", Content: "target"}
	if toolCallID != "" {
		targetAssistant.ToolCalls = []api.OpenAIToolCall{{
			ID: toolCallID,
			Function: api.OpenAIToolCallFunction{
				Name:      toolName,
				Arguments: argsJSON,
			},
		}}
	}
	history = append(history, targetAssistant, api.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		ToolName:   toolName,
	})

	for i := 0; i < age-1; i++ {
		callID := fmt.Sprintf("fill-%d", i)
		history = append(history,
			api.Message{
				Role:    "assistant",
				Content: fmt.Sprintf("fill-%d", i),
				ToolCalls: []api.OpenAIToolCall{{
					ID: callID,
					Function: api.OpenAIToolCallFunction{
						Name:      "str_replace",
						Arguments: fmt.Sprintf(`{"path":"fill-%d.go"}`, i),
					},
				}},
			},
			api.Message{
				Role:       "tool",
				Content:    makeLargeContent(60),
				ToolCallID: callID,
				ToolName:   "str_replace",
			},
		)
	}

	history = append(history, api.Message{Role: "assistant", Content: "done"})
	return history
}

func compactOldToolResultsWithoutPhase0(history []api.Message, maxLines, headLines, tailLines int) []api.Message {
	if len(history) == 0 {
		return history
	}

	lastAssistantIdx := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			lastAssistantIdx = i
			break
		}
	}
	if lastAssistantIdx < 0 {
		result := make([]api.Message, len(history))
		copy(result, history)
		return result
	}

	result := make([]api.Message, len(history))
	copy(result, history)
	toolAge := buildToolAgeMap(history, lastAssistantIdx)

	for i := 0; i < lastAssistantIdx; i++ {
		if result[i].Role != "tool" {
			continue
		}
		if compressed, ok := compressFailedPair(result, i); ok {
			result[i] = compressed
			continue
		}
		age := toolAge[i]
		if age > ultraCompactAge {
			if summary := ultraCompactToolResult(result[i]); summary != "" {
				result[i].Content = summary
				continue
			}
		}
		h, t := graduatedTruncateByAge(age, headLines, tailLines)
		result[i], _ = truncateToolResult(result[i], maxLines, h, t)
	}

	return result
}

func compactedContentLen(history []api.Message) int {
	total := 0
	for _, msg := range history {
		total += len(msg.Content)
	}
	return total
}

func TestToolResultContentRatio(t *testing.T) {
	toolContent := strings.Repeat("t", 80)
	userContent := strings.Repeat("u", 20)

	history := []api.Message{
		{Role: "tool", Content: toolContent},
		{Role: "user", Content: userContent},
	}

	ratio := ToolResultContentRatio(history)
	if ratio <= 0.70 {
		t.Fatalf("ToolResultContentRatio() = %.2f, want > 0.70", ratio)
	}
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

func TestBuildToolAgeMap(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: "tool1", ToolCallID: "c1", ToolName: "read_file"},
		{Role: "assistant", Content: "a2"},
		{Role: "tool", Content: "tool2", ToolCallID: "c2", ToolName: "bash"},
		{Role: "user", Content: "still same turn"},
		{Role: "tool", Content: "tool3", ToolCallID: "c3", ToolName: "read_file"},
		{Role: "assistant", Content: "done"},
	}

	ages := buildToolAgeMap(history, 7)

	if len(ages) != 3 {
		t.Fatalf("expected 3 tool ages, got %d", len(ages))
	}
	if ages[6] != 1 {
		t.Fatalf("tool at index 6 age = %d, want 1", ages[6])
	}
	if ages[4] != 2 {
		t.Fatalf("tool at index 4 age = %d, want 2", ages[4])
	}
	if ages[2] != 3 {
		t.Fatalf("tool at index 2 age = %d, want 3", ages[2])
	}
}

func TestGraduatedTruncateByAge(t *testing.T) {
	tests := []struct {
		age      int
		wantHead int
		wantTail int
	}{
		{age: 1, wantHead: 20, wantTail: 5},
		{age: 5, wantHead: 20, wantTail: 5},
		{age: 6, wantHead: 10, wantTail: 3},
		{age: 15, wantHead: 10, wantTail: 3},
		{age: 16, wantHead: 5, wantTail: 2},
		{age: 99, wantHead: 5, wantTail: 2},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("age-%d", tt.age), func(t *testing.T) {
			head, tail := graduatedTruncateByAge(tt.age, 20, 5)
			if head != tt.wantHead || tail != tt.wantTail {
				t.Fatalf("graduatedTruncateByAge(%d) = (%d, %d), want (%d, %d)",
					tt.age, head, tail, tt.wantHead, tt.wantTail)
			}
		})
	}
}

func TestCompactOldToolResults_GraduatedCompression(t *testing.T) {
	history := makeSingleTurnToolHistory(16, 100, "str_replace")

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	oldest := result[2].Content
	medium := result[22].Content
	recent := result[32].Content

	if !strings.Contains(oldest, "Old ") {
		t.Fatalf("expected oldest tool result to be ultra-compacted, got %q", oldest)
	}
	if !strings.Contains(medium, "truncated") || !strings.Contains(recent, "truncated") {
		t.Fatal("expected medium/recent tool results to be truncated")
	}
	if len(oldest) >= len(medium) {
		t.Fatalf("expected age 16 compaction to be shorter than age 6 truncation: %d >= %d", len(oldest), len(medium))
	}
	if len(medium) >= len(recent) {
		t.Fatalf("expected age 6 truncation to be shorter than age 1 truncation: %d >= %d", len(medium), len(recent))
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

	if metrics.Phase0Clears != 0 {
		t.Fatalf("Phase0Clears = %d, want 0", metrics.Phase0Clears)
	}
	if metrics.FailedPairCompressions != 1 {
		t.Fatalf("FailedPairCompressions = %d, want 1", metrics.FailedPairCompressions)
	}
	if metrics.TruncationCount != 1 {
		t.Fatalf("TruncationCount = %d, want 1", metrics.TruncationCount)
	}
	if metrics.ErrorCompressions != 1 {
		t.Fatalf("ErrorCompressions = %d, want 1", metrics.ErrorCompressions)
	}
}

func TestPhase0Clear_ReadFile_FromArgs(t *testing.T) {
	history := makePhase0History("read_file", "c1", `{"path":"internal/agent/foo.go"}`, makeLargeContent(60), phase0ClearAge)

	result, metrics := CompactOldToolResults(history, 50, 20, 5)

	want := "[Cleared read_file for internal/agent/foo.go: 60 lines. Re-run read_file if needed.]"
	if result[2].Content != want {
		t.Fatalf("Phase 0 placeholder = %q, want %q", result[2].Content, want)
	}
	if metrics.Phase0Clears != 1 {
		t.Fatalf("Phase0Clears = %d, want 1", metrics.Phase0Clears)
	}
}

func TestPhase0Clear_ReadFile_FallbackToContent(t *testing.T) {
	content := "File: internal/agent/fallback.go\n" + strings.Repeat("body line\n", 40)
	history := makePhase0History("read_file", "", "", content, phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if !strings.Contains(result[2].Content, "internal/agent/fallback.go") {
		t.Fatalf("expected fallback path in placeholder, got %q", result[2].Content)
	}
}

func TestPhase0Clear_SearchCode_WithPattern(t *testing.T) {
	history := makePhase0History("search_code", "c1", `{"pattern":"CompactOldToolResults"}`, makeSearchCodeContent(), phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if !strings.Contains(result[2].Content, `for "CompactOldToolResults"`) {
		t.Fatalf("expected pattern in placeholder, got %q", result[2].Content)
	}
	if !strings.Contains(result[2].Content, "Found 7 matches in 2 files") {
		t.Fatalf("expected search summary in placeholder, got %q", result[2].Content)
	}
}

func TestPhase0Clear_FromParallelAssistantToolCalls(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "parallel", ToolCalls: []api.OpenAIToolCall{
			{
				ID: "c1",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"internal/agent/parallel.go"}`,
				},
			},
			{
				ID: "c2",
				Function: api.OpenAIToolCallFunction{
					Name:      "search_code",
					Arguments: `{"pattern":"parallelLookup"}`,
				},
			},
		}},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "read_file"},
		{Role: "tool", Content: makeSearchCodeContent(), ToolCallID: "c2", ToolName: "search_code"},
		{Role: "assistant", Content: "fill-1", ToolCalls: []api.OpenAIToolCall{
			{ID: "f1", Function: api.OpenAIToolCallFunction{Name: "str_replace", Arguments: `{"path":"fill-1.go"}`}},
		}},
		{Role: "tool", Content: large, ToolCallID: "f1", ToolName: "str_replace"},
		{Role: "assistant", Content: "fill-2", ToolCalls: []api.OpenAIToolCall{
			{ID: "f2", Function: api.OpenAIToolCallFunction{Name: "str_replace", Arguments: `{"path":"fill-2.go"}`}},
		}},
		{Role: "tool", Content: large, ToolCallID: "f2", ToolName: "str_replace"},
		{Role: "assistant", Content: "done"},
	}

	result, metrics := CompactOldToolResults(history, 50, 20, 5)

	if result[2].Content != "[Cleared read_file for internal/agent/parallel.go: 60 lines. Re-run read_file if needed.]" {
		t.Fatalf("read_file placeholder = %q", result[2].Content)
	}
	if !strings.Contains(result[3].Content, `for "parallelLookup"`) {
		t.Fatalf("search_code placeholder should use matching ToolCallID args, got %q", result[3].Content)
	}
	if metrics.Phase0Clears != 2 {
		t.Fatalf("Phase0Clears = %d, want 2", metrics.Phase0Clears)
	}
}

func TestPhase0Clear_ListDir(t *testing.T) {
	history := makePhase0History("list_dir", "c1", `{"path":"/tmp/project"}`, makeListDirContent(), phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	want := "[Cleared list_dir for /tmp/project: 8 entries. Re-run list_dir if needed.]"
	if result[2].Content != want {
		t.Fatalf("Phase 0 placeholder = %q, want %q", result[2].Content, want)
	}
}

func TestPhase0Clear_InspectSymbol(t *testing.T) {
	history := makePhase0History("inspect_symbol", "c1", `{"symbol":"Build","path":"internal/agent/foo.go"}`, makeInspectContent("Build", "internal/agent/foo.go"), phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	want := `[Cleared inspect_symbol for "Build" in internal/agent/foo.go. Re-run inspect_symbol if needed.]`
	if result[2].Content != want {
		t.Fatalf("Phase 0 placeholder = %q, want %q", result[2].Content, want)
	}
}

func TestPhase0Clear_Bash(t *testing.T) {
	content := "Command interrupted.\nPartial output:\n" + strings.Repeat("line\n", 40)
	history := makePhase0History("bash", "c1", `{"command":"go test ./... -run TestPhase0"}`, content, phase0ClearAgeCautious)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if !strings.Contains(result[2].Content, `for "go test ./... -run TestPhase0"`) {
		t.Fatalf("expected command in placeholder, got %q", result[2].Content)
	}
	if !strings.Contains(result[2].Content, "error") {
		t.Fatalf("expected error status in placeholder, got %q", result[2].Content)
	}
}

func TestPhase0Clear_WebSearch(t *testing.T) {
	history := makePhase0History("web_search", "c1", `{"query":"golang slices package"}`, makeWebSearchContent(), phase0ClearAgeCautious)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	want := `[Cleared web_search for "golang slices package": 2 sources. Re-run web_search for latest results.]`
	if result[2].Content != want {
		t.Fatalf("Phase 0 placeholder = %q, want %q", result[2].Content, want)
	}
}

func TestPhase0Clear_ProtectsRecentResults(t *testing.T) {
	readHistory := makePhase0History("read_file", "c1", `{"path":"recent.go"}`, makeLargeContent(60), phase0ClearAge-1)
	readResult, _ := CompactOldToolResults(readHistory, 50, 20, 5)
	if strings.HasPrefix(readResult[2].Content, "[Cleared ") {
		t.Fatalf("recent Tier 1 result should not be cleared, got %q", readResult[2].Content)
	}

	bashHistory := makePhase0History("bash", "c1", `{"command":"ls -la"}`, strings.Repeat("output line\n", 60), phase0ClearAgeCautious-1)
	bashResult, _ := CompactOldToolResults(bashHistory, 50, 20, 5)
	if strings.HasPrefix(bashResult[2].Content, "[Cleared ") {
		t.Fatalf("recent Tier 2 result should not be cleared, got %q", bashResult[2].Content)
	}
}

func TestPhase0Clear_ProtectsUnreadResults(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "step 1", ToolCalls: []api.OpenAIToolCall{
			{ID: "c1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"read.go"}`}},
		}},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "read_file"},
		{Role: "assistant", Content: "done"},
		{Role: "tool", Content: large, ToolCallID: "c2", ToolName: "read_file"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if result[4].Content != large {
		t.Fatalf("expected unread tool result to be preserved, got %q", result[4].Content)
	}
}

func TestPhase0Clear_ProtectsShortResults(t *testing.T) {
	content := strings.Repeat("short\n", 20)
	history := makePhase0History("read_file", "c1", `{"path":"short.go"}`, content, phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if result[2].Content != content {
		t.Fatalf("short result should be preserved, got %q", result[2].Content)
	}
}

func TestPhase0Clear_ProtectsWriteTools(t *testing.T) {
	history := makePhase0History("write_file", "c1", `{"path":"out.go"}`, makeLargeContent(60), phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if strings.HasPrefix(result[2].Content, "[Cleared ") {
		t.Fatalf("write_file should not be cleared, got %q", result[2].Content)
	}
}

func TestPhase0Clear_ProtectsMCPTools(t *testing.T) {
	history := makePhase0History("mcp__notion__search", "c1", `{"query":"compact"}`, makeLargeContent(60), phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if strings.HasPrefix(result[2].Content, "[Cleared ") {
		t.Fatalf("MCP tool should not be cleared, got %q", result[2].Content)
	}
}

func TestPhase0Clear_ProtectsUnknownTools(t *testing.T) {
	history := makePhase0History("custom_tool", "c1", `{"target":"x"}`, makeLargeContent(60), phase0ClearAge)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if strings.HasPrefix(result[2].Content, "[Cleared ") {
		t.Fatalf("unknown tool should not be cleared, got %q", result[2].Content)
	}
}

func TestPhase0Clear_FailedPairTakesPriority(t *testing.T) {
	errorContent := "Error: failed to read file\n" + strings.Repeat("detail line\n", 30)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "try", ToolCalls: []api.OpenAIToolCall{
			{ID: "c1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"broken.go"}`}},
		}},
		{Role: "tool", Content: errorContent, ToolCallID: "c1", ToolName: "read_file"},
		{Role: "assistant", Content: "retry", ToolCalls: []api.OpenAIToolCall{
			{ID: "c2", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"broken.go"}`}},
		}},
		{Role: "tool", Content: makeLargeContent(60), ToolCallID: "c2", ToolName: "read_file"},
		{Role: "assistant", Content: "fill", ToolCalls: []api.OpenAIToolCall{
			{ID: "c3", Function: api.OpenAIToolCallFunction{Name: "str_replace", Arguments: `{"path":"fill.go"}`}},
		}},
		{Role: "tool", Content: makeLargeContent(60), ToolCallID: "c3", ToolName: "str_replace"},
		{Role: "assistant", Content: "done"},
	}

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if !strings.HasPrefix(result[2].Content, "[Failed: read_file") {
		t.Fatalf("failed pair should win before Phase 0, got %q", result[2].Content)
	}
}

func TestPhase0Clear_ThenPhase1Fallback(t *testing.T) {
	history := makePhase0History("read_file", "c1", `{"path":"recent.go"}`, makeLargeContent(60), phase0ClearAge-1)

	result, _ := CompactOldToolResults(history, 50, 20, 5)

	if strings.HasPrefix(result[2].Content, "[Cleared ") {
		t.Fatalf("age-protected result should not be cleared, got %q", result[2].Content)
	}
	if !strings.Contains(result[2].Content, "truncated") {
		t.Fatalf("expected Phase 1 fallback truncation, got %q", result[2].Content)
	}
}

func TestPhase0Clear_SkipsAlreadyCleared(t *testing.T) {
	msg := api.Message{
		Role:     "tool",
		ToolName: "read_file",
		Content:  "[Cleared read_file for internal/agent/foo.go: 30 lines. Re-run read_file if needed.]\n" + strings.Repeat("extra\n", 20),
	}

	if _, ok := phase0Clear(msg, phase0ClearAge, nil); ok {
		t.Fatal("already cleared content should not be cleared again")
	}
	if got := ultraCompactToolResult(msg); got != "" {
		t.Fatalf("already cleared content should skip ultra-compact, got %q", got)
	}
	truncated, metrics := truncateToolResult(msg, 5, 2, 1)
	if truncated.Content != msg.Content {
		t.Fatalf("already cleared content should skip truncate, got %q", truncated.Content)
	}
	if metrics != (CompactionMetrics{}) {
		t.Fatalf("unexpected metrics for already cleared content: %+v", metrics)
	}
}

func TestPhase0Clear_MetricsIndependent(t *testing.T) {
	large := makeLargeContent(60)
	history := []api.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "read", ToolCalls: []api.OpenAIToolCall{
			{ID: "c1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"a.go"}`}},
		}},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "read_file"},
		{Role: "assistant", Content: "search", ToolCalls: []api.OpenAIToolCall{
			{ID: "c2", Function: api.OpenAIToolCallFunction{Name: "search_code", Arguments: `{"pattern":"foo"}`}},
		}},
		{Role: "tool", Content: makeSearchCodeContent(), ToolCallID: "c2", ToolName: "search_code"},
		{Role: "assistant", Content: "list", ToolCalls: []api.OpenAIToolCall{
			{ID: "c3", Function: api.OpenAIToolCallFunction{Name: "list_dir", Arguments: `{"path":"."}`}},
		}},
		{Role: "tool", Content: makeListDirContent(), ToolCallID: "c3", ToolName: "list_dir"},
		{Role: "assistant", Content: "edit", ToolCalls: []api.OpenAIToolCall{
			{ID: "c4", Function: api.OpenAIToolCallFunction{Name: "str_replace"}},
		}},
		{Role: "tool", Content: large, ToolCallID: "c4", ToolName: "str_replace"},
		{Role: "assistant", Content: "custom", ToolCalls: []api.OpenAIToolCall{
			{ID: "c5", Function: api.OpenAIToolCallFunction{Name: "custom_tool"}},
		}},
		{Role: "tool", Content: large, ToolCallID: "c5", ToolName: "custom_tool"},
		{Role: "assistant", Content: "done"},
	}

	_, metrics := CompactOldToolResults(history, 50, 20, 5)

	if metrics.Phase0Clears != 3 {
		t.Fatalf("Phase0Clears = %d, want 3", metrics.Phase0Clears)
	}
	if metrics.TruncationCount != 2 {
		t.Fatalf("TruncationCount = %d, want 2", metrics.TruncationCount)
	}
}

func TestCompactOldToolResults_Phase0TokenReduction(t *testing.T) {
	var history []api.Message
	history = append(history, api.Message{Role: "user", Content: "task"})

	addTool := func(id, toolName, argsJSON, content string) {
		history = append(history,
			api.Message{
				Role:    "assistant",
				Content: id,
				ToolCalls: []api.OpenAIToolCall{{
					ID: id,
					Function: api.OpenAIToolCallFunction{
						Name:      toolName,
						Arguments: argsJSON,
					},
				}},
			},
			api.Message{
				Role:       "tool",
				Content:    content,
				ToolCallID: id,
				ToolName:   toolName,
			},
		)
	}

	for i := 0; i < 8; i++ {
		addTool(fmt.Sprintf("read-%d", i), "read_file", fmt.Sprintf(`{"path":"file%d.go"}`, i), makeLargeContent(120))
	}
	for i := 0; i < 4; i++ {
		addTool(fmt.Sprintf("search-%d", i), "search_code", fmt.Sprintf(`{"pattern":"pattern-%d"}`, i), makeSearchCodeContent())
	}
	for i := 0; i < 4; i++ {
		addTool(fmt.Sprintf("bash-%d", i), "bash", fmt.Sprintf(`{"command":"printf bash-%d"}`, i), strings.Repeat("bash output line\n", 90))
	}
	for i := 0; i < 2; i++ {
		addTool(fmt.Sprintf("inspect-%d", i), "inspect_symbol", fmt.Sprintf(`{"symbol":"Build%d","path":"internal/agent/foo.go"}`, i), makeInspectContent(fmt.Sprintf("Build%d", i), "internal/agent/foo.go"))
	}
	for i := 0; i < 2; i++ {
		addTool(fmt.Sprintf("edit-%d", i), "str_replace", fmt.Sprintf(`{"path":"edit-%d.go"}`, i), strings.Repeat("ok\n", 30))
	}
	history = append(history, api.Message{Role: "assistant", Content: "done"})

	withPhase0, metrics := CompactOldToolResults(history, DefaultMaxLines, DefaultHeadLines, DefaultTailLines)
	withoutPhase0 := compactOldToolResultsWithoutPhase0(history, DefaultMaxLines, DefaultHeadLines, DefaultTailLines)

	withLen := compactedContentLen(withPhase0)
	withoutLen := compactedContentLen(withoutPhase0)
	if withLen >= withoutLen {
		t.Fatalf("Phase 0 should reduce compacted size: with=%d without=%d", withLen, withoutLen)
	}
	if withoutLen-withLen < 1500 {
		t.Fatalf("expected substantial reduction from Phase 0, diff=%d", withoutLen-withLen)
	}
	if metrics.Phase0Clears == 0 {
		t.Fatal("expected Phase 0 to clear at least one result")
	}
}

// ── ultraCompactToolResult tests ──

func TestUltraCompactToolResult_ReadFile(t *testing.T) {
	msg := api.Message{
		Role:     "tool",
		Content:  strings.Repeat("line\n", 50),
		ToolName: "read_file",
	}
	got := ultraCompactToolResult(msg)
	if !strings.Contains(got, "Old read_file") {
		t.Errorf("expected read_file summary, got %q", got)
	}
	if !strings.Contains(got, "50 lines") {
		t.Errorf("expected line count, got %q", got)
	}
}

func TestUltraCompactToolResult_SearchCode(t *testing.T) {
	msg := api.Message{
		Role:     "tool",
		Content:  "Found 10 matches in 3 files:\n" + strings.Repeat("  match\n", 50),
		ToolName: "search_code",
	}
	got := ultraCompactToolResult(msg)
	if !strings.Contains(got, "Found 10 matches") {
		t.Errorf("expected first line preserved, got %q", got)
	}
}

func TestUltraCompactToolResult_ShortContent(t *testing.T) {
	msg := api.Message{
		Role:     "tool",
		Content:  "short",
		ToolName: "read_file",
	}
	got := ultraCompactToolResult(msg)
	if got != "" {
		t.Errorf("short content should not be ultra-compacted, got %q", got)
	}
}

func TestUltraCompactToolResult_Bash(t *testing.T) {
	msg := api.Message{
		Role:     "tool",
		Content:  strings.Repeat("output line\n", 30),
		ToolName: "bash",
	}
	got := ultraCompactToolResult(msg)
	if !strings.Contains(got, "Old bash") {
		t.Errorf("expected bash summary, got %q", got)
	}
}

func TestUltraCompactToolResult_EmptyToolName(t *testing.T) {
	msg := api.Message{
		Role:    "tool",
		Content: strings.Repeat("x\n", 30),
	}
	got := ultraCompactToolResult(msg)
	if got != "" {
		t.Error("empty ToolName should return empty (fallback to graduated truncation)")
	}
}

func TestCompactOldToolResults_UltraCompactTier(t *testing.T) {
	// Build history with a very old tool result (age > ultraCompactAge)
	var history []api.Message

	// Add many assistant-tool pairs to make the first tool result very old
	longContent := strings.Repeat("line content here\n", 50)
	history = append(history, api.Message{
		Role:       "tool",
		Content:    longContent,
		ToolCallID: "old",
		ToolName:   "str_replace",
	})
	// Add enough assistant+tool pairs to push age beyond ultraCompactAge
	for i := 0; i < ultraCompactAge+5; i++ {
		history = append(history, api.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("step %d", i),
		})
		history = append(history, api.Message{
			Role:       "tool",
			Content:    fmt.Sprintf("result %d", i),
			ToolCallID: fmt.Sprintf("t%d", i),
			ToolName:   "str_replace",
		})
	}
	// Final assistant to make all tool results "read"
	history = append(history, api.Message{
		Role:    "assistant",
		Content: "final",
	})

	result, metrics := CompactOldToolResults(history, DefaultMaxLines, DefaultHeadLines, DefaultTailLines)

	// The first (oldest) tool result should be ultra-compacted
	if !strings.Contains(result[0].Content, "[Old str_replace") {
		t.Errorf("expected ultra-compact summary for oldest tool result, got:\n%s", result[0].Content)
	}
	if metrics.TruncationCount == 0 {
		t.Error("expected at least 1 truncation from ultra-compact")
	}
}
