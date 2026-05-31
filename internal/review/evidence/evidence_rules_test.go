package evidence

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewEvidenceBuilder_RuleFiles(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "AGENTS.md"), strings.Repeat("A", 8))
	writeTestFile(t, filepath.Join(repo, ".codex", "config.toml"), "model = 'x'\n")
	runGit(t, repo, "add", "AGENTS.md", ".codex/config.toml")
	runGit(t, repo, "commit", "-m", "add rule files")

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxRuleFileBytes: 5,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	agents := ruleFileByPath(t, bundle, "AGENTS.md")
	if agents.Content != "AAAAA" || !agents.Truncated || agents.SizeBytes != 8 {
		t.Fatalf("AGENTS.md evidence = %#v, want truncated content", agents)
	}
	codex := ruleFileByPath(t, bundle, ".codex/config.toml")
	if codex.Content != "model" || !codex.Truncated {
		t.Fatalf(".codex/config.toml evidence = %#v, want truncated content", codex)
	}
}

func TestReviewEvidenceBuilder_RuleFilesSkipsNonRegularFile(t *testing.T) {
	if _, err := exec.LookPath("mkfifo"); err != nil {
		t.Skipf("mkfifo not available: %v", err)
	}

	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	cmd := exec.Command("mkfifo", filepath.Join(repo, "AGENTS.md"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("mkfifo AGENTS.md failed: %v\n%s", err, string(out))
	}

	type buildResult struct {
		bundle ReviewEvidenceBundle
		err    error
	}
	resultCh := make(chan buildResult, 1)
	go func() {
		bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
		resultCh <- buildResult{bundle: bundle, err: err}
	}()

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("BuildCurrentChanges() error = %v", result.err)
		}
		for _, file := range result.bundle.RuleFiles {
			if file.Path == "AGENTS.md" {
				t.Fatalf("RuleFiles = %#v, want AGENTS.md skipped because it is non-regular", result.bundle.RuleFiles)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("BuildCurrentChanges() timed out reading non-regular AGENTS.md")
	}
}
