package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectMapRoot_UsesGitRootWithoutXelyonYAML(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	cwd := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGit("init")

	bundle, err := config.LoadProjectInstructionBundleForDir(config.DefaultConfig(), cwd)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}

	got := resolveProjectMapSourceRootPath(cwd, bundle)
	if got != root {
		t.Fatalf("project map root = %q, want git root %q", got, root)
	}
}

func TestProjectMapRoot_SkipsCwdFallbackWithoutProjectRoot(t *testing.T) {
	cwd := t.TempDir()

	bundle, err := config.LoadProjectInstructionBundleForDir(config.DefaultConfig(), cwd)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}

	got := resolveProjectMapSourceRootPath(cwd, bundle)
	if got != "" {
		t.Fatalf("project map root = %q, want empty when only cwd fallback is available", got)
	}
}

func TestProjectMapRoot_UsesGuidanceRootWithoutGitOrXelyon(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# guidance\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bundle, err := config.LoadProjectInstructionBundleForDir(config.DefaultConfig(), cwd)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}

	got := resolveProjectMapSourceRootPath(cwd, bundle)
	if got != root {
		t.Fatalf("project map root = %q, want guidance root %q", got, root)
	}
}

func TestProjectMapRoot_UsesXelyonYAMLDirectory(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	cwd := filepath.Join(projectRoot, "sub")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "xelyon.yaml"), []byte("context: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bundle, err := config.LoadProjectInstructionBundleForDir(config.DefaultConfig(), cwd)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}

	got := resolveProjectMapSourceRootPath(cwd, bundle)
	if got != projectRoot {
		t.Fatalf("project map root = %q, want xelyon dir %q", got, projectRoot)
	}
}
