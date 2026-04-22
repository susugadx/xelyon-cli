package file

import (
	"strings"
	"testing"
)

func TestBuildBatchStringReplacementPreviewPlan_Failure(t *testing.T) {
	edits := []EditEntry{
		{OldStr: "aaa", NewStr: "AAA"},
		{OldStr: "missing", NewStr: "XXX"},
	}

	plan := buildBatchStringReplacementPreviewPlan("test.txt", "aaa\nbbb\nccc", edits)
	if plan.outcome.failure == nil {
		t.Fatal("expected failure outcome")
	}
	if plan.terminalResult.status != fileMutationStatusError {
		t.Fatalf("expected error terminal result, got status=%v", plan.terminalResult.status)
	}
	if !strings.Contains(plan.terminalResult.message, "edits[1].old_str not found in test.txt") {
		t.Fatalf("unexpected error message: %s", plan.terminalResult.message)
	}
}

func TestBuildBatchStringReplacementPreviewPlan_Noop(t *testing.T) {
	edits := []EditEntry{
		{OldStr: "hello", NewStr: "hi"},
		{OldStr: "hi", NewStr: "hello"},
	}

	plan := buildBatchStringReplacementPreviewPlan("test.txt", "hello world", edits)
	if plan.outcome.failure != nil {
		t.Fatalf("unexpected failure outcome: %+v", plan.outcome.failure)
	}
	if plan.terminalResult.status != fileMutationStatusNoop {
		t.Fatalf("expected noop terminal result, got status=%v", plan.terminalResult.status)
	}
	if plan.terminalResult.message != "No changes after applying all edits" {
		t.Fatalf("unexpected noop message: %s", plan.terminalResult.message)
	}
}

func TestBuildBatchStringReplacementPreviewPlan_Success(t *testing.T) {
	plan := buildBatchStringReplacementPreviewPlan("test.txt", "hello world", []EditEntry{
		{OldStr: "hello", NewStr: "hi"},
	})

	if plan.terminalResult.IsTerminal() {
		t.Fatalf("expected non-terminal preview result, got: %+v", plan.terminalResult)
	}
	if plan.outcome.failure != nil {
		t.Fatalf("unexpected failure outcome: %+v", plan.outcome.failure)
	}
	if plan.outcome.plan.newContent != "hi world" {
		t.Fatalf("unexpected new content: %q", plan.outcome.plan.newContent)
	}
}
