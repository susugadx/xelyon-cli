package review

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAndBuildRepoSandboxFiles_AllowsNewRelativeFiles(t *testing.T) {
	worktree := t.TempDir()

	files, err := validateAndBuildRepoSandboxFiles(worktree, []ReviewProbeFile{
		{Path: "check.py", Content: "print('ok')\n"},
		{Path: "tests/check_test.go", Content: "package tests\n"},
	})
	if err != nil {
		t.Fatalf("validateAndBuildRepoSandboxFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
}

func TestValidateAndBuildRepoSandboxFiles_BlockedCases(t *testing.T) {
	tests := []struct {
		name          string
		files         []ReviewProbeFile
		setup         func(t *testing.T, worktree string)
		errorContains string
	}{
		{
			name: "absolute path",
			files: []ReviewProbeFile{{
				Path:    "/tmp/escape.py",
				Content: "print('x')\n",
			}},
			errorContains: "must be relative",
		},
		{
			name: "escape path",
			files: []ReviewProbeFile{{
				Path:    "../escape.py",
				Content: "print('x')\n",
			}},
			errorContains: "escapes sandbox worktree",
		},
		{
			name: "duplicate normalized path",
			files: []ReviewProbeFile{
				{Path: "check.py", Content: "print('a')\n"},
				{Path: "./check.py", Content: "print('b')\n"},
			},
			errorContains: "duplicate repo_sandbox generated file path",
		},
		{
			name: "existing overwrite",
			setup: func(t *testing.T, worktree string) {
				writeTestFile(t, filepath.Join(worktree, "keep.txt"), "keep\n")
			},
			files: []ReviewProbeFile{{
				Path:    "keep.txt",
				Content: "new\n",
			}},
			errorContains: "would overwrite existing file",
		},
		{
			name: "symlink parent escape",
			setup: func(t *testing.T, worktree string) {
				if err := os.Symlink(t.TempDir(), filepath.Join(worktree, "outside-link")); err != nil {
					t.Skipf("symlink is not supported: %v", err)
				}
			},
			files: []ReviewProbeFile{{
				Path:    "outside-link/generated.txt",
				Content: "new\n",
			}},
			errorContains: "escapes sandbox worktree",
		},
		{
			name: "max files",
			files: func() []ReviewProbeFile {
				files := make([]ReviewProbeFile, 0, defaultRepoSandboxMaxGeneratedFiles+1)
				for i := 0; i < defaultRepoSandboxMaxGeneratedFiles+1; i++ {
					files = append(files, ReviewProbeFile{Path: fmt.Sprintf("f-%d.txt", i), Content: "ok"})
				}
				return files
			}(),
			errorContains: "allows at most",
		},
		{
			name: "single generated file bytes",
			files: []ReviewProbeFile{{
				Path:    "large.txt",
				Content: strings.Repeat("x", defaultRepoSandboxMaxGeneratedFileBytes+1),
			}},
			errorContains: "exceeds max file bytes",
		},
		{
			name: "generated total bytes",
			files: []ReviewProbeFile{
				{Path: "a.txt", Content: strings.Repeat("x", defaultRepoSandboxMaxGeneratedFileBytes)},
				{Path: "b.txt", Content: strings.Repeat("x", defaultRepoSandboxMaxGeneratedFileBytes)},
				{Path: "c.txt", Content: strings.Repeat("x", defaultRepoSandboxMaxGeneratedFileBytes)},
				{Path: "d.txt", Content: strings.Repeat("x", defaultRepoSandboxMaxGeneratedFileBytes)},
				{Path: "e.txt", Content: strings.Repeat("x", defaultRepoSandboxMaxGeneratedFileBytes)},
			},
			errorContains: "exceed max total bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktree := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, worktree)
			}

			_, err := validateAndBuildRepoSandboxFiles(worktree, tt.files)
			if err == nil {
				t.Fatal("validateAndBuildRepoSandboxFiles() error = nil")
			}
			if !errors.Is(err, ErrHostReadOnlyBlocked) {
				t.Fatalf("validateAndBuildRepoSandboxFiles() error = %v, want ErrHostReadOnlyBlocked", err)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("error = %q, want to contain %q", err, tt.errorContains)
			}
		})
	}
}
