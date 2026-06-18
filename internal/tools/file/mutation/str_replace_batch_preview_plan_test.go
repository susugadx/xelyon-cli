package mutation

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine"
)

func TestBuildBatchStringReplacementPreviewPlan_Failure(t *testing.T) {
	edits := []replaceengine.Edit{
		{OldStr: "aaa", NewStr: "AAA"},
		{OldStr: "missing", NewStr: "XXX"},
	}

	plan := buildBatchStringReplacementPreviewPlan("test.txt", "aaa\nbbb\nccc", edits)
	if _, ok := plan.outcome.Failure(); !ok {
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
	edits := []replaceengine.Edit{
		{OldStr: "hello", NewStr: "hi"},
		{OldStr: "hi", NewStr: "hello"},
	}

	plan := buildBatchStringReplacementPreviewPlan("test.txt", "hello world", edits)
	if failure, ok := plan.outcome.Failure(); ok {
		t.Fatalf("unexpected failure outcome: %+v", failure)
	}
	if plan.terminalResult.status != fileMutationStatusNoop {
		t.Fatalf("expected noop terminal result, got status=%v", plan.terminalResult.status)
	}
	if plan.terminalResult.message != "No changes after applying all edits" {
		t.Fatalf("unexpected noop message: %s", plan.terminalResult.message)
	}
}

func TestBuildBatchStringReplacementPreviewPlan_Success(t *testing.T) {
	plan := buildBatchStringReplacementPreviewPlan("test.txt", "hello world", []replaceengine.Edit{
		{OldStr: "hello", NewStr: "hi"},
	})

	if plan.terminalResult.IsTerminal() {
		t.Fatalf("expected non-terminal preview result, got: %+v", plan.terminalResult)
	}
	if failure, ok := plan.outcome.Failure(); ok {
		t.Fatalf("unexpected failure outcome: %+v", failure)
	}
	if plan.outcome.Plan().NewContent() != "hi world" {
		t.Fatalf("unexpected new content: %q", plan.outcome.Plan().NewContent())
	}
}
