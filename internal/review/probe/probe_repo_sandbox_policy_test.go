package probe

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestProbeRunner_RepoSandbox_BlockedCasesAreNotExecuted(t *testing.T) {
	tests := []probeModeBlockedCase{
		{
			name: "blocked absolute generated file path",
			request: ReviewProbeRequest{
				ID: "repo-sandbox-blocked-abs-file",
				Files: []ReviewProbeFile{{
					Path:    "/tmp/escape.py",
					Content: "print('x')\n",
				}},
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"keep.txt"}}},
			},
			errorContains: "must be relative",
		},
		{
			name: "blocked existing generated file overwrite",
			request: ReviewProbeRequest{
				ID: "repo-sandbox-blocked-overwrite",
				Files: []ReviewProbeFile{{
					Path:    "keep.txt",
					Content: "new\n",
				}},
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"keep.txt"}}},
			},
			errorContains: "would overwrite existing file",
		},
		{
			name: "blocked max commands",
			request: func() ReviewProbeRequest {
				commands := make([]ReviewProbeCommand, 0, defaultRepoSandboxMaxCommands+1)
				for i := 0; i < defaultRepoSandboxMaxCommands+1; i++ {
					commands = append(commands, ReviewProbeCommand{Command: "cat", Args: []string{"keep.txt"}})
				}
				return ReviewProbeRequest{
					ID:       "repo-sandbox-blocked-max-commands",
					Commands: commands,
				}
			}(),
			errorContains: "allows at most",
		},
		{
			name: "blocked command",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-command",
				Commands: []ReviewProbeCommand{{Command: "sh", Args: []string{"-c", "echo x"}}},
			},
			errorContains: "is not allowed in repo_sandbox",
		},
		{
			name: "blocked command path",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-command-path",
				Commands: []ReviewProbeCommand{{Command: "./cat", Args: []string{"keep.txt"}}},
			},
			errorContains: "command path is not allowed in repo_sandbox",
		},
		{
			name: "blocked workdir escape",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-workdir-escape",
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"keep.txt"}, WorkDir: "../outside"}},
			},
			errorContains: "escapes sandbox worktree",
		},
		{
			name: "blocked cat outside",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-cat-outside",
				Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"../keep.txt"}}},
			},
			errorContains: "outside sandbox worktree",
		},
		{
			name: "blocked python -c",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-python-c",
				Commands: []ReviewProbeCommand{{Command: "python3", Args: []string{"-c", "print('x')"}}},
			},
			errorContains: "python3 argument -c",
		},
		{
			name: "blocked go get",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-get",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"get", "example.com/nope"}}},
			},
			errorContains: "go get is not allowed in repo_sandbox",
		},
		{
			name: "blocked go install",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-install",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"install", "example.com/nope"}}},
			},
			errorContains: "go install is not allowed in repo_sandbox",
		},
		{
			name: "blocked go generate",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-generate",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"generate", "./..."}}},
			},
			errorContains: "go generate is not allowed in repo_sandbox",
		},
		{
			name: "blocked go env write",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-env-write",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"env", "-w", "GOPROXY=direct"}}},
			},
			errorContains: "go env -w is not allowed in repo_sandbox",
		},
		{
			name: "blocked go test exec attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-exec-attached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "-exec=/bin/echo", "./probe"}}},
			},
			errorContains: "go argument -exec",
		},
		{
			name: "blocked go test exec double dash attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-exec-double-dash-attached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "--exec=/bin/echo", "./probe"}}},
			},
			errorContains: "go argument --exec",
		},
		{
			name: "blocked go test exec double dash detached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-exec-double-dash-detached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "--exec", "/bin/echo", "./probe"}}},
			},
			errorContains: "go argument --exec",
		},
		{
			name: "blocked go test exec detached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-exec-detached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "-exec", "/bin/echo", "./probe"}}},
			},
			errorContains: "go argument -exec",
		},
		{
			name: "blocked go test toolexec attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-toolexec-attached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "-toolexec=/bin/echo", "./probe"}}},
			},
			errorContains: "go argument -toolexec",
		},
		{
			name: "blocked go test toolexec double dash attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-toolexec-double-dash-attached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "--toolexec=/bin/echo", "./probe"}}},
			},
			errorContains: "go argument --toolexec",
		},
		{
			name: "blocked go test toolexec double dash detached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-toolexec-double-dash-detached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "--toolexec", "/bin/echo", "./probe"}}},
			},
			errorContains: "go argument --toolexec",
		},
		{
			name: "blocked go test toolexec detached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-toolexec-detached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "-toolexec", "/bin/echo", "./probe"}}},
			},
			errorContains: "go argument -toolexec",
		},
		{
			name: "blocked go vet vettool attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-vet-vettool-attached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"vet", "-vettool=/bin/echo", "./probe"}}},
			},
			errorContains: "go argument -vettool",
		},
		{
			name: "blocked go vet vettool double dash attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-vet-vettool-double-dash-attached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"vet", "--vettool=/bin/echo", "./probe"}}},
			},
			errorContains: "go argument --vettool",
		},
		{
			name: "blocked go vet vettool double dash detached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-vet-vettool-double-dash-detached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"vet", "--vettool", "/bin/echo", "./probe"}}},
			},
			errorContains: "go argument --vettool",
		},
		{
			name: "blocked go vet vettool detached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-vet-vettool-detached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"vet", "-vettool", "/bin/echo", "./probe"}}},
			},
			errorContains: "go argument -vettool",
		},
		{
			name: "blocked go run exec double dash attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-run-exec-double-dash-attached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"run", "--exec=/bin/echo", "./probe/main.go"}}},
			},
			errorContains: "go argument --exec",
		},
		{
			name: "blocked go run exec double dash detached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-run-exec-double-dash-detached",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"run", "--exec", "/bin/echo", "./probe/main.go"}}},
			},
			errorContains: "go argument --exec",
		},
		{
			name: "blocked go run outside file",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-run-outside",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"run", "../outside.go"}}},
			},
			errorContains: "outside sandbox worktree",
		},
		{
			name: "blocked go build output escape",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-build-output-escape",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"build", "-o", "../probe-bin", "./probe"}}},
			},
			errorContains: "outside sandbox worktree",
		},
		{
			name: "blocked go test profile escape attached",
			request: ReviewProbeRequest{
				ID:       "repo-sandbox-blocked-go-test-profile-escape",
				Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"test", "-coverprofile=../cover.out", "./probe"}}},
			},
			errorContains: "outside sandbox worktree",
		},
	}

	runProbeModeBlockedCases(t, domain.ReviewProbeRepoSandbox, tests)
}

func TestProbeRunner_RepoSandbox_BlocksCommandPathUnderSymlinkEscape(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	if err := os.Symlink(t.TempDir(), filepath.Join(repo, "outside-link")); err != nil {
		t.Skipf("symlink is not supported: %v", err)
	}
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:       "repo-sandbox-blocked-command-symlink-parent",
		Mode:     domain.ReviewProbeRepoSandbox,
		Commands: []ReviewProbeCommand{{Command: "go", Args: []string{"build", "-o", "outside-link/probe-bin", "./probe"}}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, domain.ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "escapes sandbox worktree") {
		t.Fatalf("Error = %q, want symlink escape block", result.Error)
	}
}
