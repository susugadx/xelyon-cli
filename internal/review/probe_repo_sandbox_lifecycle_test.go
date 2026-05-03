package review

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProbeRunner_RepoSandbox_CopiesUntrackedFileAndLeavesOriginalClean(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "untracked.txt"), "untracked\n")
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "repo-sandbox-untracked",
		Mode:           ReviewProbeRepoSandbox,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"untracked.txt"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if !strings.Contains(result.CommandResults[0].Output, "untracked") {
		t.Fatalf("output = %q, want copied untracked file", result.CommandResults[0].Output)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
}

func TestProbeRunner_RepoSandbox_GeneratedPassingAndFailingTests(t *testing.T) {
	tests := []struct {
		name       string
		file       ReviewProbeFile
		runPattern string
		wantStatus ReviewProbeStatus
	}{
		{
			name: "passing generated test",
			file: ReviewProbeFile{
				Path: "probe/generated_pass_test.go",
				Content: "package probe\n\nimport \"testing\"\n\n" +
					"func TestGeneratedPass(t *testing.T) { if Add(1, 2) != 3 { t.Fatal(\"bad add\") } }\n",
			},
			runPattern: "^TestGeneratedPass$",
			wantStatus: ReviewProbePassed,
		},
		{
			name: "failing generated test",
			file: ReviewProbeFile{
				Path: "probe/generated_fail_test.go",
				Content: "package probe\n\nimport \"testing\"\n\n" +
					"func TestGeneratedFail(t *testing.T) { t.Fatal(\"intentional failure\") }\n",
			},
			runPattern: "^TestGeneratedFail$",
			wantStatus: ReviewProbeFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
			runner := NewProbeRunner(repo)

			result, err := runner.Run(context.Background(), ReviewProbeRequest{
				ID:             "repo-sandbox-generated-test",
				Mode:           ReviewProbeRepoSandbox,
				Timeout:        15 * time.Second,
				MaxOutputBytes: 8 * 1024,
				Files:          []ReviewProbeFile{tt.file},
				Commands: []ReviewProbeCommand{
					{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", tt.runPattern}},
				},
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if result.Status != tt.wantStatus {
				t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, tt.wantStatus, result.Error, firstCommandOutput(result))
			}
			if _, err := os.Lstat(filepath.Join(repo, tt.file.Path)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("generated file should not be written to original repo, stat error = %v", err)
			}
			if result.MutatedWorktree {
				t.Fatalf("MutatedWorktree = true, want false")
			}
		})
	}
}

func TestProbeRunner_RepoSandbox_RelativeWorkDirAndSandboxMutationDoNotMarkOriginalMutated(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "repo-sandbox-workdir",
		Mode:           ReviewProbeRepoSandbox,
		Timeout:        15 * time.Second,
		MaxOutputBytes: 4 * 1024,
		Files: []ReviewProbeFile{{
			Path: "probe/sandbox_mutate_test.go",
			Content: "package probe\n\nimport \"os\"\nimport \"testing\"\n\n" +
				"func TestSandboxMutate(t *testing.T) { if err := os.WriteFile(\"sandbox_only.txt\", []byte(\"sandbox\"), 0o644); err != nil { t.Fatal(err) } }\n",
		}},
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "-count=1", ".", "-run", "^TestSandboxMutate$"}, WorkDir: "probe"},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, ReviewProbePassed, result.Error, firstCommandOutput(result))
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
	if _, err := os.Lstat(filepath.Join(repo, "probe", "sandbox_only.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sandbox mutation should not appear in original repo, stat error = %v", err)
	}
}

func TestProbeRunner_RepoSandbox_DetectsOriginalRepoMutation(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "repo-sandbox-original-mutation",
		Mode:           ReviewProbeRepoSandbox,
		Timeout:        15 * time.Second,
		MaxOutputBytes: 4 * 1024,
		Files: []ReviewProbeFile{{
			Path: "mutate_original.go",
			Content: "package main\n\nimport \"os\"\n\n" +
				fmt.Sprintf("func main() { _ = os.WriteFile(%q, []byte(\"mutated\\n\"), 0o644) }\n", filepath.Join(repo, "keep.txt")),
		}},
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"run", "mutate_original.go"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error, firstCommandOutput(result))
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("keep.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain keep.txt", result.MutatedFiles)
	}
}

