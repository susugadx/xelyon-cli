package review

import (
	"reflect"
	"testing"
)

func TestRedactReviewProbeSummariesRedactsUserFacingFieldsAndPreservesStructure(t *testing.T) {
	rawSummaries := []ReviewProbeSummary{
		{
			ProbeID:         "probe-1",
			Mode:            ReviewProbeRepoSandbox,
			Status:          ReviewProbeMutatedWorktree,
			MutatedWorktree: true,
			MutatedFiles: []string{
				"/tmp/repo/internal/review/runner.go",
				"/tmp/probe-workdir/runtime/home/output.txt",
			},
			OutputTruncated: true,
			Error:           "probe changed /tmp/repo/internal/review/runner.go and /tmp/probe-workdir/runtime/home/output.txt",
			Commands: []ReviewProbeCommandSummary{
				{
					Command:         "cat /tmp/repo/internal/review/runner.go",
					Args:            []string{"/tmp/repo/internal/review/runner.go", "/tmp/probe-workdir/runtime/home/output.txt"},
					WorkDir:         "/tmp/probe-workdir/runtime/home",
					Status:          ReviewProbeFailed,
					ExitCode:        12,
					OutputTruncated: true,
					Error:           "failed in /tmp/probe-workdir/runtime/home/output.txt",
					DurationMs:      345,
				},
			},
		},
	}
	original := cloneReviewProbeSummariesForRedactionTest(rawSummaries)
	redactor := reviewRunnerPromptRedactor{repoRoot: normalizeReviewRunnerPromptPath("/tmp/repo")}
	redactor.addReplacement("/tmp/repo", reviewEvidenceRepoRootPathDisplay)
	redactor.addReplacement("/tmp/probe-workdir", reviewRunnerPromptProbeWorkDirDisplay)

	got := redactReviewProbeSummaries(rawSummaries, redactor)

	if !reflect.DeepEqual(rawSummaries, original) {
		t.Fatalf("redactReviewProbeSummaries() mutated input:\ngot  %#v\nwant %#v", rawSummaries, original)
	}
	if got[0].ProbeID != rawSummaries[0].ProbeID {
		t.Fatalf("ProbeID = %q, want %q", got[0].ProbeID, rawSummaries[0].ProbeID)
	}
	if got[0].Mode != rawSummaries[0].Mode {
		t.Fatalf("Mode = %q, want %q", got[0].Mode, rawSummaries[0].Mode)
	}
	if got[0].Status != rawSummaries[0].Status {
		t.Fatalf("Status = %q, want %q", got[0].Status, rawSummaries[0].Status)
	}
	if got[0].MutatedWorktree != rawSummaries[0].MutatedWorktree {
		t.Fatalf("MutatedWorktree = %v, want %v", got[0].MutatedWorktree, rawSummaries[0].MutatedWorktree)
	}
	if got[0].OutputTruncated != rawSummaries[0].OutputTruncated {
		t.Fatalf("OutputTruncated = %v, want %v", got[0].OutputTruncated, rawSummaries[0].OutputTruncated)
	}
	if got[0].Commands[0].Status != rawSummaries[0].Commands[0].Status {
		t.Fatalf("command Status = %q, want %q", got[0].Commands[0].Status, rawSummaries[0].Commands[0].Status)
	}
	if got[0].Commands[0].ExitCode != rawSummaries[0].Commands[0].ExitCode {
		t.Fatalf("command ExitCode = %d, want %d", got[0].Commands[0].ExitCode, rawSummaries[0].Commands[0].ExitCode)
	}
	if got[0].Commands[0].OutputTruncated != rawSummaries[0].Commands[0].OutputTruncated {
		t.Fatalf("command OutputTruncated = %v, want %v", got[0].Commands[0].OutputTruncated, rawSummaries[0].Commands[0].OutputTruncated)
	}
	if got[0].Commands[0].DurationMs != rawSummaries[0].Commands[0].DurationMs {
		t.Fatalf("command DurationMs = %d, want %d", got[0].Commands[0].DurationMs, rawSummaries[0].Commands[0].DurationMs)
	}

	wantMutatedFiles := []string{
		"internal/review/runner.go",
		"<probe_workdir>/runtime/home/output.txt",
	}
	if !reflect.DeepEqual(got[0].MutatedFiles, wantMutatedFiles) {
		t.Fatalf("MutatedFiles = %#v, want %#v", got[0].MutatedFiles, wantMutatedFiles)
	}
	if got[0].Error != "probe changed <repo_root>/internal/review/runner.go and <probe_workdir>/runtime/home/output.txt" {
		t.Fatalf("Error = %q, want redacted text", got[0].Error)
	}
	if got[0].Commands[0].Command != "cat <repo_root>/internal/review/runner.go" {
		t.Fatalf("command Command = %q, want redacted command", got[0].Commands[0].Command)
	}
	wantArgs := []string{"<repo_root>/internal/review/runner.go", "<probe_workdir>/runtime/home/output.txt"}
	if !reflect.DeepEqual(got[0].Commands[0].Args, wantArgs) {
		t.Fatalf("command Args = %#v, want %#v", got[0].Commands[0].Args, wantArgs)
	}
	if got[0].Commands[0].WorkDir != "<probe_workdir>/runtime/home" {
		t.Fatalf("command WorkDir = %q, want redacted workdir", got[0].Commands[0].WorkDir)
	}
	if got[0].Commands[0].Error != "failed in <probe_workdir>/runtime/home/output.txt" {
		t.Fatalf("command Error = %q, want redacted error", got[0].Commands[0].Error)
	}

	got[0].MutatedFiles[0] = "changed"
	got[0].Commands[0].Args[0] = "changed"
	if !reflect.DeepEqual(rawSummaries, original) {
		t.Fatalf("mutating redacted output changed input:\ngot  %#v\nwant %#v", rawSummaries, original)
	}
}

