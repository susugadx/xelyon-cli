package review

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func TestProbeRunner_HostReadOnlyBlockedOutsidePathIsNotExecuted(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-blocked-outside-path",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "cat",
				Args:    []string{"/etc/hosts"},
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
	if !strings.Contains(result.Error, "outside repository root") {
		t.Fatalf("Error = %q, want to contain %q", result.Error, "outside repository root")
	}
}
