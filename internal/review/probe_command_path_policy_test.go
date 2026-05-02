package review

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHostReadOnlyCommandPathPolicy_AllowsRepoPaths(t *testing.T) {
	repoRoot := t.TempDir()
	workDir := filepath.Join(repoRoot, "internal")

	tests := []struct {
		name    string
		command string
		args    []string
		workDir string
	}{
		{
			name:    "cat file",
			command: "cat",
			args:    []string{"file.txt"},
		},
		{
			name:    "ls empty args",
			command: "ls",
			args:    nil,
		},
		{
			name:    "ls internal",
			command: "ls",
			args:    []string{"internal"},
		},
		{
			name:    "find empty args treated as current dir",
			command: "find",
			args:    nil,
		},
		{
			name:    "find current dir",
			command: "find",
			args:    []string{".", "-name", "*.go"},
		},
		{
			name:    "find internal dir",
			command: "find",
			args:    []string{"internal", "-name", "*.go"},
		},
		{
			name:    "find with option separator internal",
			command: "find",
			args:    []string{"--", "internal", "-name", "*.go"},
		},
		{
			name:    "rg pattern only",
			command: "rg",
			args:    []string{"pattern"},
		},
		{
			name:    "rg with path after separator",
			command: "rg",
			args:    []string{"pattern", "--", "internal"},
		},
		{
			name:    "grep with path after separator",
			command: "grep",
			args:    []string{"pattern", "--", "internal/file.go"},
		},
		{
			name:    "git diff with path after separator",
			command: "git",
			args:    []string{"diff", "--", "internal/review"},
		},
		{
			name:    "git global option status",
			command: "git",
			args:    []string{"--no-optional-locks", "status", "--short"},
		},
		{
			name:    "go path policy excluded",
			command: "go",
			args:    []string{"test", "/etc"},
		},
		{
			name:    "npm path policy excluded",
			command: "npm",
			args:    []string{"test"},
		},
		{
			name:    "cargo path policy excluded",
			command: "cargo",
			args:    []string{"test"},
		},
		{
			name:    "sed path policy excluded",
			command: "sed",
			args:    []string{"-n", "p", "/etc/hosts"},
		},
		{
			name:    "cat relative to workdir",
			command: "cat",
			args:    []string{"file.txt"},
			workDir: workDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd := repoRoot
			if tt.workDir != "" {
				wd = tt.workDir
			}

			err := validateHostReadOnlyCommandPathPolicy(repoRoot, wd, tt.command, tt.args)
			if err != nil {
				t.Fatalf("validateHostReadOnlyCommandPathPolicy(%q, %q, %#v) error = %v", tt.command, wd, tt.args, err)
			}
		})
	}
}

func TestHostReadOnlyCommandPathPolicy_BlocksOutsidePaths(t *testing.T) {
	repoRoot := t.TempDir()

	tests := []struct {
		name    string
		command string
		args    []string
		workDir string
	}{
		{
			name:    "cat absolute outside",
			command: "cat",
			args:    []string{"/etc/hosts"},
		},
		{
			name:    "cat parent outside",
			command: "cat",
			args:    []string{"../outside.txt"},
			workDir: repoRoot,
		},
		{
			name:    "ls absolute outside",
			command: "ls",
			args:    []string{"/etc"},
		},
		{
			name:    "find absolute outside",
			command: "find",
			args:    []string{"/etc", "-name", "hosts"},
		},
		{
			name:    "find option separator absolute outside",
			command: "find",
			args:    []string{"--", "/etc", "-name", "hosts"},
		},
		{
			name:    "find parent outside",
			command: "find",
			args:    []string{"../outside", "-name", "*.go"},
			workDir: repoRoot,
		},
		{
			name:    "rg absolute outside after separator",
			command: "rg",
			args:    []string{"pattern", "--", "/etc"},
		},
		{
			name:    "grep absolute outside after separator",
			command: "grep",
			args:    []string{"pattern", "--", "/etc/passwd"},
		},
		{
			name:    "git diff absolute outside after separator",
			command: "git",
			args:    []string{"diff", "--", "/etc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd := repoRoot
			if tt.workDir != "" {
				wd = tt.workDir
			}

			err := validateHostReadOnlyCommandPathPolicy(repoRoot, wd, tt.command, tt.args)
			if err == nil {
				t.Fatalf("validateHostReadOnlyCommandPathPolicy(%q, %q, %#v) error = nil", tt.command, wd, tt.args)
			}
			if !errors.Is(err, ErrHostReadOnlyOutsideRepoPath) {
				t.Fatalf("validateHostReadOnlyCommandPathPolicy(%q, %q, %#v) error = %v, want ErrHostReadOnlyOutsideRepoPath", tt.command, wd, tt.args, err)
			}
		})
	}
}

func TestHostReadOnlyCommandPathPolicy_BlocksSymlinkEscape(t *testing.T) {
	repoRoot := t.TempDir()
	linkPath := filepath.Join(repoRoot, "hosts_link")
	if err := os.Symlink("/etc/hosts", linkPath); err != nil {
		t.Skipf("symlink is not supported: %v", err)
	}

	err := validateHostReadOnlyCommandPathPolicy(repoRoot, repoRoot, "cat", []string{"hosts_link"})
	if err == nil {
		t.Fatal("validateHostReadOnlyCommandPathPolicy(cat hosts_link) error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyOutsideRepoPath) {
		t.Fatalf("validateHostReadOnlyCommandPathPolicy(cat hosts_link) error = %v, want ErrHostReadOnlyOutsideRepoPath", err)
	}
}
