package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	probeTestModulePath = "example.com/reviewprobe"
)

type probeTestRepoConfig struct {
	includeLargeFile bool
	includeKeepFile  bool
}

type probeTestRepoOption func(*probeTestRepoConfig)

func withProbeTestRepoNoLargeFile() probeTestRepoOption {
	return func(cfg *probeTestRepoConfig) {
		cfg.includeLargeFile = false
	}
}

func newProbeTestRepo(t *testing.T, opts ...probeTestRepoOption) string {
	t.Helper()
	requireGit(t)

	cfg := probeTestRepoConfig{
		includeLargeFile: true,
		includeKeepFile:  true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	repo := t.TempDir()
	initProbeTestGitRepo(t, repo)
	writeProbeTestScaffoldFiles(t, repo, cfg)
	commitProbeTestRepo(t, repo)

	return repo
}

func initProbeTestGitRepo(t *testing.T, repo string) {
	t.Helper()

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")
}

func writeProbeTestScaffoldFiles(t *testing.T, repo string, cfg probeTestRepoConfig) {
	t.Helper()

	writeProbeTestGoMod(t, repo)
	writeProbeTestPackageFiles(t, repo)
	if cfg.includeLargeFile {
		writeTestFile(t, filepath.Join(repo, "large.txt"), strings.Repeat("large-output-line\n", 128))
	}
	if cfg.includeKeepFile {
		writeTestFile(t, filepath.Join(repo, "keep.txt"), "keep\n")
	}
}

func writeProbeTestGoMod(t *testing.T, repo string) {
	t.Helper()

	writeTestFile(t, filepath.Join(repo, "go.mod"), "module "+probeTestModulePath+"\n\ngo "+readRepositoryGoVersionForProbeTests(t)+"\n")
}

func readRepositoryGoVersionForProbeTests(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	goModPath := filepath.Join(repoRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", goModPath, err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "go ") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && fields[1] != "" {
			return fields[1]
		}
	}

	t.Fatalf("go directive not found in %q", goModPath)
	return ""
}

func writeProbeTestPackageFiles(t *testing.T, repo string) {
	t.Helper()

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
}

func commitProbeTestRepo(t *testing.T, repo string) {
	t.Helper()

	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
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

func createProbeTestScriptCommand(t *testing.T, binDir, name, scriptBody string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("script command helper is unix-only")
	}

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", binDir, err)
	}
	path := filepath.Join(binDir, name)
	content := "#!/bin/sh\nset -eu\n" + scriptBody + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
