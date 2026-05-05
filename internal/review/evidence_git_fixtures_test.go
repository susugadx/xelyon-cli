package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func createReviewEvidenceSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symlink() unavailable: %v", err)
	}
}

func createReviewEvidenceIndexAtHead(t *testing.T, repo string) string {
	t.Helper()

	indexPath := filepath.Join(t.TempDir(), "alternate.index")
	cmd := exec.Command("git", "read-tree", "HEAD")
	cmd.Dir = repo
	cmd.Env = appendReviewEvidenceTestEnv(os.Environ(), "GIT_INDEX_FILE", indexPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git read-tree HEAD with alternate index failed: %v\n%s", err, string(out))
	}
	return indexPath
}

func appendReviewEvidenceTestEnv(environ []string, key, value string) []string {
	cleaned := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		currentKey, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(currentKey, key) {
			continue
		}
		cleaned = append(cleaned, entry)
	}
	return append(cleaned, key+"="+value)
}

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
