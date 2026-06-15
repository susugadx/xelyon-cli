package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewRunDirectoryArtifactWriterWritesFilesWithPrivatePermissionsAndSuffixes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "review-run")
	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v", err)
	}
	if got, want := filePermForReviewArtifactTest(t, dir), os.FileMode(0o700); got != want {
		t.Fatalf("artifact dir mode = %v, want %v", got, want)
	}

	if err := writer.WriteReviewRunArtifact("evidence.md", []byte("first")); err != nil {
		t.Fatalf("WriteReviewRunArtifact(first) error = %v", err)
	}
	if err := writer.WriteReviewRunArtifact("evidence.md", []byte("second")); err != nil {
		t.Fatalf("WriteReviewRunArtifact(second) error = %v", err)
	}

	assertReviewArtifactFileForTest(t, filepath.Join(dir, "evidence.md"), "first")
	assertReviewArtifactFileForTest(t, filepath.Join(dir, "evidence_2.md"), "second")
}

func TestReviewRunDirectoryArtifactWriterAllowsSymlinkedTrustedAncestor(t *testing.T) {
	realBase := t.TempDir()
	base := t.TempDir()
	link := filepath.Join(base, "trusted-ancestor")
	createReviewArtifactSymlinkForTest(t, realBase, link)
	dir := filepath.Join(link, "review-run")

	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v, want nil", err)
	}
	if err := writer.WriteReviewRunArtifact("evidence.md", []byte("through trusted ancestor")); err != nil {
		t.Fatalf("WriteReviewRunArtifact() error = %v", err)
	}

	assertReviewArtifactFileForTest(t, filepath.Join(realBase, "review-run", "evidence.md"), "through trusted ancestor")
}

func TestReviewRunDirectoryArtifactWriterRejectsFinalSymlink(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(base, "review-run")
	createReviewArtifactSymlinkForTest(t, outside, link)

	_, err := NewReviewRunDirectoryArtifactWriter(link)
	if err == nil {
		t.Fatal("NewReviewRunDirectoryArtifactWriter() error = nil, want final symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %q, want symlink", err)
	}
}

func TestReviewRunDirectoryArtifactWriterRejectsFinalSymlinkBeforeWrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "review-run")
	outside := t.TempDir()
	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("RemoveAll(review-run) error = %v", err)
	}
	createReviewArtifactSymlinkForTest(t, outside, dir)

	err = writer.WriteReviewRunArtifact("evidence.md", []byte("must not escape"))
	if err == nil {
		t.Fatal("WriteReviewRunArtifact() error = nil, want final symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("WriteReviewRunArtifact() error = %q, want symlink", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "evidence.md")); !os.IsNotExist(err) {
		t.Fatalf("artifact escaped through final symlink: err=%v", err)
	}
}

func TestReviewRunRepoArtifactWriterWritesUnderRepoLocalDirectory(t *testing.T) {
	repo := t.TempDir()
	writer, err := NewReviewRunRepoArtifactWriter(repo, "20260101T000000.000000000Z")
	if err != nil {
		t.Fatalf("NewReviewRunRepoArtifactWriter() error = %v", err)
	}

	if err := writer.WriteReviewRunArtifact("evidence.md", []byte("repo-local")); err != nil {
		t.Fatalf("WriteReviewRunArtifact() error = %v", err)
	}

	assertReviewArtifactFileForTest(t, filepath.Join(repo, ".xelyon", "review-runs", "20260101T000000.000000000Z", "evidence.md"), "repo-local")
}

