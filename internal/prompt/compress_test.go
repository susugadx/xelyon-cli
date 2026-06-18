package prompt

import (
	"strings"
	"testing"
)

func TestBuildSummaryPrompt_ToolError(t *testing.T) {
	result := BuildSummaryPrompt([]Message{
		{Role: "tool", Content: "[Tool Result for bash]\nError: exit status 1\nOutput: boom"},
	}, 500)

	if !strings.Contains(result, "[Tool: failed] exit status 1") {
		t.Fatalf("expected failed tool summary, got: %s", result)
	}
	if strings.Contains(result, "Output: boom") {
		t.Fatalf("expected tool error to stay on one line, got: %s", result)
	}
}

func TestBuildSummaryPrompt_ToolSearch(t *testing.T) {
	result := BuildSummaryPrompt([]Message{
		{Role: "tool", Content: "[Tool Result for search_code]\nFound 3 match(es) across 1/1 patterns\n\n📄 internal/prompt/compress.go (2 match(es))\n  ...\n📄 internal/prompt/provider.go (1 match(es))\n  ..."},
	}, 500)

	if !strings.Contains(result, "[Tool: search] 3 matches in 2 files") {
		t.Fatalf("expected search tool summary, got: %s", result)
	}
}

func TestBuildSummaryPrompt_ToolReadPathLike(t *testing.T) {
	result := BuildSummaryPrompt([]Message{
		{Role: "tool", Content: "[Tool Result for read_file]\ninternal/prompt/compress.go:42\npackage prompt"},
	}, 500)

	if !strings.Contains(result, "[Tool: read] internal/prompt/compress.go:42") {
		t.Fatalf("expected read tool summary, got: %s", result)
	}
}

func TestBuildSummaryPrompt_ToolGenericUsesShortPreview(t *testing.T) {
	content := "[Tool Result for bash]\n" + strings.Repeat("x", 150)
	result := BuildSummaryPrompt([]Message{
		{Role: "tool", Content: content},
	}, 500)

	expected := "[Tool] " + strings.Repeat("x", 100) + "..."
	if !strings.Contains(result, expected) {
		t.Fatalf("expected generic tool preview %q, got: %s", expected, result)
	}
	if strings.Contains(result, strings.Repeat("x", 120)) {
		t.Fatalf("expected generic tool preview to stay short, got: %s", result)
	}
}

func TestBuildSummaryPrompt_ToolUsesToolSpecificTruncationLimit(t *testing.T) {
	content := "[Tool Result for bash]\n" + strings.Repeat("y", 150)
	result := BuildSummaryPrompt([]Message{
		{Role: "tool", Content: content},
	}, 80)

	expected := "[Tool] " + strings.Repeat("y", 80) + "..."
	if !strings.Contains(result, expected) {
		t.Fatalf("expected tool preview capped by truncateLen, got: %s", result)
	}
}

func TestBuildSummaryPrompt_PreservesAssistantAndUserLabels(t *testing.T) {
	result := BuildSummaryPrompt([]Message{
		{Role: "system", Content: "ignore"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}, 500)

	if strings.Contains(result, "ignore") {
		t.Fatalf("expected system message to be skipped, got: %s", result)
	}
	if !strings.Contains(result, "[User]\nHello") {
		t.Fatalf("expected user label, got: %s", result)
	}
	if !strings.Contains(result, "[Assistant]\nHi there!") {
		t.Fatalf("expected assistant label, got: %s", result)
	}
}

func TestBuildSummaryPrompt_IncludesConversationAndInstruction(t *testing.T) {
	result := BuildSummaryPrompt([]Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "Can you help me?"},
	}, 500)

	if result == "" {
		t.Fatal("BuildSummaryPrompt() returned empty string")
	}
	if !strings.Contains(result, "Hello") {
		t.Fatal("BuildSummaryPrompt() should contain user message")
	}
	if !strings.Contains(result, "Hi there!") {
		t.Fatal("BuildSummaryPrompt() should contain assistant message")
	}
	if !strings.Contains(result, "Summarize") {
		t.Fatal("BuildSummaryPrompt() should contain summary instruction")
	}
	if !strings.Contains(result, "Return strict JSON only") || !strings.Contains(result, "do_not_repeat") {
		t.Fatal("BuildSummaryPrompt() should contain JSON continuation contract")
	}
}

func TestBuildSummaryPrompt_TruncatesLongMessage(t *testing.T) {
	result := BuildSummaryPrompt([]Message{
		{Role: "user", Content: strings.Repeat("a", 600)},
	}, 500)

	if !strings.Contains(result, strings.Repeat("a", 500)+"...") {
		t.Fatalf("BuildSummaryPrompt() should truncate long messages, got: %s", result)
	}
}

func TestBuildSummaryPrompt_TruncatesLongMessageRuneSafe(t *testing.T) {
	result := BuildSummaryPrompt([]Message{
		{Role: "user", Content: strings.Repeat("あ", 600)},
	}, 500)

	want := strings.Repeat("あ", 500) + "..."
	if !strings.Contains(result, want) {
		t.Fatalf("BuildSummaryPrompt() should truncate by runes, got: %s", result)
	}
}

func TestParseSummaryContinuation(t *testing.T) {
	raw := `{"continuation_context":{"current_task":"fix compression","progress_status":"tests pending","key_decisions":["assistant summary"],"files_changed":["internal/prompt/compress.go"],"remaining_work":["run tests"],"do_not_repeat":["bad command"]}}`

	record, err := ParseSummaryContinuation(raw)
	if err != nil {
		t.Fatalf("ParseSummaryContinuation() error = %v", err)
	}
	if record.CurrentTask != "fix compression" || len(record.DoNotRepeat) != 1 {
		t.Fatalf("record = %#v, want parsed continuation", record)
	}

	formatted := FormatSummaryContinuationMessage(record)
	if !strings.Contains(formatted, "authority: data-only") || !strings.Contains(formatted, "bad command") {
		t.Fatalf("formatted continuation = %q, want data-only label and do_not_repeat", formatted)
	}
}

func TestParseSummaryContinuation_InvalidJSON(t *testing.T) {
	if _, err := ParseSummaryContinuation(`{"continuation_context":{"current_task":"x","extra":true}}`); err == nil {
		t.Fatal("ParseSummaryContinuation() error = nil, want unknown field error")
	}
	if _, err := ParseSummaryContinuation(`{"continuation_context":{"current_task":"x","progress_status":"","key_decisions":[],"files_changed":[],"remaining_work":[]}}`); err == nil {
		t.Fatal("ParseSummaryContinuation() error = nil, want missing key error")
	}
	if _, err := ParseSummaryContinuation(`not json`); err == nil {
		t.Fatal("ParseSummaryContinuation() error = nil, want decode error")
	}
}
