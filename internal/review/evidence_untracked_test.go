package review

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_UntrackedTextSnapshot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "notes.txt"), "review notes\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if got, want := bundle.UntrackedFiles, []ReviewUntrackedFile{{
		Path:      "notes.txt",
		Snapshot:  "review notes\n",
		SizeBytes: int64(len("review notes\n")),
		ReadBytes: int64(len("review notes\n")),
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("UntrackedFiles = %#v, want %#v", got, want)
	}
	if bundle.UntrackedSnapshotsTruncated {
		t.Fatal("UntrackedSnapshotsTruncated = true, want false")
	}
	if !containsString(bundle.Inventory.Untracked, "notes.txt") {
		t.Fatalf("Inventory.Untracked = %#v, want notes.txt", bundle.Inventory.Untracked)
	}
}

func TestReviewEvidenceBuilder_UntrackedPathUsesNULOutputWhenQuotePathEnabled(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runGit(t, repo, "config", "core.quotePath", "true")
	writeTestFile(t, filepath.Join(repo, "日本語.txt"), "review notes\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if got := bundle.UntrackedFiles; len(got) != 1 || got[0].Path != "日本語.txt" {
		t.Fatalf("UntrackedFiles = %#v, want 日本語.txt", got)
	}
	assertStringSlice(t, bundle.Inventory.Untracked, []string{"日本語.txt"})
}

func TestReviewEvidenceBuilder_UntrackedBinarySnapshot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	path := filepath.Join(repo, "bin.dat")
	if err := os.WriteFile(path, []byte{'a', 0, 'b'}, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if got := bundle.UntrackedFiles; len(got) != 1 || got[0].Path != "bin.dat" || !got[0].Binary || got[0].Snapshot != "" {
		t.Fatalf("UntrackedFiles = %#v, want binary snapshot without content", got)
	}
}

func TestReviewEvidenceBuilder_UntrackedFileSnapshotTruncationDoesNotTruncateSnapshotSet(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "long.txt"), "abcdef")

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxUntrackedFileBytes: 4,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if got := bundle.UntrackedFiles; len(got) != 1 || got[0].Snapshot != "abcd" || !got[0].Truncated || got[0].ReadBytes != 4 {
		t.Fatalf("UntrackedFiles = %#v, want truncated text snapshot", got)
	}
	if bundle.UntrackedSnapshotsTruncated {
		t.Fatal("UntrackedSnapshotsTruncated = true, want false for per-file truncation only")
	}
}

func TestReviewEvidenceBuilder_UntrackedFileLimitTruncatesSnapshotSet(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "a.txt"), "a\n")
	writeTestFile(t, filepath.Join(repo, "b.txt"), "b\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxUntrackedFiles: 1,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if !bundle.UntrackedSnapshotsTruncated {
		t.Fatal("UntrackedSnapshotsTruncated = false, want true")
	}
	if got := bundle.UntrackedFiles; len(got) != 1 {
		t.Fatalf("len(UntrackedFiles) = %d, want 1: %#v", len(got), got)
	}
}

func TestReviewEvidenceBuilder_UntrackedTotalByteLimitTruncatesSnapshotSet(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "long.txt"), "abcdef")

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxTotalUntrackedBytes: 4,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if !bundle.UntrackedSnapshotsTruncated {
		t.Fatal("UntrackedSnapshotsTruncated = false, want true")
	}
	if got := bundle.UntrackedFiles; len(got) != 1 || got[0].Snapshot != "abcd" || !got[0].Truncated || got[0].ReadBytes != 4 {
		t.Fatalf("UntrackedFiles = %#v, want total-budget truncated text snapshot", got)
	}
}

func TestReviewEvidenceBuilder_UntrackedListTruncationDropsPartialPath(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "notes.txt"), "notes\n")

	limit := int64(len("notes.txt\x00par"))
	bundle, err := NewReviewEvidenceBuilder(repo, repo,
		WithReviewEvidenceLimits(ReviewEvidenceLimits{
			MaxCommandOutputBytes: limit,
		}),
		WithReviewEvidenceCommandRunner(fakeReviewEvidenceRunner{
			outputs: map[string]string{
				fakeReviewEvidenceGitKey(reviewUntrackedListGitArgs()...): "notes.txt\x00partial.txt\x00",
			},
		}),
	).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if !bundle.UntrackedListTruncated {
		t.Fatal("UntrackedListTruncated = false, want true")
	}
	if bundle.UntrackedSnapshotsTruncated {
		t.Fatal("UntrackedSnapshotsTruncated = true, want false")
	}
	if got := bundle.UntrackedFiles; len(got) != 1 || got[0].Path != "notes.txt" {
		t.Fatalf("UntrackedFiles = %#v, want only complete notes.txt entry", got)
	}
}

