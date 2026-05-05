package review

import (
	"context"
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
