package mutation

import (
	"strings"
	"testing"
)

func TestBuildStringReplacementExecution_ExactMatch(t *testing.T) {
	execution := buildStringReplacementExecution("line1\nline2\nline3", "line2", "REPLACED")
	if execution.failure.hasFailure() {
		t.Fatalf("expected success execution, got failure: %+v", execution.failure)
	}
	if execution.attemptedNormalized {
		t.Fatalf("did not expect normalized path, got: %+v", execution)
	}
	if execution.plan.newContent != "line1\nREPLACED\nline3" {
		t.Fatalf("unexpected new content: %q", execution.plan.newContent)
	}
	if execution.plan.matchStartLine != 2 || execution.plan.matchEndLine != 2 {
		t.Fatalf("unexpected match range: %d-%d", execution.plan.matchStartLine, execution.plan.matchEndLine)
	}
	if execution.plan.replacedEndLine != 2 {
		t.Fatalf("unexpected replaced end line: %d", execution.plan.replacedEndLine)
	}
	if execution.plan.startLineForDisplay != 2 {
		t.Fatalf("unexpected display start line: %d", execution.plan.startLineForDisplay)
	}
}

func TestBuildStringReplacementExecution_MultipleMatchesFailure(t *testing.T) {
	content := "foo\nalpha\nfoo\nbeta"
	execution := buildStringReplacementExecution(content, "foo", "REPLACED")
	if !execution.failure.hasFailure() {
		t.Fatal("expected failure for multiple matches")
	}
	if execution.failure.reason != stringReplacementFailureMultipleMatches {
		t.Fatalf("unexpected failure reason: %v", execution.failure.reason)
	}
	if execution.failure.exactCount != 2 {
		t.Fatalf("unexpected exact count: %d", execution.failure.exactCount)
	}
	message := buildStringReplacementFailure("test.txt", content, "foo", execution.failure)
	if !strings.Contains(message, "Error: old_str appears 2 times in test.txt (must be unique).") {
		t.Fatalf("unexpected failure message: %s", message)
	}
}

func TestBuildStringReplacementExecution_NotFoundFailure(t *testing.T) {
	content := "first\nsecond\nthird"
	execution := buildStringReplacementExecution(content, "missing", "REPLACED")
	if !execution.failure.hasFailure() {
		t.Fatal("expected not-found failure")
	}
	if !execution.attemptedNormalized {
		t.Fatal("expected normalized matching attempt")
	}
	if execution.failure.reason != stringReplacementFailureNotFound {
		t.Fatalf("unexpected failure reason: %v", execution.failure.reason)
	}
	message := buildStringReplacementFailure("test.txt", content, "missing", execution.failure)
	if !strings.Contains(message, "Error: old_str not found in test.txt (tried exact and normalized matching).") {
		t.Fatalf("unexpected failure summary: %s", message)
	}
	if !strings.Contains(message, "Preview: 1:first | 2:second | 3:third") {
		t.Fatalf("unexpected preview summary: %s", message)
	}
}

func TestBuildAppliedStrReplaceResult(t *testing.T) {
	result := buildAppliedStrReplaceResult("test.txt", stringReplacementPlan{
		matchStartLine:      3,
		matchEndLine:        4,
		replacedEndLine:     5,
		usedNormalizedMatch: true,
	})
	expected := "Successfully replaced text in test.txt (lines 3-4 → 3-5, used normalized whitespace matching)"
	if result != expected {
		t.Fatalf("unexpected applied result:\nwant: %s\n got: %s", expected, result)
	}
}
