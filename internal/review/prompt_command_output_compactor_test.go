package review

import (
	"fmt"
	"strings"
	"testing"
)

func TestReviewPromptCommandOutputCompactorBuildsValidationPlaceholder(t *testing.T) {
	stats := newReviewPromptReductionStats(ReviewPromptReductionModeApply)
	compacted, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeApply, stats).CompactCommandOutput(
		"go test ./...",
		strings.Join([]string{
			"ok   github.com/susugadx/xelyon-cli/internal/review 0.123s",
			strings.Repeat("verbose detail that should not be sent after compaction\n", 80),
		}, "\n"),
	)
	if !ok {
		t.Fatal("CompactCommandOutput() ok = false, want true")
	}
	if compacted.Classifier != "validation" {
		t.Fatalf("Classifier = %q, want validation", compacted.Classifier)
	}
	if !strings.Contains(compacted.Output, `[omitted old successful validation command output; command="go test ./..."; exit=0; classifier=validation`) {
		t.Fatalf("Output missing validation placeholder:\n%s", compacted.Output)
	}
	if strings.Contains(compacted.Output, "verbose detail that should not be sent") {
		t.Fatalf("Output leaked raw verbose detail:\n%s", compacted.Output)
	}
	report := stats.reportValue()
	if report.CandidateCount != 1 || report.ReplacedCount != 1 || report.ReplacementSavedBytes <= 0 || report.ClassifierCounts["validation"] != 1 {
		t.Fatalf("stats report = %#v, want one applied validation compact with savings", report)
	}
}

func TestReviewPromptCommandOutputCompactorCompactsFailureAndRedactsSecrets(t *testing.T) {
	lines := []string{
		"--- FAIL: TestTokenLeak (0.00s)",
		"    auth_test.go:12: api_key=super-secret-token",
		"FAIL",
		"process exited with code 1",
	}
	for i := 0; i < 90; i++ {
		lines = append(lines, "noise line without diagnostic value")
	}

	stats := newReviewPromptReductionStats(ReviewPromptReductionModeApply)
	compacted, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeApply, stats).CompactCommandOutput("go test ./...", strings.Join(lines, "\n"))
	if !ok {
		t.Fatal("CompactCommandOutput() ok = false, want true")
	}
	if compacted.Classifier != "validation_failure" {
		t.Fatalf("Classifier = %q, want validation_failure", compacted.Classifier)
	}
	if !strings.Contains(compacted.Output, "[compacted old failed command output") {
		t.Fatalf("Output missing failure compact header:\n%s", compacted.Output)
	}
	if !strings.Contains(compacted.Output, "api_key=[redacted]") {
		t.Fatalf("Output missing redacted secret assignment:\n%s", compacted.Output)
	}
	if strings.Contains(compacted.Output, "super-secret-token") {
		t.Fatalf("Output leaked secret:\n%s", compacted.Output)
	}
}

func TestReviewPromptCommandOutputCompactorKeepsPassingExpectedExceptionLogSuccessful(t *testing.T) {
	for _, command := range []string{"go test ./...", "make ci-check"} {
		t.Run(command, func(t *testing.T) {
			stats := newReviewPromptReductionStats(ReviewPromptReductionModeApply)
			compacted, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeApply, stats).CompactCommandOutput(
				command,
				strings.Join([]string{
					"=== RUN   TestExpectedExceptionPath",
					"expected exception: sentinel value",
					"Traceback (most recent call last):",
					"ValueError: expected sentinel",
					"panic: expected sentinel recovered by test",
					"ok   github.com/susugadx/xelyon-cli/internal/review 0.123s",
					"Process exited with code 0",
					strings.Repeat("verbose passing test detail\n", 90),
				}, "\n"),
			)
			if !ok {
				t.Fatal("CompactCommandOutput() ok = false, want true")
			}
			if compacted.Classifier != "validation" {
				t.Fatalf("Classifier = %q, want validation", compacted.Classifier)
			}
			for _, reject := range []string{"validation_failure", "lint_failure", "build_failure"} {
				if strings.Contains(compacted.Output, reject) {
					t.Fatalf("Output mislabeled passing expected-exception log with %q:\n%s", reject, compacted.Output)
				}
			}
			report := stats.reportValue()
			if report.ClassifierCounts["validation"] != 1 || report.ClassifierCounts["validation_failure"] != 0 || report.ClassifierCounts["lint_failure"] != 0 {
				t.Fatalf("stats report = %#v, want validation only", report)
			}
		})
	}
}

