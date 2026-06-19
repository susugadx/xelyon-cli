package evidence

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_CurrentChangesCollectsChangedAndRelatedGoContext(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "probe", "probe.go"), `package probe

type Calculator struct{}

func Add(a, b int) int { return a + b }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	changed := requireReviewContextFile(t, bundle.ChangedFileContext, "probe/probe.go")
	if changed.Role != reviewContextFileRoleChanged {
		t.Fatalf("changed role = %q, want %q", changed.Role, reviewContextFileRoleChanged)
	}
	if changed.Skipped {
		t.Fatalf("changed context skipped: %#v", changed)
	}
	if !strings.Contains(changed.Content, "type Calculator struct{}") {
		t.Fatalf("changed context content = %q, want Calculator type", changed.Content)
	}

	related := requireReviewContextFile(t, bundle.RelatedContextFiles, "probe/probe_test.go")
	if related.Role != reviewContextFileRoleRelatedTest {
		t.Fatalf("related role = %q, want %q", related.Role, reviewContextFileRoleRelatedTest)
	}
	if related.Skipped {
		t.Fatalf("related context skipped: %#v", related)
	}
	if !strings.Contains(related.Content, "func TestProbeSleep") {
		t.Fatalf("related context content = %q, want existing related test", related.Content)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "probe/probe_test.go")
	if hit.Line <= 0 {
		t.Fatalf("search hit line = %d, want positive line", hit.Line)
	}
	if hit.Snippet == "" || hit.Reason == "" {
		t.Fatalf("search hit = %#v, want snippet and reason", hit)
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)
	for _, want := range []string{
		"## changed file context\n",
		"### changed context file: probe/probe.go\n",
		"type Calculator struct{}",
		"## related tests/context files\n",
		"### related context file: probe/probe_test.go\n",
		"func TestProbeSleep",
		"## related search hits\n",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want context fragment %q", markdown, want)
		}
	}
}

func TestReviewEvidenceBuilder_CurrentChangesUsesUntrackedGoFileAsContextSeed(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "probe", "new_feature_test.go"), `package probe

func TestNewFeature(t *testing.T) {
	_ = NewFeature()
}
`)
	runGit(t, repo, "add", "probe/new_feature_test.go")
	runGit(t, repo, "commit", "-m", "add future feature test")

	writeTestFile(t, filepath.Join(repo, "probe", "new_feature.go"), `package probe

type UntrackedFeature struct{}

