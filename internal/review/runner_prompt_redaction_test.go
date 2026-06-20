package review

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func TestReviewRunnerPromptRedactorRedactsWindowsNativePathVariants(t *testing.T) {
	var redactor reviewRunnerPromptRedactor
	redactor.addReplacement(`C:/repo`, reviewevidence.RepoRootPathDisplay)
	redactor.addReplacement(`C:/Users/me/AppData/Local/Temp/xelyon-review-sandbox-123`, reviewRunnerPromptProbeWorkDirDisplay)

	input := strings.Join([]string{
		`repo native C:\repo\internal\review\runner.go`,
		`repo slash C:/repo/internal/review/runner.go`,
		`probe native C:\Users\me\AppData\Local\Temp\xelyon-review-sandbox-123\runtime\home\out.txt`,
		`probe slash C:/Users/me/AppData/Local/Temp/xelyon-review-sandbox-123/runtime/home/out.txt`,
	}, "\n")

	got := redactor.redactText(input)
	for _, leaked := range []string{
		`C:\repo`,
		`C:/repo`,
		`C:\Users\me\AppData\Local\Temp\xelyon-review-sandbox-123`,
		`C:/Users/me/AppData/Local/Temp/xelyon-review-sandbox-123`,
	} {
		if strings.Contains(got, leaked) {
			t.Fatalf("redactText() leaked %q in:\n%s", leaked, got)
		}
	}
	for _, want := range []string{
		`<repo_root>\internal\review\runner.go`,
		`<repo_root>/internal/review/runner.go`,
		`<probe_workdir>\runtime\home\out.txt`,
		`<probe_workdir>/runtime/home/out.txt`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("redactText() missing %q in:\n%s", want, got)
		}
	}

	gotPath := redactor.redactPath(`C:\repo\internal\review\runner.go`)
	if gotPath != `<repo_root>/internal/review/runner.go` {
		t.Fatalf("redactPath() = %q, want redacted slash display path", gotPath)
	}
}

func TestReviewRunnerPromptIsolatedProbeRootAcceptsWindowsNativePath(t *testing.T) {
	root, ok := reviewRunnerPromptIsolatedProbeRoot(`C:\Users\me\AppData\Local\Temp\xelyon-review-sandbox-123\worktree`)
	if !ok {
		t.Fatal("reviewRunnerPromptIsolatedProbeRoot() ok = false, want true")
	}
	if got, want := reviewRunnerPromptSlashPath(root), `C:/Users/me/AppData/Local/Temp/xelyon-review-sandbox-123`; got != want {
		t.Fatalf("reviewRunnerPromptIsolatedProbeRoot() = %q, want %q", got, want)
	}
}

func TestReviewRunnerPromptRedactorRedactsIsolatedRootFromResultWithoutCommandResults(t *testing.T) {
	repoRoot := t.TempDir()
	sandboxRoot := filepath.Join(t.TempDir(), reviewprobe.ReviewProbeSandboxTempPrefix+"setup-failed")
	sandboxFile := filepath.Join(sandboxRoot, "runtime/home/output.txt")
	scratchRoot := filepath.Join(t.TempDir(), reviewprobe.ReviewProbeScratchTempPrefix+"cleanup-failed")
	scratchFile := filepath.Join(scratchRoot, "tmp/mutated.txt")
	result := reviewprobe.ReviewProbeResult{
		ID:           "probe-1",
		Mode:         domain.ReviewProbeRepoSandbox,
		Status:       domain.ReviewProbeBlocked,
		MutatedFiles: []string{scratchFile},
		Error:        "failed to prepare repo_sandbox at " + sandboxFile + ": copy failed",
	}

	redactor := newReviewRunnerPromptRedactor(newRunnerEvidenceBundleForTest(repoRoot), []reviewprobe.ReviewProbeResult{result})
	contexts := reviewmodelinput.BuildProbeResultPromptContexts([]reviewprobe.ReviewProbeResult{result}, redactor)
	if got, want := len(contexts), 1; got != want {
		t.Fatalf("contexts = %d, want %d", got, want)
	}

	got := contexts[0]
	for _, leaked := range []string{sandboxRoot, sandboxFile, scratchRoot, scratchFile} {
		if strings.Contains(got.Error, leaked) || strings.Contains(strings.Join(got.MutatedFiles, "\n"), leaked) {
			t.Fatalf("redacted prompt context leaked %q: error=%q mutated_files=%v", leaked, got.Error, got.MutatedFiles)
		}
	}
	if !strings.Contains(got.Error, "<probe_workdir>/runtime/home/output.txt") {
		t.Fatalf("Error = %q, want sandbox path redacted with suffix", got.Error)
	}
	if len(got.MutatedFiles) != 1 || !strings.Contains(got.MutatedFiles[0], "<probe_workdir_2>/tmp/mutated.txt") {
		t.Fatalf("MutatedFiles = %#v, want scratch path redacted with suffix", got.MutatedFiles)
	}
}
