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
	_, runner := newScratchOnlyProbeRunner(t)

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
	_, runner := newScratchOnlyProbeRunner(t)

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
	_, runner := newScratchOnlyProbeRunner(t)

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

func TestProbeRunner_ScratchOnly_Cleanup(t *testing.T) {
	_, runner := newScratchOnlyProbeRunner(t)

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
	_, runner := newScratchOnlyProbeRunner(t)

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
