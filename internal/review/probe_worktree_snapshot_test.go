package review

import (
	"context"
	"path/filepath"
	"reflect"
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
