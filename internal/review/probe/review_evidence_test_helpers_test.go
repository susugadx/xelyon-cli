package probe

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func createReviewEvidenceSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symlink() unavailable: %v", err)
	}
}

func createReviewEvidenceMarkerScript(t *testing.T, name, extraBody string) (string, string) {
	t.Helper()

	binDir := t.TempDir()
	marker := filepath.Join(binDir, name+".marker")
	t.Setenv("REVIEW_EVIDENCE_MARKER", marker)

	body := `printf invoked > "$REVIEW_EVIDENCE_MARKER"`
	if extraBody != "" {
		body += "\n" + extraBody
	}
	createProbeTestScriptCommand(t, binDir, name, body)
	return marker, filepath.Join(binDir, name)
}

func assertReviewEvidenceMarkerAbsent(t *testing.T, marker string) {
	t.Helper()

	assertFileAbsent(t, marker)
}

func assertReviewEvidenceMarkerCanBeInvoked(t *testing.T, repo, marker string, args ...string) {
	t.Helper()

	if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Remove(%q) error = %v", marker, err)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed while checking marker fixture: %v\n%s", args, err, string(out))
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("git %v did not create marker %q: %v\n%s", args, marker, err, string(out))
	}
	if err := os.Remove(marker); err != nil {
		t.Fatalf("Remove(%q) error = %v", marker, err)
	}
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
