package review

import (
	"context"
	"strconv"
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
			name: "blocked max files",
			request: func() ReviewProbeRequest {
				files := make([]ReviewProbeFile, 0, defaultScratchOnlyMaxFiles+1)
				for i := 0; i < defaultScratchOnlyMaxFiles+1; i++ {
					files = append(files, ReviewProbeFile{
						Path:    "f-" + strconv.Itoa(i) + ".txt",
						Content: "ok",
					})
				}
				return ReviewProbeRequest{
					ID:       "scratch-blocked-max-files",
					Mode:     ReviewProbeScratchOnly,
					Files:    files,
					Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"check.txt"}}},
				}
			}(),
			errorContains: "allows at most",
		},
		{
			name: "blocked single file bytes",
			request: ReviewProbeRequest{
				ID:   "scratch-blocked-single-file-bytes",
				Mode: ReviewProbeScratchOnly,
				Files: []ReviewProbeFile{{
					Path:    "large.py",
					Content: strings.Repeat("x", defaultScratchOnlyMaxFileBytes+1),
				}},
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"check.txt"}}},
			},
			errorContains: "exceeds max file bytes",
		},
		{
			name: "blocked total file bytes",
			request: func() ReviewProbeRequest {
				content := strings.Repeat("x", defaultScratchOnlyMaxFileBytes)
				return ReviewProbeRequest{
					ID:   "scratch-blocked-total-file-bytes",
					Mode: ReviewProbeScratchOnly,
					Files: []ReviewProbeFile{
						{Path: "a.txt", Content: content},
						{Path: "b.txt", Content: content},
						{Path: "c.txt", Content: content},
						{Path: "d.txt", Content: content},
						{Path: "e.txt", Content: content},
					},
					Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"check.txt"}}},
				}
			}(),
			errorContains: "exceed max total bytes",
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
		{
			name: "blocked max commands",
			request: func() ReviewProbeRequest {
				commands := make([]ReviewProbeCommand, 0, defaultScratchOnlyMaxCommands+1)
				for i := 0; i < defaultScratchOnlyMaxCommands+1; i++ {
					commands = append(commands, ReviewProbeCommand{
						Command: "cat",
						Args:    []string{"check.txt"},
					})
				}
				return ReviewProbeRequest{
					ID:       "scratch-blocked-max-commands",
					Mode:     ReviewProbeScratchOnly,
					Commands: commands,
				}
			}(),
			errorContains: "allows at most",
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