func TestReviewRunRepoArtifactWriterRejectsSymlinkedArtifactComponents(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(t *testing.T, repo, outside, runID string)
	}{
		{
			name: ".xelyon symlink",
			setup: func(t *testing.T, repo, outside, _ string) {
				createReviewArtifactSymlinkForTest(t, outside, filepath.Join(repo, ".xelyon"))
			},
		},
		{
			name: "review-runs symlink",
			setup: func(t *testing.T, repo, outside, _ string) {
				if err := os.Mkdir(filepath.Join(repo, ".xelyon"), 0o700); err != nil {
					t.Fatalf("Mkdir(.xelyon) error = %v", err)
				}
				createReviewArtifactSymlinkForTest(t, outside, filepath.Join(repo, ".xelyon", "review-runs"))
			},
		},
		{
			name: "run directory symlink",
			setup: func(t *testing.T, repo, outside, runID string) {
				if err := os.MkdirAll(filepath.Join(repo, ".xelyon", "review-runs"), 0o700); err != nil {
					t.Fatalf("MkdirAll(review-runs) error = %v", err)
				}
				createReviewArtifactSymlinkForTest(t, outside, filepath.Join(repo, ".xelyon", "review-runs", runID))
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := t.TempDir()
			outside := t.TempDir()
			runID := "20260101T000000.000000000Z"
			tt.setup(t, repo, outside, runID)

			_, err := NewReviewRunRepoArtifactWriter(repo, runID)
			if err == nil {
				t.Fatal("NewReviewRunRepoArtifactWriter() error = nil, want symlink rejection")
			}
			if !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("NewReviewRunRepoArtifactWriter() error = %q, want symlink", err)
			}
			for _, escaped := range []string{
				filepath.Join(outside, "review-runs"),
				filepath.Join(outside, runID),
			} {
				if _, err := os.Stat(escaped); !os.IsNotExist(err) {
					t.Fatalf("outside artifact path exists after rejected constructor: %s err=%v", escaped, err)
				}
			}
		})
	}
}

func TestReviewRunRepoArtifactWriterRejectsSymlinkedDirectoryBeforeWrite(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()
	runID := "20260101T000000.000000000Z"
	writer, err := NewReviewRunRepoArtifactWriter(repo, runID)
	if err != nil {
		t.Fatalf("NewReviewRunRepoArtifactWriter() error = %v", err)
	}
	reviewRunsDir := filepath.Join(repo, ".xelyon", "review-runs")
	if err := os.RemoveAll(reviewRunsDir); err != nil {
		t.Fatalf("RemoveAll(review-runs) error = %v", err)
	}
	createReviewArtifactSymlinkForTest(t, outside, reviewRunsDir)

	err = writer.WriteReviewRunArtifact("evidence.md", []byte("must not escape"))
	if err == nil {
		t.Fatal("WriteReviewRunArtifact() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("WriteReviewRunArtifact() error = %q, want symlink", err)
	}
	if _, err := os.Stat(filepath.Join(outside, runID, "evidence.md")); !os.IsNotExist(err) {
		t.Fatalf("artifact escaped through symlinked review-runs: err=%v", err)
	}
}

func TestBufferedReviewRunArtifactWriterFlushesToDirectoryWriter(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "review-run")
	buffer := NewBufferedReviewRunArtifactWriter()
	first := []byte("first")

	if err := buffer.WriteReviewRunArtifact("evidence.md", first); err != nil {
		t.Fatalf("WriteReviewRunArtifact(first) error = %v", err)
	}
	first[0] = 'F'
	if err := buffer.WriteReviewRunArtifact("evidence.md", []byte("second")); err != nil {
		t.Fatalf("WriteReviewRunArtifact(second) error = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("buffer created artifact directory before flush: err=%v", err)
	}

	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v", err)
	}
	if err := buffer.FlushTo(writer); err != nil {
		t.Fatalf("FlushTo() error = %v", err)
	}

	assertReviewArtifactFileForTest(t, filepath.Join(dir, "evidence.md"), "first")
	assertReviewArtifactFileForTest(t, filepath.Join(dir, "evidence_2.md"), "second")
}

