package replaceengine

import "testing"

func TestBuildLineRangeReplacementExecution_Success(t *testing.T) {
	execution := BuildLineRangeExecution("line1\nline2\nline3", "A\nB", "2", "3")
	if execution.Failure().HasFailure() {
		t.Fatalf("expected success execution, got failure: %+v", execution.Failure())
	}
	plan := execution.Plan()
	if plan.StartLine() != 2 || plan.EndLine() != 3 {
		t.Fatalf("unexpected range: %d-%d", plan.StartLine(), plan.EndLine())
	}
	if plan.NewContent() != "line1\nA\nB" {
		t.Fatalf("unexpected new content: %q", plan.NewContent())
	}
	if plan.ReplacedEndLine() != 3 {
		t.Fatalf("unexpected replaced end line: %d", plan.ReplacedEndLine())
	}
}

func TestBuildLineRangeReplacementExecution_MissingRange(t *testing.T) {
	execution := BuildLineRangeExecution("line1\nline2\nline3", "A\nB", "", "")
	failure := execution.Failure()
	if !failure.HasFailure() {
		t.Fatal("expected missing-range failure")
	}
	if !failure.IsMissingRange() {
		t.Fatalf("unexpected failure: %+v", failure)
	}
}
