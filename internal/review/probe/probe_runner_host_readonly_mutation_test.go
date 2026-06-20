package probe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestProbeRunner_HostReadOnlySandboxBlocksRepoMutation(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        probeNestedGoTestTimeout,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbeFailed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, domain.ReviewProbeFailed, result.Error, firstCommandOutput(result))
	}
	if result.MutatedWorktree {
		t.Fatal("MutatedWorktree = true, want false")
	}
	if _, err := os.Lstat(filepath.Join(repo, "probe", "probe_generated.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("probe_generated.txt should not be written to original repo, stat error = %v", err)
	}
}

func TestProbeRunner_HostReadOnlySandboxBlocksRepoMutation_DirtyWorktreeUntouched(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	writeTestFile(t, filepath.Join(repo, "keep.txt"), "pre-existing-dirty\n")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation-dirty",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        probeNestedGoTestTimeout,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbeFailed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, domain.ReviewProbeFailed, result.Error, firstCommandOutput(result))
	}
	if result.MutatedWorktree {
		t.Fatal("MutatedWorktree = true, want false")
	}
	if len(result.MutatedFiles) != 0 {
		t.Fatalf("MutatedFiles = %#v, want empty", result.MutatedFiles)
	}
	if _, err := os.Lstat(filepath.Join(repo, "probe", "probe_generated.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("probe_generated.txt should not be written to original repo, stat error = %v", err)
	}
}

func TestProbeRunner_HostReadOnlySandboxBlocksDirtyExistingPathMutation(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	writeTestFile(t, filepath.Join(repo, "keep.txt"), "pre-existing-dirty\n")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation-dirty-existing-path",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        probeNestedGoTestTimeout,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutateDirtyExistingPath$"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbeFailed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, domain.ReviewProbeFailed, result.Error, firstCommandOutput(result))
	}
	if result.MutatedWorktree {
		t.Fatal("MutatedWorktree = true, want false")
	}
	if len(result.MutatedFiles) != 0 {
		t.Fatalf("MutatedFiles = %#v, want empty", result.MutatedFiles)
	}

	content, err := os.ReadFile(filepath.Join(repo, "keep.txt"))
	if err != nil {
		t.Fatalf("ReadFile(keep.txt) error = %v", err)
	}
	if string(content) != "pre-existing-dirty\n" {
		t.Fatalf("keep.txt = %q, want pre-existing dirty content", string(content))
	}
}

func TestProbeRunner_HostReadOnlySandboxBlocksHostFileRead(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	writeTestFile(t, filepath.Join(repo, "probe", "sandbox_read_test.go"), `package probe

import (
	"os"
	"testing"
)

func TestCannotReadHostEtcPasswd(t *testing.T) {
	if _, err := os.ReadFile("/etc/passwd"); err == nil {
		t.Fatal("read /etc/passwd unexpectedly succeeded")
	}
}
`)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-host-file-read",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        probeNestedGoTestTimeout,
		MaxOutputBytes: 2048,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "^TestCannotReadHostEtcPasswd$"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, domain.ReviewProbePassed, result.Error, firstCommandOutput(result))
	}
	if result.MutatedWorktree {
		t.Fatal("MutatedWorktree = true, want false")
	}
	if strings.Contains(firstCommandOutput(result), "read /etc/passwd unexpectedly succeeded") {
		t.Fatalf("output = %q", firstCommandOutput(result))
	}
}