func TestBufferedReviewRunArtifactWriterLenNilAndFlushOrder(t *testing.T) {
	var nilBuffer *BufferedReviewRunArtifactWriter
	if got := nilBuffer.Len(); got != 0 {
		t.Fatalf("nil Len() = %d, want 0", got)
	}
	if err := nilBuffer.WriteReviewRunArtifact("evidence.md", []byte("content")); err == nil {
		t.Fatal("nil WriteReviewRunArtifact() error = nil, want nil writer error")
	}
	if err := nilBuffer.FlushTo(&recordingReviewRunArtifactWriter{}); err == nil {
		t.Fatal("nil FlushTo() error = nil, want nil writer error")
	}

	buffer := NewBufferedReviewRunArtifactWriter()
	for _, artifact := range []struct {
		name    string
		content string
	}{
		{name: "first.md", content: "first"},
		{name: "second.md", content: "second"},
	} {
		if err := buffer.WriteReviewRunArtifact(artifact.name, []byte(artifact.content)); err != nil {
			t.Fatalf("WriteReviewRunArtifact(%s) error = %v", artifact.name, err)
		}
	}
	if got := buffer.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
	if err := buffer.FlushTo(nil); err == nil {
		t.Fatal("FlushTo(nil) error = nil, want destination error")
	}

	recorder := &recordingReviewRunArtifactWriter{}
	if err := buffer.FlushTo(recorder); err != nil {
		t.Fatalf("FlushTo(recorder) error = %v", err)
	}
	if got, want := len(recorder.artifacts), 2; got != want {
		t.Fatalf("recorded artifacts len = %d, want %d", got, want)
	}
	if recorder.artifacts[0].name != "first.md" || string(recorder.artifacts[0].content) != "first" ||
		recorder.artifacts[1].name != "second.md" || string(recorder.artifacts[1].content) != "second" {
		t.Fatalf("recorded artifacts = %#v, want flush order preserved", recorder.artifacts)
	}
}

func TestReviewRunArtifactWriterRejectsInvalidNamesAndRepoInputs(t *testing.T) {
	buffer := NewBufferedReviewRunArtifactWriter()
	for _, name := range []string{"", ".", "..", "../evidence.md", "nested/evidence.md", `nested\evidence.md`} {
		t.Run("invalid artifact name "+name, func(t *testing.T) {
			if err := buffer.WriteReviewRunArtifact(name, []byte("content")); err == nil {
				t.Fatalf("WriteReviewRunArtifact(%q) error = nil, want validation error", name)
			}
		})
	}

	repo := t.TempDir()
	for _, runID := range []string{"", ".", "..", "../run", "nested/run", `nested\run`} {
		t.Run("invalid run id "+runID, func(t *testing.T) {
			if _, err := NewReviewRunRepoArtifactWriter(repo, runID); err == nil {
				t.Fatalf("NewReviewRunRepoArtifactWriter(%q) error = nil, want validation error", runID)
			}
		})
	}

	repoFile := filepath.Join(t.TempDir(), "repo-as-file")
	if err := os.WriteFile(repoFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(repo-as-file) error = %v", err)
	}
	if _, err := NewReviewRunRepoArtifactWriter(repoFile, "20260101T000000.000000000Z"); err == nil {
		t.Fatal("NewReviewRunRepoArtifactWriter(file repoRoot) error = nil, want directory validation error")
	}
}

type recordingReviewRunArtifact struct {
	name    string
	content []byte
}

type recordingReviewRunArtifactWriter struct {
	artifacts []recordingReviewRunArtifact
}

func (w *recordingReviewRunArtifactWriter) WriteReviewRunArtifact(name string, content []byte) error {
	w.artifacts = append(w.artifacts, recordingReviewRunArtifact{
		name:    name,
		content: append([]byte(nil), content...),
	})
	return nil
}

func createReviewArtifactSymlinkForTest(t *testing.T, oldname, newname string) {
	t.Helper()

	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}
}

func assertReviewArtifactFileForTest(t *testing.T, path, wantContent string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if string(data) != wantContent {
		t.Fatalf("ReadFile(%s) = %q, want %q", path, data, wantContent)
	}
	if got, want := filePermForReviewArtifactTest(t, path), os.FileMode(0o600); got != want {
		t.Fatalf("%s mode = %v, want %v", path, got, want)
	}
}

func filePermForReviewArtifactTest(t *testing.T, path string) os.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	return info.Mode().Perm()
}
