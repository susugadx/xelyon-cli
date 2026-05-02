package review

import (
	"context"
	"strings"
	"testing"
)

func TestProbeRunner_ScratchOnly_BlockedCasesAreNotExecuted(t *testing.T) {
	tests := []struct {
		name          string
		request       ReviewProbeRequest
		errorContains string
	}{
		{
			name: "blocked absolute file path",
			request: ReviewProbeRequest{
				ID:   "scratch-blocked-abs-file",
				Mode: ReviewProbeScratchOnly,
				Files: []ReviewProbeFile{{
					Path:    "/tmp/escape.py",
					Content: "print('x')\n",
				}},
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"x"}}},
			},
			errorContains: "must be relative",
		},
		{
			name: "blocked file path escape",
			request: ReviewProbeRequest{
				ID:   "scratch-blocked-escape-file",
				Mode: ReviewProbeScratchOnly,
				Files: []ReviewProbeFile{{
					Path:    "../escape.py",
					Content: "print('x')\n",
				}},
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"x"}}},
			},
			errorContains: "escapes scratch directory",
		},
		{
			name: "blocked duplicate files",
			request: ReviewProbeRequest{
				ID:   "scratch-blocked-duplicate-files",
				Mode: ReviewProbeScratchOnly,
				Files: []ReviewProbeFile{
					{Path: "check.py", Content: "print('a')\n"},
					{Path: "check.py", Content: "print('b')\n"},
				},
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"x"}}},
			},
			errorContains: "duplicate scratch file path",
		},
		{
			name: "blocked workdir escape",
			request: ReviewProbeRequest{
				ID:       "scratch-blocked-workdir-escape",
				Mode:     ReviewProbeScratchOnly,
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"x"}, WorkDir: "../outside"}},
			},
			errorContains: "escapes scratch directory",
		},
		{
			name: "blocked command",
			request: ReviewProbeRequest{
				ID:       "scratch-blocked-command",
				Mode:     ReviewProbeScratchOnly,
				Commands: []ReviewProbeCommand{{Command: "sh", Args: []string{"-c", "echo x"}}},
			},
			errorContains: "is not allowed in scratch_only",
		},
		{
			name: "blocked python -c",
			request: ReviewProbeRequest{
				ID:       "scratch-blocked-python-c",
				Mode:     ReviewProbeScratchOnly,
				Commands: []ReviewProbeCommand{{Command: "python3", Args: []string{"-c", "print('x')"}}},
			},
			errorContains: "python3 argument -c",
		},
		{
			name: "blocked go run outside",
			request: ReviewProbeRequest{
				ID:       "scratch-blocked-go-outside",
				Mode:     ReviewProbeScratchOnly,
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"run", "/etc/x.go"}}},
			},
			errorContains: "outside scratch directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, runner := newScratchOnlyProbeRunner(t)

			result, err := runner.Run(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if result.Status != ReviewProbeBlocked {
				t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeBlocked, result.Error)
			}
			if len(result.CommandResults) != 0 {
				t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
			}
			if !strings.Contains(result.Error, tt.errorContains) {
				t.Fatalf("Error = %q, want to contain %q", result.Error, tt.errorContains)
			}
		})
	}
}