func TestRepoSandboxExecutor_ChildProcessUsesHardenedEnvAndCleansUp(t *testing.T) {
	pathValue := os.Getenv("PATH")
	if strings.TrimSpace(pathValue) == "" {
		t.Skip("PATH is empty")
	}

	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	sandboxRoot := filepath.Join(t.TempDir(), "xelyon-review-sandbox-env-child")

	executor := newRepoSandboxExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + pathValue,
		"SECRET_TOKEN=secret",
		"OPENAI_API_KEY=sk-xxx",
		"LANG=C.UTF-8",
		"XELYON_REVIEW_ORIGINAL_REPO_ROOT=/should/not/leak",
	}
	executor.mktemp = func(dir, pattern string) (string, error) {
		if err := os.MkdirAll(sandboxRoot, 0o755); err != nil {
			return "", err
		}
		return sandboxRoot, nil
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:             "repo-sandbox-env-child",
		Mode:           ReviewProbeRepoSandbox,
		Timeout:        15 * time.Second,
		MaxOutputBytes: 8 * 1024,
		Files: []ReviewProbeFile{{
			Path: "print_env.go",
			Content: "package main\n" +
				"import (\n" +
				"\t\"encoding/json\"\n" +
				"\t\"os\"\n" +
				")\n" +
				"func main(){\n" +
				"\tkeys := []string{\"PATH\",\"SECRET_TOKEN\",\"OPENAI_API_KEY\",\"XELYON_REVIEW_REPO_ROOT\",\"XELYON_REVIEW_SANDBOX_ROOT\",\"XELYON_REVIEW_ORIGINAL_REPO_ROOT\",\"HOME\",\"TMPDIR\",\"GOCACHE\",\"GOMODCACHE\",\"GOTMPDIR\",\"GOTOOLCHAIN\",\"GOPROXY\",\"GOSUMDB\",\"PYTHONDONTWRITEBYTECODE\",\"PYTHONNOUSERSITE\",\"PIP_NO_INDEX\"}\n" +
				"\tm := map[string]string{}\n" +
				"\tfor _, k := range keys { m[k] = os.Getenv(k) }\n" +
				"\t_ = json.NewEncoder(os.Stdout).Encode(m)\n" +
				"}\n",
		}},
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"run", "print_env.go"}},
		},
	})

	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, ReviewProbePassed, result.Error, firstCommandOutput(result))
	}

	envMap := decodeEnvMapFromFirstCommandOutput(t, result)

	cleanSandboxRoot := filepath.Clean(sandboxRoot)
	wantWorktree := filepath.Join(cleanSandboxRoot, "worktree")
	assertIsolatedProbeEnv(t, envMap, isolatedProbeEnvExpectation{
		repoRootKey:   repoSandboxEnvRepoRoot,
		repoRootValue: wantWorktree,
		modeRootKey:   repoSandboxEnvSandboxRoot,
		modeRootValue: cleanSandboxRoot,
		rootDir:       filepath.Join(cleanSandboxRoot, "runtime"),
	})
	if envMap["XELYON_REVIEW_ORIGINAL_REPO_ROOT"] != "" {
		t.Fatalf("XELYON_REVIEW_ORIGINAL_REPO_ROOT = %q, want empty", envMap["XELYON_REVIEW_ORIGINAL_REPO_ROOT"])
	}
	if _, err := os.Stat(sandboxRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sandbox root should be removed, stat error = %v", err)
	}
}

func TestRepoSandboxExecutor_AppendsCleanupErrorWithoutChangingStatus(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	sandboxRoot := filepath.Join(t.TempDir(), "xelyon-review-sandbox-cleanup-error")
	t.Cleanup(func() {
		_ = os.RemoveAll(sandboxRoot)
	})

	executor := newRepoSandboxExecutor(repo)
	executor.mktemp = func(dir, pattern string) (string, error) {
		if err := os.MkdirAll(sandboxRoot, 0o755); err != nil {
			return "", err
		}
		return sandboxRoot, nil
	}
	executor.removeAll = func(path string) error {
		return errors.New("cleanup failed")
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:             "repo-sandbox-cleanup-error",
		Mode:           ReviewProbeRepoSandbox,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"keep.txt"}},
		},
	})

	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if !strings.Contains(result.Error, "failed to remove repo_sandbox root") {
		t.Fatalf("Error = %q, want cleanup error", result.Error)
	}
}

func firstCommandOutput(result ReviewProbeResult) string {
	if len(result.CommandResults) == 0 {
		return ""
	}
	return result.CommandResults[0].Output
}
