package probe

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProbeRunner_HostReadOnlyBlockedCommand(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
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
	assertHostReadOnlyBlockedWithoutExecution(t, err, result, "blocked command")
	if _, err := os.Stat(keepFile); err != nil {
		t.Fatalf("blocked command should not remove file, stat error = %v", err)
	}
}

func TestProbeRunner_HostReadOnlyBlockedCasesAreNotExecuted(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		command       string
		args          []string
		errorContains string
	}{
		{
			name:          "blocked command path",
			id:            "probe-blocked-command-path",
			command:       "./git",
			args:          []string{"status", "--short"},
			errorContains: "command path is not allowed",
		},
		{
			name:          "blocked dangerous argument",
			id:            "probe-blocked-arg",
			command:       "git",
			args:          []string{"diff", "--ext-diff"},
			errorContains: "blocked command",
		},
		{
			name:          "blocked outside path",
			id:            "probe-blocked-outside-path",
			command:       "cat",
			args:          []string{"/etc/hosts"},
			errorContains: "outside repository root",
		},
		{
			name:          "blocked find leading option",
			id:            "probe-blocked-find-leading-option",
			command:       "find",
			args:          []string{"-L", "/etc", "-name", "hosts"},
			errorContains: "find leading option -L",
		},
		{
			name:          "blocked npm prefix",
			id:            "probe-blocked-npm-prefix",
			command:       "npm",
			args:          []string{"test", "--prefix", "/tmp/project"},
			errorContains: "npm argument --prefix",
		},
		{
			name:          "blocked rg follow",
			id:            "probe-blocked-rg-follow",
			command:       "rg",
			args:          []string{"--follow", "pattern"},
			errorContains: "rg argument --follow",
		},
	}
	for _, sc := range hostReadOnlyRunnerBlockedSearchOutsideScenarios() {
		tests = append(tests, struct {
			name          string
			id            string
			command       string
			args          []string
			errorContains string
		}{
			name:          sc.name,
			id:            sc.id,
			command:       sc.command,
			args:          sc.args,
			errorContains: sc.errorContains,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
			runner := NewProbeRunner(repo)

			result, err := runner.Run(context.Background(), ReviewProbeRequest{
				ID:             tt.id,
				Mode:           ReviewProbeHostReadOnly,
				Timeout:        2 * time.Second,
				MaxOutputBytes: 1024,
				Commands: []ReviewProbeCommand{
					{
						Command: tt.command,
						Args:    tt.args,
					},
				},
			})

			assertHostReadOnlyBlockedWithoutExecution(t, err, result, tt.errorContains)
		})
	}
}

func TestHostReadOnlyExecutor_BlocksCommandResolvedInsideRepoRoot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	repoBin := filepath.Join(repo, "bin")
	safeBin := filepath.Join(t.TempDir(), "safe-bin")
	createProbeTestScriptCommand(t, repoBin, "git", "exit 0")
	createProbeTestScriptCommand(t, safeBin, "git", "exit 0")

	executor := newHostReadOnlyExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + strings.Join([]string{repoBin, safeBin}, string(filepath.ListSeparator)),
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "host-readonly-blocked-repo-bin",
		Mode: ReviewProbeHostReadOnly,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"status", "--short"}},
		},
	})

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
}

func assertHostReadOnlyBlockedWithoutExecution(t *testing.T, err error, result ReviewProbeResult, errorContains string) {
	t.Helper()

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, errorContains) {
		t.Fatalf("Error = %q, want to contain %q", result.Error, errorContains)
	}
}
