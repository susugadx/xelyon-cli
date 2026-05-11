package review

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

func TestReviewEvidenceBuilder_CurrentChangesExcludesGitIgnoredRelatedGoFiles(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, ".gitignore"), "probe/secret.go\n")
	runGit(t, repo, "add", ".gitignore")
	runGit(t, repo, "commit", "-m", "ignore local secret")

	writeTestFile(t, filepath.Join(repo, "probe", "probe.go"), `package probe

type Calculator struct{}

func Add(a, b int) int { return a + b }
`)
	writeTestFile(t, filepath.Join(repo, "probe", "secret.go"), `package probe

func HiddenSecret() string { return "do-not-send" }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if hasReviewContextFile(bundle.RelatedContextFiles, "probe/secret.go") {
		t.Fatalf("RelatedContextFiles = %#v, want ignored secret.go omitted", bundle.RelatedContextFiles)
	}
	if hasReviewSearchHit(bundle.RelatedSearchHits, "probe/secret.go") {
		t.Fatalf("RelatedSearchHits = %#v, want ignored secret.go omitted", bundle.RelatedSearchHits)
	}
	markdown := RenderReviewEvidenceMarkdown(bundle)
	for _, forbidden := range []string{"probe/secret.go", "HiddenSecret", "do-not-send"} {
		if strings.Contains(markdown, forbidden) {
			t.Fatalf("markdown contains ignored file data %q: %q", forbidden, markdown)
		}
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

func TestReviewEvidenceBuilder_CurrentChangesUsesDeletedGoPathForRelatedContextDir(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runGit(t, repo, "rm", "probe/probe.go")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if hasReviewContextFile(bundle.ChangedFileContext, "probe/probe.go") {
		t.Fatalf("ChangedFileContext = %#v, want deleted probe.go omitted", bundle.ChangedFileContext)
	}
	related := requireReviewContextFile(t, bundle.RelatedContextFiles, "probe/probe_test.go")
	if related.Role != reviewContextFileRoleRelatedTest || related.Skipped {
		t.Fatalf("related context for deleted path = %#v, want readable related test", related)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesUsesOldAndNewGoRenamePathsForRelatedContextDirs(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "oldpkg", "foo.go"), `package oldpkg

func Foo() int { return 1 }
`)
	writeTestFile(t, filepath.Join(repo, "oldpkg", "foo_test.go"), `package oldpkg

func TestOldFoo(t *testing.T) {
	_ = Foo()
}
`)
	writeTestFile(t, filepath.Join(repo, "newpkg", "foo_test.go"), `package newpkg

func TestNewFoo(t *testing.T) {
	_ = Foo()
}
`)
	runGit(t, repo, "add", "oldpkg", "newpkg")
	runGit(t, repo, "commit", "-m", "add rename fixture")
	runGit(t, repo, "mv", "oldpkg/foo.go", "newpkg/foo.go")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	oldRelated := requireReviewContextFile(t, bundle.RelatedContextFiles, "oldpkg/foo_test.go")
	if oldRelated.Role != reviewContextFileRoleRelatedTest || oldRelated.Skipped {
		t.Fatalf("old related context for renamed path = %#v, want readable related test", oldRelated)
	}
	newRelated := requireReviewContextFile(t, bundle.RelatedContextFiles, "newpkg/foo_test.go")
	if newRelated.Role != reviewContextFileRoleRelatedTest || newRelated.Skipped {
		t.Fatalf("new related context for renamed path = %#v, want readable related test", newRelated)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesUsesOldGoPathForGoToNonGoRenameRelatedContextDir(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "oldpkg", "foo.go"), `package oldpkg

func Foo() int { return 1 }
`)
	writeTestFile(t, filepath.Join(repo, "oldpkg", "foo_test.go"), `package oldpkg

func TestOldFoo(t *testing.T) {
	_ = Foo()
}
`)
	runGit(t, repo, "add", "oldpkg")
	runGit(t, repo, "commit", "-m", "add go to non-go rename fixture")
	runGit(t, repo, "mv", "oldpkg/foo.go", "oldpkg/foo.txt")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	related := requireReviewContextFile(t, bundle.RelatedContextFiles, "oldpkg/foo_test.go")
	if related.Role != reviewContextFileRoleRelatedTest || related.Skipped {
		t.Fatalf("related context for old Go path rename = %#v, want readable related test", related)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesSkipsGeneratedAndVendorOldGoPathsForRelatedContextDir(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "legacy", "service.gen.go"), `package legacy

func GeneratedService() int { return 1 }
`)
	writeTestFile(t, filepath.Join(repo, "legacy", "service_test.go"), `package legacy

func TestLegacyService() {
	_ = GeneratedService()
}
`)
	writeTestFile(t, filepath.Join(repo, "current", "service_test.go"), `package current

func TestCurrentService() {
	_ = GeneratedService()
}
`)
	runGit(t, repo, "add", "legacy", "current")
	runGit(t, repo, "commit", "-m", "add generated old path rename fixture")
	runGit(t, repo, "mv", "legacy/service.gen.go", "current/service.go")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if hasReviewContextFile(bundle.RelatedContextFiles, "legacy/service_test.go") {
		t.Fatalf("RelatedContextFiles = %#v, want generated old path directory omitted", bundle.RelatedContextFiles)
	}
	currentRelated := requireReviewContextFile(t, bundle.RelatedContextFiles, "current/service_test.go")
	if currentRelated.Role != reviewContextFileRoleRelatedTest || currentRelated.Skipped {
		t.Fatalf("current related context for renamed path = %#v, want readable related test", currentRelated)
	}

	dirs := (&reviewContextEvidenceCollector{repoRoot: repo}).changedGoFileDirs([]ReviewChangedFile{
		{Path: "notes/generated.txt", OldPath: "legacy/service.gen.go", Status: "R100"},
		{Path: "notes/vendor.txt", OldPath: "vendor/example.com/pkg/pkg.go", Status: "R100"},
	})
	if len(dirs) != 0 {
		t.Fatalf("changedGoFileDirs() = %#v, want generated/vendor old paths excluded", dirs)
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
