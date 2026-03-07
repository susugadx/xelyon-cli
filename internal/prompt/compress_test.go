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
