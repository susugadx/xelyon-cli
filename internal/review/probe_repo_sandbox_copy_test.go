package review

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyRepoToSandboxWorktree_CopiesCurrentFilesystemState(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "untracked.txt"), "untracked\n")
	worktree := filepath.Join(t.TempDir(), "worktree")

	_, err := copyRepoToSandboxWorktree(repo, worktree, defaultRepoSandboxCopyLimits())
	if err != nil {
		t.Fatalf("copyRepoToSandboxWorktree() error = %v", err)
	}

	if got := readTestFile(t, filepath.Join(worktree, "keep.txt")); got != "keep\n" {
		t.Fatalf("copied keep.txt = %q, want keep", got)
	}
	if got := readTestFile(t, filepath.Join(worktree, "untracked.txt")); got != "untracked\n" {
		t.Fatalf("copied untracked.txt = %q, want untracked", got)
	}
	if _, err := os.Lstat(filepath.Join(worktree, ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf(".git should not be copied, stat error = %v", err)
	}
}

func TestCopyRepoToSandboxWorktree_CopiesSymlinkAsSymlink(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	linkPath := filepath.Join(repo, "keep-link")
	if err := os.Symlink("keep.txt", linkPath); err != nil {
		t.Skipf("symlink is not supported: %v", err)
	}
	worktree := filepath.Join(t.TempDir(), "worktree")

	_, err := copyRepoToSandboxWorktree(repo, worktree, defaultRepoSandboxCopyLimits())
	if err != nil {
		t.Fatalf("copyRepoToSandboxWorktree() error = %v", err)
	}

	target, err := os.Readlink(filepath.Join(worktree, "keep-link"))
	if err != nil {
		t.Fatalf("Readlink(copied symlink) error = %v", err)
	}
	if target != "keep.txt" {
		t.Fatalf("copied symlink target = %q, want keep.txt", target)
	}
}

func TestCopyRepoToSandboxWorktree_BlocksWhenLimitsExceeded(t *testing.T) {
	tests := []struct {
		name          string
		limits        repoSandboxCopyLimits
		errorContains string
	}{
		{
			name: "file count",
			limits: repoSandboxCopyLimits{
				maxFiles:     1,
				maxBytes:     defaultRepoSandboxMaxCopyBytes,
				maxFileBytes: defaultRepoSandboxMaxCopyFileBytes,
			},
			errorContains: "max file count",
		},
		{
			name: "single file bytes",
			limits: repoSandboxCopyLimits{
				maxFiles:     defaultRepoSandboxMaxCopyFiles,
				maxBytes:     defaultRepoSandboxMaxCopyBytes,
				maxFileBytes: 4,
			},
			errorContains: "max copy file bytes",
		},
		{
			name: "total bytes",
			limits: repoSandboxCopyLimits{
				maxFiles:     defaultRepoSandboxMaxCopyFiles,
				maxBytes:     4,
				maxFileBytes: defaultRepoSandboxMaxCopyFileBytes,
			},
			errorContains: "max total bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
			worktree := filepath.Join(t.TempDir(), "worktree")

			_, err := copyRepoToSandboxWorktree(repo, worktree, tt.limits)
			if err == nil {
				t.Fatal("copyRepoToSandboxWorktree() error = nil")
			}
			if !errors.Is(err, ErrHostReadOnlyBlocked) {
				t.Fatalf("copyRepoToSandboxWorktree() error = %v, want ErrHostReadOnlyBlocked", err)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("error = %q, want to contain %q", err, tt.errorContains)
			}
		})
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}
