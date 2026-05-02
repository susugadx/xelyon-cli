package review

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProbeRunner_HostReadOnlyPassed(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-pass",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"status", "--short"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != ReviewProbePassed {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, ReviewProbePassed)
	}
}

func TestProbeRunner_HostReadOnlyPassed_GitGlobalOption(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-pass-git-global-option",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"--no-optional-locks", "status", "--short"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != ReviewProbePassed {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, ReviewProbePassed)
	}
}

func TestProbeRunner_HostReadOnlyTimedOut(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-timeout",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        100 * time.Millisecond,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "TestProbeSleep"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeTimedOut {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeTimedOut, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != ReviewProbeTimedOut {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, ReviewProbeTimedOut)
	}
}

func TestProbeRunner_HostReadOnlyOutputTruncated(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-truncate",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 32,
		Commands: []ReviewProbeCommand{
			{
				Command: "cat",
				Args:    []string{"large.txt"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbePassed)
	}
	if !result.OutputTruncated {
		t.Fatal("OutputTruncated = false, want true")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !result.CommandResults[0].OutputTruncated {
		t.Fatal("CommandResults[0].OutputTruncated = false, want true")
	}
	if len(result.CommandResults[0].Output) > 32 {
		t.Fatalf("len(CommandResults[0].Output) = %d, want <= 32", len(result.CommandResults[0].Output))
	}
}

func TestProbeRunner_HostReadOnlyMutationDetected(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error)
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("probe/probe_generated.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain probe/probe_generated.txt", result.MutatedFiles)
	}
}

func TestProbeRunner_HostReadOnlyMutationDetected_DirtyWorktreeReportsOnlyDelta(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	writeTestFile(t, filepath.Join(repo, "keep.txt"), "pre-existing-dirty\n")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation-dirty",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error)
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("probe/probe_generated.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain probe/probe_generated.txt", result.MutatedFiles)
	}
	if containsString(result.MutatedFiles, filepath.ToSlash("keep.txt")) {
		t.Fatalf("MutatedFiles = %#v, keep.txt should not be included", result.MutatedFiles)
	}
}

func TestProbeRunner_HostReadOnlyMutationDetected_DirtyExistingPathChanged(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	writeTestFile(t, filepath.Join(repo, "keep.txt"), "pre-existing-dirty\n")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation-dirty-existing-path",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutateDirtyExistingPath$"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error)
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("keep.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain keep.txt", result.MutatedFiles)
	}
}

func TestProbeRunner_HostReadOnlyBlockedCommand(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	keepFile := filepath.Join(repo, "keep.txt")
	if _, err := os.Stat(keepFile); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", keepFile, err)
	}

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-blocked",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "rm",
				Args:    []string{"-f", "keep.txt"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "blocked command") {
		t.Fatalf("Error = %q, want to contain %q", result.Error, "blocked command")
	}
	if _, err := os.Stat(keepFile); err != nil {
		t.Fatalf("blocked command should not remove file, stat error = %v", err)
	}
}

func TestProbeRunner_HostReadOnlyBlockedCommandPath(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-blocked-command-path",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "./git",
				Args:    []string{"status", "--short"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "command path is not allowed") {
		t.Fatalf("Error = %q, want to contain %q", result.Error, "command path is not allowed")
	}
}

func TestProbeRunner_HostReadOnlyBlockedArgIsNotExecuted(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-blocked-arg",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "git",
				Args:    []string{"diff", "--ext-diff"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "blocked command") {
		t.Fatalf("Error = %q, want to contain %q", result.Error, "blocked command")
	}
}

func TestProbeRunner_HostReadOnlyArgsDoNotUseShell(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)
	injectedPath := filepath.Join(repo, "shell_pwned")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-no-shell",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "git",
				Args:    []string{"status", "--short; touch shell_pwned"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeFailed {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeFailed)
	}
	if _, err := os.Stat(injectedPath); !os.IsNotExist(err) {
		t.Fatalf("shell-like argument should not create %s, stat error = %v", injectedPath, err)
	}
}

func TestProbeRunner_UnsupportedMode(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:   "probe-unsupported",
		Mode: ReviewProbeScratchOnly,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want unsupported mode error")
	}
	if !errors.Is(err, ErrUnsupportedReviewProbeMode) {
		t.Fatalf("Run() error = %v, want ErrUnsupportedReviewProbeMode", err)
	}
	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
}

func newProbeTestRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)

	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	writeTestFile(t, filepath.Join(repo, "go.mod"), "module example.com/reviewprobe\n\ngo 1.26.0\n")
	writeTestFile(t, filepath.Join(repo, "probe", "probe.go"), "package probe\n\nfunc Add(a, b int) int { return a + b }\n")
	writeTestFile(t, filepath.Join(repo, "probe", "probe_test.go"), `package probe

import (
	"os"
	"testing"
	"time"
)

func TestProbeSleep(t *testing.T) {
	time.Sleep(1 * time.Second)
}

func TestProbeMutate(t *testing.T) {
	if err := os.WriteFile("probe_generated.txt", []byte("generated"), 0o644); err != nil {
		t.Fatalf("write probe_generated.txt: %v", err)
	}
}

func TestProbeMutateDirtyExistingPath(t *testing.T) {
	if err := os.WriteFile("../keep.txt", []byte("probe-updated-existing-dirty-path\n"), 0o644); err != nil {
		t.Fatalf("write ../keep.txt: %v", err)
	}
}
`)
	writeTestFile(t, filepath.Join(repo, "large.txt"), strings.Repeat("large-output-line\n", 128))
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "keep\n")

	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")

	return repo
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if filepath.ToSlash(item) == filepath.ToSlash(target) {
			return true
		}
	}
	return false
}