func TestRedactReviewProbeSummariesPreservesEmptySliceBehavior(t *testing.T) {
	var redactor reviewRunnerPromptRedactor

	gotSummaries := redactReviewProbeSummaries(nil, redactor)
	if gotSummaries == nil || len(gotSummaries) != 0 {
		t.Fatalf("redactReviewProbeSummaries(nil) = %#v, want non-nil empty slice", gotSummaries)
	}
	gotSummaries = redactReviewProbeSummaries([]ReviewProbeSummary{}, redactor)
	if gotSummaries == nil || len(gotSummaries) != 0 {
		t.Fatalf("redactReviewProbeSummaries(empty) = %#v, want non-nil empty slice", gotSummaries)
	}

	gotCommands := redactReviewProbeCommandSummaries(nil, redactor)
	if gotCommands == nil || len(gotCommands) != 0 {
		t.Fatalf("redactReviewProbeCommandSummaries(nil) = %#v, want non-nil empty slice", gotCommands)
	}
	gotCommands = redactReviewProbeCommandSummaries([]ReviewProbeCommandSummary{}, redactor)
	if gotCommands == nil || len(gotCommands) != 0 {
		t.Fatalf("redactReviewProbeCommandSummaries(empty) = %#v, want non-nil empty slice", gotCommands)
	}
}

func cloneReviewProbeSummariesForRedactionTest(summaries []ReviewProbeSummary) []ReviewProbeSummary {
	cloned := make([]ReviewProbeSummary, len(summaries))
	for i, summary := range summaries {
		cloned[i] = summary
		cloned[i].MutatedFiles = append([]string(nil), summary.MutatedFiles...)
		cloned[i].Commands = make([]ReviewProbeCommandSummary, len(summary.Commands))
		for j, command := range summary.Commands {
			cloned[i].Commands[j] = command
			cloned[i].Commands[j].Args = append([]string(nil), command.Args...)
		}
	}
	return cloned
}
