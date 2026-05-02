package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func newProbeTestRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)

	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	writeTestFile(t, filepath.Join(repo, "go.mod"), "module example.com/reviewprobe\n\ngo 1.26.0\n")
	writeTestFile(t, filepath.Join(repo, "probe", "probe.go"), "package probe\n\nfunc Add(a, b int) int { return a + b }\n")
	writeTestFile(t, filepath.Join(repo, "probe", "probe_test.go"), `package probe

import (
	"os"
	"testing"
	"time"
)

func TestProbeSleep(t *testing.T) {
	time.Sleep(1 * time.Second)
}

func TestProbeMutate(t *testing.T) {
	if err := os.WriteFile("probe_generated.txt", []byte("generated"), 0o644); err != nil {
		t.Fatalf("write probe_generated.txt: %v", err)
	}
}

func TestProbeMutateDirtyExistingPath(t *testing.T) {
	if err := os.WriteFile("../keep.txt", []byte("probe-updated-existing-dirty-path\n"), 0o644); err != nil {
		t.Fatalf("write ../keep.txt: %v", err)
	}
}
`)
	writeTestFile(t, filepath.Join(repo, "large.txt"), strings.Repeat("large-output-line\n", 128))
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "keep\n")

	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")

	return repo
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if filepath.ToSlash(item) == filepath.ToSlash(target) {
			return true
		}
	}
	return false
}