func TestReviewEvidenceBuilder_UntrackedDirectoryIsSkipped(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	if err := os.Mkdir(filepath.Join(repo, "empty-dir"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceCommandRunner(fakeReviewEvidenceRunner{
		outputs: map[string]string{
			fakeReviewEvidenceGitKey(reviewUntrackedListGitArgs()...): "empty-dir\x00",
		},
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}
	if len(bundle.UntrackedFiles) != 0 {
		t.Fatalf("UntrackedFiles = %#v, want empty because directory is skipped", bundle.UntrackedFiles)
	}
}

func TestReviewEvidenceBuilder_UntrackedSymlinkToTrackedFileRecordsMetadataOnly(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	secret := "tracked secret body\n"
	writeTestFile(t, filepath.Join(repo, "tracked-secret.txt"), secret)
	runGit(t, repo, "add", "tracked-secret.txt")
	runGit(t, repo, "commit", "-m", "add tracked secret")
	createReviewEvidenceSymlink(t, "tracked-secret.txt", filepath.Join(repo, "secret-link"))

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	got := requireReviewUntrackedFile(t, bundle, "secret-link")
	want := ReviewUntrackedFile{
		Path:       "secret-link",
		Symlink:    true,
		LinkTarget: "tracked-secret.txt",
		SizeBytes:  int64(len("tracked-secret.txt")),
		ReadBytes:  int64(len("tracked-secret.txt")),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UntrackedFiles[secret-link] = %#v, want %#v", got, want)
	}
	if strings.Contains(got.Snapshot, secret) || strings.Contains(got.LinkTarget, secret) {
		t.Fatalf("symlink evidence leaked tracked file content: %#v", got)
	}
}

func TestReviewEvidenceBuilder_UntrackedSymlinkEscapingRepoRecordsMetadataOnly(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("outside\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", target, err)
	}
	createReviewEvidenceSymlink(t, target, filepath.Join(repo, "outside-link"))

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	got := requireReviewUntrackedFile(t, bundle, "outside-link")
	if !got.Symlink || got.LinkTarget != target || got.Snapshot != "" || got.Binary {
		t.Fatalf("UntrackedFiles[outside-link] = %#v, want symlink metadata for outside target", got)
	}
}

func TestReviewEvidenceBuilder_UntrackedSymlinkToDirectoryIsCollected(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "linked-dir", "file.txt"), "tracked directory file\n")
	runGit(t, repo, "add", "linked-dir/file.txt")
	runGit(t, repo, "commit", "-m", "add linked directory")
	createReviewEvidenceSymlink(t, "linked-dir", filepath.Join(repo, "dir-link"))

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	got := requireReviewUntrackedFile(t, bundle, "dir-link")
	if !got.Symlink || got.LinkTarget != "linked-dir" {
		t.Fatalf("UntrackedFiles[dir-link] = %#v, want symlink metadata", got)
	}
}

func TestReviewEvidenceBuilder_UntrackedSymlinkTargetUsesSnapshotBudget(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	target := strings.Repeat("target", 6)
	createReviewEvidenceSymlink(t, target, filepath.Join(repo, "long-link"))

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxUntrackedFileBytes: 8,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	got := requireReviewUntrackedFile(t, bundle, "long-link")
	if got.LinkTarget != target[:8] || !got.Truncated || got.SizeBytes != int64(len(target)) || got.ReadBytes != 8 {
		t.Fatalf("UntrackedFiles[long-link] = %#v, want truncated symlink target metadata", got)
	}
	if bundle.UntrackedSnapshotsTruncated {
		t.Fatal("UntrackedSnapshotsTruncated = true, want false for per-file symlink target truncation only")
	}
}
