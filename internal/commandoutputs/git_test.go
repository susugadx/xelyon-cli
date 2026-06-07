package commandoutputs

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildReplacementGitDiffKeepsRawOutputForArtifactBackedCompaction(t *testing.T) {
	output := strings.Repeat("diff --git a/internal/a.go b/internal/a.go\n@@ -1,2 +1,2 @@ func f()\n-old\n+new\n", 80)

	replacement, reason, ok := BuildReplacement(NewRequest("git diff", output))

	if ok || reason != gitDiffEvidenceKeepReason {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want git diff artifact-backed keep reason", replacement, reason, ok)
	}
	decision := Decide(NewRequest("git diff", output))
	if decision.Action != DecisionArtifactBackedCandidate ||
		decision.SemanticRole != SemanticRoleDataBearing ||
		decision.Classifier != "git_diff" ||
		!decision.ArtifactPolicy.RequiredForApply {
		t.Fatalf("Decision = %#v, want artifact-backed git diff evidence", decision)
	}
}

func TestBuildReplacementGitStatusCompactPreservesShortStatusColumnsForClassification(t *testing.T) {
	var lines []string
	for i := 1; i <= 45; i++ {
		lines = append(lines,
			fmt.Sprintf(" M unstaged-%03d.go", i),
			fmt.Sprintf("M  staged-%03d.go", i),
			fmt.Sprintf("?? untracked-%03d.go", i),
			fmt.Sprintf("UU conflicted-%03d.go", i),
		)
	}
	output := strings.Join(lines, "\n")

	replacement, reason, ok := BuildReplacement(NewRequest("git status --short", output))

	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want git status compact", replacement, reason, ok)
	}
	text := replacement.Text()
	assertSectionContains(t, text, "staged:", "staged-001.go")
	assertSectionContains(t, text, "unstaged:", "unstaged-001.go")
	assertSectionContains(t, text, "untracked:", "untracked-001.go")
	assertSectionContains(t, text, "conflicted:", "conflicted-001.go")
	assertSectionNotContains(t, text, "staged:", "unstaged-001.go")
}
