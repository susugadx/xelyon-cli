package review

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProbeRunner_ScratchOnly_FileAndCommandOutput(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "scratch-cat",
		Mode:           ReviewProbeScratchOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Files: []ReviewProbeFile{{
			Path:    "check.txt",
			Content: "ok\n",
		}},
		Commands: []ReviewProbeCommand{{
			Command: "cat",
			Args:    []string{"check.txt"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !strings.Contains(result.CommandResults[0].Output, "ok") {
		t.Fatalf("CommandResults[0].Output = %q, want to contain ok", result.CommandResults[0].Output)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
}

func TestProbeRunner_ScratchOnly_Timeout(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "scratch-timeout",
		Mode:           ReviewProbeScratchOnly,
		Timeout:        100 * time.Millisecond,
		MaxOutputBytes: 1024,
		Files: []ReviewProbeFile{{
			Path: "sleep.go",
			Content: "package main\n" +
				"import \"time\"\n" +
				"func main(){time.Sleep(2 * time.Second)}\n",
		}},
		Commands: []ReviewProbeCommand{{
			Command: "go",
			Args:    []string{"run", "sleep.go"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeTimedOut {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeTimedOut, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != ReviewProbeTimedOut {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, ReviewProbeTimedOut)
	}
}

func TestProbeRunner_ScratchOnly_OutputCap(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "scratch-output-cap",
		Mode:           ReviewProbeScratchOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 32,
		Files: []ReviewProbeFile{{
			Path:    "large.txt",
			Content: strings.Repeat("x", 1024),
		}},
		Commands: []ReviewProbeCommand{{
			Command: "cat",
			Args:    []string{"large.txt"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbePassed)
	}
	if !result.OutputTruncated {
		t.Fatal("OutputTruncated = false, want true")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !result.CommandResults[0].OutputTruncated {
		t.Fatal("CommandResults[0].OutputTruncated = false, want true")
	}
}

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
			repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
			runner := NewProbeRunner(repo)

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

func TestProbeRunner_ScratchOnly_Cleanup(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "scratch-cleanup",
		Mode:           ReviewProbeScratchOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 2048,
		Files: []ReviewProbeFile{{
			Path: "print_scratch.go",
			Content: "package main\n" +
				"import (\n" +
				"\t\"fmt\"\n" +
				"\t\"os\"\n" +
				")\n" +
				"func main(){fmt.Println(os.Getenv(\"XELYON_REVIEW_SCRATCH_DIR\"))}\n",
		}},
		Commands: []ReviewProbeCommand{{
			Command: "go",
			Args:    []string{"run", "print_scratch.go"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}

	scratchDir := strings.TrimSpace(result.CommandResults[0].Output)
	if scratchDir == "" {
		t.Fatal("scratch dir output is empty")
	}
	if _, statErr := os.Stat(scratchDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("scratch dir should be removed, stat error = %v", statErr)
	}
}

func TestProbeRunner_ScratchOnly_RepoMutationDetection(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "scratch-mutation",
		Mode:           ReviewProbeScratchOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Files: []ReviewProbeFile{{
			Path: "mutate_repo.go",
			Content: "package main\n" +
				"import (\n" +
				"\t\"os\"\n" +
				"\t\"path/filepath\"\n" +
				")\n" +
				"func main(){\n" +
				"\trepo := os.Getenv(\"XELYON_REVIEW_REPO_ROOT\")\n" +
				"\t_ = os.WriteFile(filepath.Join(repo, \"keep.txt\"), []byte(\"mutated\\n\"), 0o644)\n" +
				"}\n",
		}},
		Commands: []ReviewProbeCommand{{
			Command: "go",
			Args:    []string{"run", "mutate_repo.go"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error)
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("keep.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain keep.txt", result.MutatedFiles)
	}
}

func TestScratchOnlyExecutor_BlocksScratchDirInsideRepoAndCleansUp(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	scratchDir := filepath.Join(repo, ".xelyon-review-scratch")
	removed := make([]string, 0, 1)

	executor := newScratchOnlyExecutor(repo)
	executor.mktemp = func(dir, pattern string) (string, error) {
		if err := os.MkdirAll(scratchDir, 0o755); err != nil {
			return "", err
		}
		return scratchDir, nil
	}
	executor.removeAll = func(path string) error {
		removed = append(removed, path)
		return os.RemoveAll(path)
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "scratch-inside-repo",
		Mode: ReviewProbeScratchOnly,
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"check.txt"}},
		},
	})

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "outside repository root") {
		t.Fatalf("Error = %q, want to contain outside repository root", result.Error)
	}
	if len(removed) != 1 {
		t.Fatalf("len(removed) = %d, want 1", len(removed))
	}
	if filepath.Clean(removed[0]) != filepath.Clean(scratchDir) {
		t.Fatalf("removed[0] = %q, want %q", removed[0], scratchDir)
	}
	if _, err := os.Stat(scratchDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scratch dir should be removed, stat error = %v", err)
	}
}