func NewFeature() UntrackedFeature { return UntrackedFeature{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if len(bundle.ChangedFiles) != 0 {
		t.Fatalf("ChangedFiles = %#v, want no git diff changed files for untracked seed", bundle.ChangedFiles)
	}
	changed := requireReviewContextFile(t, bundle.ChangedFileContext, "probe/new_feature.go")
	if changed.Role != reviewContextFileRoleChanged || changed.Skipped {
		t.Fatalf("untracked changed context = %#v, want readable changed context", changed)
	}
	if !strings.Contains(changed.Content, "func NewFeature()") {
		t.Fatalf("untracked changed context content = %q, want NewFeature", changed.Content)
	}
	requireReviewContextFile(t, bundle.RelatedContextFiles, "probe/new_feature_test.go")
	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "probe/new_feature_test.go")
	if hit.Reason != "symbol:NewFeature" {
		t.Fatalf("untracked related search hit = %#v, want symbol:NewFeature", hit)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesUsesUntrackedReplacementForDeletedGoFileContextSeed(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "probe", "replace.go"), `package probe

func OldReplace() int { return 1 }
`)
	writeTestFile(t, filepath.Join(repo, "probe", "replace_test.go"), `package probe

func TestReplace(t *testing.T) {
	_ = NewReplace()
}
`)
	runGit(t, repo, "add", "probe/replace.go", "probe/replace_test.go")
	runGit(t, repo, "commit", "-m", "add replace fixture")
	runGit(t, repo, "rm", "probe/replace.go")
	writeTestFile(t, filepath.Join(repo, "probe", "replace.go"), `package probe

type Replacement struct{}

func NewReplace() Replacement { return Replacement{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	changed := requireReviewContextFile(t, bundle.ChangedFileContext, "probe/replace.go")
	if changed.Skipped || !strings.Contains(changed.Content, "func NewReplace()") {
		t.Fatalf("replacement changed context = %#v, want recreated untracked content", changed)
	}
	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "probe/replace_test.go")
	if hit.Reason != "symbol:NewReplace" {
		t.Fatalf("replacement related search hit = %#v, want symbol:NewReplace", hit)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesSkipsLargeChangedFileContext(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "larger than five bytes\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:  5,
		MaxTotalContextBytes: 100,
		MaxContextFiles:      10,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	got := requireReviewContextFile(t, bundle.ChangedFileContext, "keep.txt")
	if !got.Skipped || got.SkipReason != reviewContextSkipTooLarge {
		t.Fatalf("changed context = %#v, want too_large skip", got)
	}
	if got.Content != "" {
		t.Fatalf("large changed context content = %q, want empty", got.Content)
	}
	if got.SizeBytes <= 5 {
		t.Fatalf("large changed context size = %d, want > 5", got.SizeBytes)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesDoesNotReadDeletedFileContext(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runGit(t, repo, "rm", "keep.txt")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if hasReviewContextFile(bundle.ChangedFileContext, "keep.txt") {
		t.Fatalf("ChangedFileContext = %#v, want deleted keep.txt omitted", bundle.ChangedFileContext)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesSkipsGeneratedAndVendorContext(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "internal", "service.gen.go"), "package internal\n\nfunc Generated() {}\n")
	writeTestFile(t, filepath.Join(repo, "vendor", "example.com", "pkg", "pkg.go"), "package pkg\n\nfunc Vendor() {}\n")
	runGit(t, repo, "add", "internal/service.gen.go", "vendor/example.com/pkg/pkg.go")
	runGit(t, repo, "commit", "-m", "add excluded context files")

	writeTestFile(t, filepath.Join(repo, "internal", "service.gen.go"), "package internal\n\nfunc GeneratedChanged() {}\n")
	writeTestFile(t, filepath.Join(repo, "vendor", "example.com", "pkg", "pkg.go"), "package pkg\n\nfunc VendorChanged() {}\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	generated := requireReviewContextFile(t, bundle.ChangedFileContext, "internal/service.gen.go")
	if !generated.Skipped || generated.SkipReason != reviewContextSkipGenerated || generated.Content != "" {
		t.Fatalf("generated context = %#v, want generated skip without content", generated)
	}
	vendor := requireReviewContextFile(t, bundle.ChangedFileContext, "vendor/example.com/pkg/pkg.go")
	if !vendor.Skipped || vendor.SkipReason != reviewContextSkipVendor || vendor.Content != "" {
		t.Fatalf("vendor context = %#v, want vendor skip without content", vendor)
	}
}

func TestReviewFileEvidenceCollector_ContextStatFailureDoesNotFailBuild(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	fileEvidence, err := reviewFileEvidenceCollector{
		limits: DefaultReviewEvidenceLimits(),
	}.collectCurrentChanges(context.Background(), reviewFileEvidenceCollectionInput{
		repoRoot: repo,
		changedFiles: []ReviewChangedFile{{
			Path:   "missing.go",
			Status: "M",
		}},
	})
	if err != nil {
		t.Fatalf("collectCurrentChanges() error = %v", err)
	}

	got := requireReviewContextFile(t, fileEvidence.changedFileContext, "missing.go")
	if !got.Skipped || got.SkipReason != reviewContextSkipStatFailed {
		t.Fatalf("missing changed context = %#v, want stat_failed skip", got)
	}
}

func assertReviewContextFilePaths(t *testing.T, files []ReviewContextFileEvidence, want []string) {
	t.Helper()
	assertStringSlice(t, reviewContextFilePaths(files), want)
}

func reviewContextFilePaths(files []ReviewContextFileEvidence) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, filepath.ToSlash(file.Path))
	}
	return paths
}
