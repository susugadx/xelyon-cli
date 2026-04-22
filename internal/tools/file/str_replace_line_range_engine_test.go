package file

import "testing"

func TestBuildLineRangeReplacementExecution_Success(t *testing.T) {
	execution := buildLineRangeReplacementExecution("line1\nline2\nline3", "A\nB", "2", "3")
	if execution.failure.hasFailure() {
		t.Fatalf("expected success execution, got failure: %+v", execution.failure)
	}
	if execution.plan.startLine != 2 || execution.plan.endLine != 3 {
		t.Fatalf("unexpected range: %d-%d", execution.plan.startLine, execution.plan.endLine)
	}
	if execution.plan.newContent != "line1\nA\nB" {
		t.Fatalf("unexpected new content: %q", execution.plan.newContent)
	}
	if execution.plan.replacedEndLine() != 3 {
		t.Fatalf("unexpected replaced end line: %d", execution.plan.replacedEndLine())
	}
}

func TestBuildLineRangeReplacementExecution_MissingRange(t *testing.T) {
	execution := buildLineRangeReplacementExecution("line1\nline2\nline3", "A\nB", "", "")
	if !execution.failure.hasFailure() {
		t.Fatal("expected missing-range failure")
	}
	if execution.failure.reason != lineRangeReplacementFailureMissingRange {
		t.Fatalf("unexpected failure reason: %v", execution.failure.reason)
	}
}
