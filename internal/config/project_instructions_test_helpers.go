package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func createSymlinkOrSkip(t *testing.T, targetPath, linkPath string) {
	t.Helper()
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
}

func loadProjectInstructionBundleForDirOrFatal(t *testing.T, cfg *Config, dir string) *ProjectInstructionBundle {
	t.Helper()
	bundle, err := LoadProjectInstructionBundleForDir(cfg, dir)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
	return bundle
}
