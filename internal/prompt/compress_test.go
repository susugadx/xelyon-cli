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
	if strings.Contains(result, "continuation_context") {
		t.Fatal("BuildSummaryPrompt() should not contain legacy continuation_context contract")
	}
	if !strings.Contains(result, "xelyon.continuation.v1") {
		t.Fatal("BuildSummaryPrompt() should identify the continuation schema for request routing")
	}
}

func TestBuildSummarySystemPrompt_ContainsContinuationV1Contract(t *testing.T) {
	result := BuildSummarySystemPrompt()
	for _, want := range []string{
		"xelyon.continuation.v1",
		`"acceptance_criteria"`,
		`"explicit_constraints"`,
		`"do_not_repeat"`,
		"Return exactly one JSON object",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("BuildSummarySystemPrompt() missing %q:\n%s", want, result)
		}
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
	record, err := ParseSummaryContinuation(validContinuationV1JSONForPromptTest())
	if err != nil {
		t.Fatalf("ParseSummaryContinuation() error = %v", err)
	}
	if record.Goal != "fix compression" || len(record.DoNotRepeat) != 1 {
		t.Fatalf("record = %#v, want parsed continuation", record)
	}

	formatted := FormatSummaryContinuationMessage(record)
	if !strings.Contains(formatted, "authority: data-only") || !strings.Contains(formatted, "goal: fix compression") || !strings.Contains(formatted, "bad command") {
		t.Fatalf("formatted continuation = %q, want data-only label and do_not_repeat", formatted)
	}
}

func TestParseSummaryContinuation_InvalidJSON(t *testing.T) {
	if _, err := ParseSummaryContinuation(`{"schema_version":"xelyon.continuation.v1","goal":"x","extra":true}`); err == nil {
		t.Fatal("ParseSummaryContinuation() error = nil, want unknown field error")
	}
	if _, err := ParseSummaryContinuation(`{"schema_version":"xelyon.continuation.v1","goal":"x","acceptance_criteria":[],"explicit_constraints":[],"material_assumptions":[],"decisions":[],"files_changed":[],"verification":[],"open_work":[],"blockers":[],"do_not_repeat":[]}`); err == nil {
		t.Fatal("ParseSummaryContinuation() error = nil, want missing key error")
	}
	if _, err := ParseSummaryContinuation(`not json`); err == nil {
		t.Fatal("ParseSummaryContinuation() error = nil, want decode error")
	}
}

func TestParseSummaryContinuationV1RejectsMissingNestedKeys(t *testing.T) {
	base := validContinuationV1JSONForPromptTest()
	tests := []struct {
		name        string
		raw         string
		errContains string
	}{
		{
			name:        "decision reason",
			raw:         strings.Replace(base, `{"decision":"keep parser strict","reason":"provider output boundary","evidence":["internal/prompt/compress.go"]}`, `{"decision":"keep parser strict","evidence":["internal/prompt/compress.go"]}`, 1),
			errContains: "decisions[0] missing keys: reason",
		},
		{
			name:        "files changed summary",
			raw:         strings.Replace(base, `{"path":"internal/prompt/compress.go","summary":"continuation parser"}`, `{"path":"internal/prompt/compress.go"}`, 1),
			errContains: "files_changed[0] missing keys: summary",
		},
		{
			name:        "verification status",
			raw:         strings.Replace(base, `{"command":"go test ./internal/prompt","status":"passed","summary":"prompt tests"}`, `{"command":"go test ./internal/prompt","summary":"prompt tests"}`, 1),
			errContains: "verification[0] missing keys: status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseSummaryContinuation(tt.raw); err == nil || !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ParseSummaryContinuation() error = %v, want %q", err, tt.errContains)
			}
		})
	}
}

func TestParseSummaryContinuationV1AllowsEmptyNestedValues(t *testing.T) {
	raw := `{"schema_version":"xelyon.continuation.v1","goal":"keep context","acceptance_criteria":[],"explicit_constraints":[],"material_assumptions":[],"decisions":[{"decision":"","reason":"","evidence":[]}],"files_changed":[{"path":"","summary":""}],"verification":[{"command":"","status":"","summary":""}],"open_work":[],"blockers":[],"do_not_repeat":[],"relevant_instruction_refs":[]}`

	record, err := ParseSummaryContinuation(raw)
	if err != nil {
		t.Fatalf("ParseSummaryContinuation() error = %v, want nil", err)
	}
	if record.Goal != "keep context" {
		t.Fatalf("Goal = %q, want keep context", record.Goal)
	}
	if len(record.Decisions) != 0 || len(record.FilesChangedV1) != 0 || len(record.Verification) != 0 {
		t.Fatalf("record nested values = decisions:%#v files:%#v verification:%#v, want normalized empty entries removed", record.Decisions, record.FilesChangedV1, record.Verification)
	}
}

func TestParseSummaryContinuation_LegacyWrapperCompatibility(t *testing.T) {
	raw := `{"continuation_context":{"current_task":"fix compression","progress_status":"tests pending","key_decisions":["assistant summary"],"files_changed":["internal/prompt/compress.go"],"remaining_work":["run tests"],"do_not_repeat":["bad command"]}}`

	record, err := ParseSummaryContinuation(raw)
	if err != nil {
		t.Fatalf("ParseSummaryContinuation() legacy error = %v", err)
	}
	if record.CurrentTask != "fix compression" || len(record.DoNotRepeat) != 1 {
		t.Fatalf("record = %#v, want parsed legacy continuation", record)
	}
}

func validContinuationV1JSONForPromptTest() string {
	return `{"schema_version":"xelyon.continuation.v1","goal":"fix compression","acceptance_criteria":["tests pass"],"explicit_constraints":["data only"],"material_assumptions":[],"decisions":[{"decision":"keep parser strict","reason":"provider output boundary","evidence":["internal/prompt/compress.go"]}],"files_changed":[{"path":"internal/prompt/compress.go","summary":"continuation parser"}],"verification":[{"command":"go test ./internal/prompt","status":"passed","summary":"prompt tests"}],"open_work":["run tests"],"blockers":[],"do_not_repeat":["bad command"],"relevant_instruction_refs":[]}`
}
