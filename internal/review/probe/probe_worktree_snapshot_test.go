package probe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiffWorktreeSnapshots_DetectsFingerprintOnlyChange(t *testing.T) {
	before := worktreeSnapshot{
		entries: map[string]worktreeSnapshotEntry{
			"keep.txt": {
				statusCode:  " M",
				fingerprint: "file:-rw-r--r--:before",
			},
		},
	}
	after := worktreeSnapshot{
		entries: map[string]worktreeSnapshotEntry{
			"keep.txt": {
				statusCode:  " M",
				fingerprint: "file:-rw-r--r--:after",
			},
		},
	}

	got := diffWorktreeSnapshots(before, after)
	want := []string{"keep.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diffWorktreeSnapshots() = %#v, want %#v", got, want)
	}
}

func TestBuildWorktreeFingerprint_LargeRegularFileUsesBoundedFingerprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.txt")
	writeTestFile(t, path, strings.Repeat("x", int(defaultWorktreeSnapshotMaxHashBytes)+1))

	got, err := buildWorktreeFingerprint(path)
	if err != nil {
		t.Fatalf("buildWorktreeFingerprint() error = %v", err)
	}
	if !strings.HasPrefix(got, "file-large:") {
		t.Fatalf("buildWorktreeFingerprint() = %q, want large-file fingerprint", got)
	}
	if strings.Contains(got, strings.Repeat("x", 16)) {
		t.Fatalf("buildWorktreeFingerprint() leaked file content: %q", got)
	}
}

func TestParseGitStatusEntriesPorcelainV1Z_PathContainsArrow(t *testing.T) {
	status := []byte(" M foo -> bar.txt\x00")
	got := parseGitStatusEntriesPorcelainV1Z(status)
	if _, ok := got["foo -> bar.txt"]; !ok {
		t.Fatalf("parseGitStatusEntriesPorcelainV1Z(%q) should keep full path, got %#v", status, got)
	}
}

func TestParseGitStatusEntriesPorcelainV1Z_RenameEntry(t *testing.T) {
	status := []byte("R  new.txt\x00old.txt\x00")
	got := parseGitStatusEntriesPorcelainV1Z(status)
	if len(got) != 1 {
		t.Fatalf("len(parseGitStatusEntriesPorcelainV1Z(%q)) = %d, want 1", status, len(got))
	}
	if code, ok := got["new.txt"]; !ok || code != "R " {
		t.Fatalf("parseGitStatusEntriesPorcelainV1Z(%q) = %#v, want new.txt => R ", status, got)
	}
}

func TestCaptureWorktreeSnapshot_UntrackedDirectoryUsesFileEntries(t *testing.T) {
	repo := newProbeTestRepo(t)

	writeTestFile(t, filepath.Join(repo, "tmpdir", "file.txt"), "aaaa\n")

	before, err := captureWorktreeSnapshot(context.Background(), repo)
	if err != nil {
		t.Fatalf("captureWorktreeSnapshot(before) error = %v", err)
	}

	writeTestFile(t, filepath.Join(repo, "tmpdir", "file.txt"), "bbbb\n")

	after, err := captureWorktreeSnapshot(context.Background(), repo)
	if err != nil {
		t.Fatalf("captureWorktreeSnapshot(after) error = %v", err)
	}

	got := diffWorktreeSnapshots(before, after)
	want := []string{"tmpdir/file.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diffWorktreeSnapshots() = %#v, want %#v", got, want)
	}
}

func TestCaptureWorktreeSnapshot_DisablesConfiguredFSMonitor(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	marker, helper := createReviewEvidenceMarkerScript(t, "fsmonitor", "")
	runGit(t, repo, "config", "core.fsmonitor", helper)
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "fsmonitor safety\n")

	snapshot, err := captureWorktreeSnapshot(context.Background(), repo)
	if err != nil {
		t.Fatalf("captureWorktreeSnapshot() error = %v", err)
	}

	if entry, ok := snapshot.entries["keep.txt"]; !ok || entry.statusCode != " M" {
		t.Fatalf("snapshot.entries[keep.txt] = %#v, %v; want modified keep.txt", entry, ok)
	}
	assertReviewEvidenceMarkerAbsent(t, marker)
	assertReviewEvidenceMarkerCanBeInvoked(t, repo, marker, "status", "--short")
}

func TestCaptureWorktreeSnapshot_BlocksRepoControlledGitExecutableOnPATH(t *testing.T) {
	repo := newProbeTestRepo(t)
	repoBin := filepath.Join(repo, "bin")
	marker := filepath.Join(t.TempDir(), "repo-git.marker")
	t.Setenv("REVIEW_PROBE_MARKER", marker)
	createProbeTestScriptCommand(t, repoBin, "git", `printf invoked > "$REVIEW_PROBE_MARKER"`)
	t.Setenv("PATH", strings.Join([]string{repoBin, os.Getenv("PATH")}, string(os.PathListSeparator)))

	_, err := captureWorktreeSnapshot(context.Background(), repo)
	if err == nil {
		t.Fatal("captureWorktreeSnapshot() error = nil, want repo-controlled git to be blocked")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("captureWorktreeSnapshot() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	assertFileAbsent(t, marker)
}

func TestCaptureWorktreeSnapshot_BlocksRepoControlledGitExecutableThroughSymlinkedRepoRoot(t *testing.T) {
	repo := newProbeTestRepo(t)
	linkRoot := filepath.Join(t.TempDir(), "repo-link")
	createReviewEvidenceSymlink(t, repo, linkRoot)

	repoBin := filepath.Join(repo, "bin")
	marker := filepath.Join(t.TempDir(), "repo-git.marker")
	t.Setenv("REVIEW_PROBE_MARKER", marker)
	createProbeTestScriptCommand(t, repoBin, "git", `printf invoked > "$REVIEW_PROBE_MARKER"`)
	t.Setenv("PATH", strings.Join([]string{repoBin, os.Getenv("PATH")}, string(os.PathListSeparator)))

	_, err := captureWorktreeSnapshot(context.Background(), linkRoot)
	if err == nil {
		t.Fatal("captureWorktreeSnapshot() error = nil, want repo-controlled git behind symlink root to be blocked")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("captureWorktreeSnapshot() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	assertFileAbsent(t, marker)
}