func TestReviewPromptCommandOutputCompactorCompactsPackageTextualFailureAsFailure(t *testing.T) {
	stats := newReviewPromptReductionStats(ReviewPromptReductionModeApply)
	compacted, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeApply, stats).CompactCommandOutput(
		"npm install",
		strings.Repeat("error: dependency resolution failed\ncommand failed: npm install\n", 120),
	)
	if !ok {
		t.Fatal("CompactCommandOutput() ok = false, want true")
	}
	if compacted.Classifier != "package_failure" {
		t.Fatalf("Classifier = %q, want package_failure", compacted.Classifier)
	}
	if !strings.Contains(compacted.Output, "[compacted old failed command output") {
		t.Fatalf("Output missing failed command compact:\n%s", compacted.Output)
	}
	for _, reject := range []string{"successful side-effect command output", "exit=0"} {
		if strings.Contains(compacted.Output, reject) {
			t.Fatalf("Output mislabeled package failure with %q:\n%s", reject, compacted.Output)
		}
	}
	report := stats.reportValue()
	if report.ClassifierCounts["package_failure"] != 1 || report.ReplacedCount != 1 {
		t.Fatalf("stats report = %#v, want one applied package failure compact", report)
	}
}

func TestReviewPromptCommandOutputCompactorCompactsGitStatusShortColumns(t *testing.T) {
	var lines []string
	for i := 1; i <= 120; i++ {
		lines = append(lines,
			fmt.Sprintf(" M unstaged-%03d.go", i),
			fmt.Sprintf("M  staged-%03d.go", i),
		)
	}
	stats := newReviewPromptReductionStats(ReviewPromptReductionModeApply)
	compacted, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeApply, stats).CompactCommandOutput(
		"git status --short",
		strings.Join(lines, "\n"),
	)
	if !ok {
		t.Fatal("CompactCommandOutput() ok = false, want true")
	}
	if compacted.Classifier != "git_status" {
		t.Fatalf("Classifier = %q, want git_status", compacted.Classifier)
	}
	for _, want := range []string{"staged=120", "unstaged=120", "staged:", "staged-001.go", "unstaged:", "unstaged-001.go"} {
		if !strings.Contains(compacted.Output, want) {
			t.Fatalf("Output missing %q:\n%s", want, compacted.Output)
		}
	}
	stagedSection := reviewPromptCommandOutputTestSection(compacted.Output, "staged:")
	if strings.Contains(stagedSection, "unstaged-001.go") {
		t.Fatalf("staged section contains unstaged entry:\n%s", compacted.Output)
	}
}

func TestReviewPromptCommandOutputCompactorKeepsDataBearingCommandOutput(t *testing.T) {
	stats := newReviewPromptReductionStats(ReviewPromptReductionModeApply)
	_, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeApply, stats).CompactCommandOutput(
		"curl 'https://api.example.test/items?token=secret-value'",
		strings.Repeat("api-result line with unrecoverable body\n", 120),
	)
	if ok {
		t.Fatal("CompactCommandOutput() ok = true, want false for data-bearing command output")
	}
	report := stats.reportValue()
	if report.CandidateCount != 0 || report.ReplacedCount != 0 {
		t.Fatalf("stats report = %#v, want no review prompt compaction for data-bearing output", report)
	}
}

func TestReviewPromptCommandOutputCompactorDryRunRecordsWithoutChangingPrompt(t *testing.T) {
	stats := newReviewPromptReductionStats(ReviewPromptReductionModeDryRun)
	_, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeDryRun, stats).CompactCommandOutput(
		"go test ./...",
		"ok   github.com/susugadx/xelyon-cli/internal/review 0.123s\n"+strings.Repeat("verbose detail\n", 80),
	)
	if ok {
		t.Fatal("CompactCommandOutput() ok = true, want false in dry-run")
	}
	report := stats.reportValue()
	if report.CandidateCount != 1 || report.ReplacedCount != 0 || report.EstimatedSavedBytes <= 0 || report.ReplacementSavedBytes != 0 {
		t.Fatalf("stats report = %#v, want dry-run candidate without replacement", report)
	}
}

func TestReviewPromptCommandOutputCompactorSkipsWhenReplacementDoesNotSavePrompt(t *testing.T) {
	stats := newReviewPromptReductionStats(ReviewPromptReductionModeApply)
	_, ok := newReviewPromptCommandOutputCompactor(ReviewPromptReductionModeApply, stats).CompactCommandOutput(
		"go test ./...",
		"ok   github.com/susugadx/xelyon-cli/internal/review 0.123s",
	)
	if ok {
		t.Fatal("CompactCommandOutput() ok = true, want false for below-threshold output")
	}
	report := stats.reportValue()
	if report.CandidateCount != 0 || report.ReplacedCount != 0 {
		t.Fatalf("stats report = %#v, want no candidate below threshold", report)
	}
}

func reviewPromptCommandOutputTestSection(text, heading string) string {
	start := strings.Index(text, heading)
	if start < 0 {
		return ""
	}
	rest := text[start+len(heading):]
	for _, next := range []string{"\nstaged:", "\nunstaged:", "\nuntracked:", "\nconflicted:", "\nother:"} {
		if index := strings.Index(rest, next); index >= 0 {
			return rest[:index]
		}
	}
	return rest
}
