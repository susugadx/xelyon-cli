package replaceengine

import "testing"

func TestBuildStringReplacementExecution_ExactMatch(t *testing.T) {
	execution := BuildStringExecution("line1\nline2\nline3", "line2", "REPLACED")
	if execution.Failure().HasFailure() {
		t.Fatalf("expected success execution, got failure: %+v", execution.Failure())
	}
	if execution.AttemptedNormalized() {
		t.Fatalf("did not expect normalized path, got: %+v", execution)
	}
	plan := execution.Plan()
	if plan.NewContent() != "line1\nREPLACED\nline3" {
		t.Fatalf("unexpected new content: %q", plan.NewContent())
	}
	if plan.MatchStartLine() != 2 || plan.MatchEndLine() != 2 {
		t.Fatalf("unexpected match range: %d-%d", plan.MatchStartLine(), plan.MatchEndLine())
	}
	if plan.ReplacedEndLine() != 2 {
		t.Fatalf("unexpected replaced end line: %d", plan.ReplacedEndLine())
	}
	if plan.StartLineForDisplay() != 2 {
		t.Fatalf("unexpected display start line: %d", plan.StartLineForDisplay())
	}
}

func TestBuildStringReplacementExecution_MultipleMatchesFailure(t *testing.T) {
	content := "foo\nalpha\nfoo\nbeta"
	execution := BuildStringExecution(content, "foo", "REPLACED")
	failure := execution.Failure()
	if !failure.HasFailure() {
		t.Fatal("expected failure for multiple matches")
	}
	if !failure.IsMultipleMatches() {
		t.Fatalf("unexpected failure: %+v", failure)
	}
	if failure.ExactCount() != 2 {
		t.Fatalf("unexpected exact count: %d", failure.ExactCount())
	}
}

func TestBuildStringReplacementExecution_NotFoundFailure(t *testing.T) {
	content := "first\nsecond\nthird"
	execution := BuildStringExecution(content, "missing", "REPLACED")
	failure := execution.Failure()
	if !failure.HasFailure() {
		t.Fatal("expected not-found failure")
	}
	if !execution.AttemptedNormalized() {
		t.Fatal("expected normalized matching attempt")
	}
	if !failure.IsNotFound() {
		t.Fatalf("unexpected failure: %+v", failure)
	}
}

func TestBuildStringReplacementExecution_NormalizedMatch(t *testing.T) {
	execution := BuildStringExecution("func main() {\n\treturn nil\n}", "func main() {\n  return nil\n}", "func main() {\n\treturn err\n}")
	if execution.Failure().HasFailure() {
		t.Fatalf("expected normalized success, got failure: %+v", execution.Failure())
	}
	if !execution.AttemptedNormalized() {
		t.Fatal("expected normalized matching attempt")
	}
	plan := execution.Plan()
	if !plan.UsedNormalizedMatch() {
		t.Fatal("expected normalized match flag")
	}
	if plan.NewContent() != "func main() {\n\treturn err\n}" {
		t.Fatalf("unexpected new content: %q", plan.NewContent())
	}
}
