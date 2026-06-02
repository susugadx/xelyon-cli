package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func reviewEvidenceTestPathWithGit(t *testing.T, entries ...string) string {
	t.Helper()

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}
	if !filepath.IsAbs(gitPath) {
		gitPath, err = filepath.Abs(gitPath)
		if err != nil {
			t.Fatalf("Abs(git path) error = %v", err)
		}
	}

	pathEntries := make([]string, 0, len(entries)+1)
	pathEntries = append(pathEntries, entries...)
	pathEntries = append(pathEntries, filepath.Dir(filepath.Clean(gitPath)))
	return strings.Join(pathEntries, string(os.PathListSeparator))
}

func assertFileAbsent(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("file %q exists and ReadFile failed: %v", path, readErr)
		}
		t.Fatalf("file %q was created with %q", path, string(content))
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}
