package evidence

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_CurrentChangesRenameUsesNULOutputWhenQuotePathEnabled(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runGit(t, repo, "config", "core.quotePath", "true")
	writeTestFile(t, filepath.Join(repo, "日本語.txt"), "tracked\n")
	runGit(t, repo, "add", "日本語.txt")
	runGit(t, repo, "commit", "-m", "add japanese path")
	runGit(t, repo, "mv", "日本語.txt", "名前変更.txt")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:    "名前変更.txt",
		OldPath: "日本語.txt",
		Status:  "R100",
		Staged:  true,
	}})
	assertStringSlice(t, bundle.Inventory.RenamedFiles, []string{"名前変更.txt"})

	staged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceStaged)
	if strings.Contains(staged.NameStatus, "\x00") {
		t.Fatalf("NameStatus contains raw NUL output: %q", staged.NameStatus)
	}
	if !strings.Contains(staged.NameStatus, "R100\t日本語.txt\t名前変更.txt\n") {
		t.Fatalf("NameStatus = %q, want unquoted display paths", staged.NameStatus)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesNameStatusQuotesControlPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("control bytes are not valid Windows path characters")
	}

	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	rawPath := "a\nb\t.txt"
	writeTestFile(t, filepath.Join(repo, filepath.FromSlash(rawPath)), "tracked\n")
	runGit(t, repo, "add", rawPath)
	runGit(t, repo, "commit", "-m", "add control path")
	writeTestFile(t, filepath.Join(repo, filepath.FromSlash(rawPath)), "changed\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:     rawPath,
		Status:   "M",
		Unstaged: true,
	}})
	assertStringSlice(t, bundle.Inventory.Production, []string{rawPath})

	unstaged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceUnstaged)
	if got, want := unstaged.NameStatus, "M\t\"a\\nb\\t.txt\"\n"; got != want {
		t.Fatalf("NameStatus = %q, want %q", got, want)
	}
	if strings.Contains(strings.TrimSuffix(unstaged.NameStatus, "\n"), "\n") {
		t.Fatalf("NameStatus contains raw newline inside record: %q", unstaged.NameStatus)
	}
	if strings.Count(strings.TrimSuffix(unstaged.NameStatus, "\n"), "\t") != 1 {
		t.Fatalf("NameStatus contains raw tab outside status delimiter: %q", unstaged.NameStatus)
	}
}

func TestFormatReviewEvidencePathDisplay(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	insideDir := filepath.Join(repo, "src")
	if err := os.MkdirAll(insideDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", insideDir, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")

	tests := []struct {
		name      string
		candidate string
		want      string
	}{
		{
			name:      "repo relative path",
			candidate: "src/../src/main.go",
			want:      "src/main.go",
		},
		{
			name:      "repo root",
			candidate: repo,
			want:      ".",
		},
		{
			name:      "absolute path inside repo",
			candidate: filepath.Join(repo, "src", "main.go"),
			want:      "src/main.go",
		},
		{
			name:      "outside repo",
			candidate: outside,
			want:      reviewEvidenceOutsideRepoPathDisplay,
		},
		{
			name:      "dotdot relative escape",
			candidate: "../outside.txt",
			want:      reviewEvidenceOutsideRepoPathDisplay,
		},
		{
			name:      "empty path",
			candidate: "",
			want:      reviewEvidenceOutsideRepoPathDisplay,
		},
	}

	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name      string
			candidate string
			want      string
		}{
			name:      "windows absolute path on non windows",
			candidate: `C:\repo\src\main.go`,
			want:      reviewEvidenceOutsideRepoPathDisplay,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatReviewEvidencePathDisplay(repo, tt.candidate)
			if got != tt.want {
				t.Fatalf("formatReviewEvidencePathDisplay(%q, %q) = %q, want %q", repo, tt.candidate, got, tt.want)
			}
			if got != reviewEvidenceOutsideRepoPathDisplay && filepath.IsAbs(got) {
				t.Fatalf("formatReviewEvidencePathDisplay(%q, %q) leaked absolute path %q", repo, tt.candidate, got)
			}
		})
	}
}

func TestFormatReviewEvidencePathDisplayDoesNotResolveSymlinkRepoRootCWD(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	nested := filepath.Join(repo, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", nested, err)
	}

	linkRoot := filepath.Join(t.TempDir(), "repo-link")
	createReviewEvidenceSymlink(t, repo, linkRoot)
	linkCWD := filepath.Join(linkRoot, "nested")

	got := formatReviewEvidencePathDisplay(repo, linkCWD)
	if got != reviewEvidenceOutsideRepoPathDisplay {
		t.Fatalf("formatReviewEvidencePathDisplay(%q, %q) = %q, want %q", repo, linkCWD, got, reviewEvidenceOutsideRepoPathDisplay)
	}
	if strings.Contains(got, repo) || strings.Contains(got, linkRoot) || filepath.IsAbs(got) {
		t.Fatalf("formatReviewEvidencePathDisplay(%q, %q) leaked absolute context %q", repo, linkCWD, got)
	}
}
