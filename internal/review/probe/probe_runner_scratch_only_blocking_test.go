package probe

import (
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestProbeRunner_ScratchOnly_BlockedCasesAreNotExecuted(t *testing.T) {
	tests := []probeModeBlockedCase{
		{
			name: "blocked absolute file path",
			request: ReviewProbeRequest{
				ID: "scratch-blocked-abs-file",
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
				ID: "scratch-blocked-escape-file",
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
				ID: "scratch-blocked-duplicate-files",
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
					Files:    files,
					Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"check.txt"}}},
				}
			}(),
			errorContains: "allows at most",
		},
		{
			name: "blocked single file bytes",
			request: ReviewProbeRequest{
				ID: "scratch-blocked-single-file-bytes",
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
					ID: "scratch-blocked-total-file-bytes",
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
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"x"}, WorkDir: "../outside"}},
			},
			errorContains: "escapes scratch directory",
		},
		{
			name: "blocked command",
			request: ReviewProbeRequest{
				ID:       "scratch-blocked-command",
				Commands: []ReviewProbeCommand{{Command: "sh", Args: []string{"-c", "echo x"}}},
			},
			errorContains: "is not allowed in scratch_only",
		},
		{
			name: "blocked python -c",
			request: ReviewProbeRequest{
				ID:       "scratch-blocked-python-c",
				Commands: []ReviewProbeCommand{{Command: "python3", Args: []string{"-c", "print('x')"}}},
			},
			errorContains: "python3 argument -c",
		},
		{
			name: "blocked go run outside",
			request: ReviewProbeRequest{
				ID:       "scratch-blocked-go-outside",
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
					Commands: commands,
				}
			}(),
			errorContains: "allows at most",
		},
	}

	runProbeModeBlockedCases(t, domain.ReviewProbeScratchOnly, tests)
}
