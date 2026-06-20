package evidence

import (
	"context"
	"path/filepath"
	"testing"
)

func TestReviewContextEvidenceRelatedFilesPrioritizeTestsBeforeGoWhenBudgetLimited(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "pkg", "feature.go"), "package pkg\n\nfunc Feature() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "a.go"), "package pkg\n\nfunc A() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "b.go"), "package pkg\n\nfunc B() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "z_test.go"), "package pkg\n\nfunc TestZ() {}\n")

	evidence, err := buildReviewContextEvidence(context.Background(), repo,
		[]ReviewChangedFile{{Path: "pkg/feature.go", Status: "M"}},
		[]string{"pkg/a.go", "pkg/b.go", "pkg/z_test.go"},
		ReviewEvidenceLimits{MaxContextFiles: 2},
	)
	if err != nil {
		t.Fatalf("buildReviewContextEvidence() error = %v", err)
	}

	assertReviewContextFilePaths(t, evidence.relatedContextFiles, []string{"pkg/z_test.go", "pkg/a.go"})
	first := evidence.relatedContextFiles[0]
	if first.Role != reviewContextFileRoleRelatedTest || first.Skipped {
		t.Fatalf("first related context = %#v, want readable related test", first)
	}
	limited := evidence.relatedContextFiles[1]
	if !limited.Skipped || limited.SkipReason != reviewContextSkipMaxFilesExceeded {
		t.Fatalf("budget-limited related context = %#v, want max_files_exceeded skip", limited)
	}
}

func TestReviewContextEvidenceRelatedFilesPrioritizeSameStemTest(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "pkg", "foo.go"), "package pkg\n\nfunc Foo() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "a_test.go"), "package pkg\n\nfunc TestA() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "foo_test.go"), "package pkg\n\nfunc TestFoo() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "z_test.go"), "package pkg\n\nfunc TestZ() {}\n")

	evidence, err := buildReviewContextEvidence(context.Background(), repo,
		[]ReviewChangedFile{{Path: "pkg/foo.go", Status: "M"}},
		[]string{"pkg/a_test.go", "pkg/foo_test.go", "pkg/z_test.go"},
		ReviewEvidenceLimits{MaxContextFiles: 10},
	)
	if err != nil {
		t.Fatalf("buildReviewContextEvidence() error = %v", err)
	}

	assertReviewContextFilePaths(t, evidence.relatedContextFiles, []string{
		"pkg/foo_test.go",
		"pkg/a_test.go",
		"pkg/z_test.go",
	})
}

func TestReviewContextEvidenceRelatedFilesPrioritizeSameStemGoForChangedTest(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "pkg", "foo_test.go"), "package pkg\n\nfunc TestFoo() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "a.go"), "package pkg\n\nfunc A() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "foo.go"), "package pkg\n\nfunc Foo() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "z.go"), "package pkg\n\nfunc Z() {}\n")

	evidence, err := buildReviewContextEvidence(context.Background(), repo,
		[]ReviewChangedFile{{Path: "pkg/foo_test.go", Status: "M"}},
		[]string{"pkg/a.go", "pkg/foo.go", "pkg/z.go"},
		ReviewEvidenceLimits{MaxContextFiles: 10},
	)
	if err != nil {
		t.Fatalf("buildReviewContextEvidence() error = %v", err)
	}

	assertReviewContextFilePaths(t, evidence.relatedContextFiles, []string{
		"pkg/foo.go",
		"pkg/a.go",
		"pkg/z.go",
	})
}

func TestReviewContextEvidenceRelatedFilesPrioritizeSameStemGoBeforeUnrelatedTestWhenBudgetLimited(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "pkg", "foo_test.go"), "package pkg\n\nfunc TestFoo() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "foo.go"), "package pkg\n\nfunc Foo() {}\n")
	writeTestFile(t, filepath.Join(repo, "pkg", "z_test.go"), "package pkg\n\nfunc TestZ() {}\n")

	evidence, err := buildReviewContextEvidence(context.Background(), repo,
		[]ReviewChangedFile{{Path: "pkg/foo_test.go", Status: "M"}},
		[]string{"pkg/foo.go", "pkg/z_test.go"},
		ReviewEvidenceLimits{MaxContextFiles: 2},
	)
	if err != nil {
		t.Fatalf("buildReviewContextEvidence() error = %v", err)
	}

	assertReviewContextFilePaths(t, evidence.relatedContextFiles, []string{"pkg/foo.go", "pkg/z_test.go"})
	first := evidence.relatedContextFiles[0]
	if first.Role != reviewContextFileRoleRelatedGo || first.Skipped {
		t.Fatalf("first related context = %#v, want readable same-stem implementation", first)
	}
	limited := evidence.relatedContextFiles[1]
	if !limited.Skipped || limited.SkipReason != reviewContextSkipMaxFilesExceeded {
		t.Fatalf("budget-limited related context = %#v, want max_files_exceeded skip", limited)
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
	for _, forbidden := range []string{"probe/secret.go", "HiddenSecret", "do-not-send"} {
		if reviewContextEvidenceContainsText(bundle.ChangedFileContext, forbidden) ||
			reviewContextEvidenceContainsText(bundle.RelatedContextFiles, forbidden) ||
			reviewSearchEvidenceContainsText(bundle.RelatedSearchHits, forbidden) {
			t.Fatalf("bundle contains ignored file data %q: %#v", forbidden, bundle)
		}
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

	scope := (&reviewContextEvidenceCollector{repoRoot: repo}).changedRelatedLanguageScope([]ReviewChangedFile{
		{Path: "notes/generated.txt", OldPath: "legacy/service.gen.go", Status: "R100"},
		{Path: "notes/vendor.txt", OldPath: "vendor/example.com/pkg/pkg.go", Status: "R100"},
	})
	if len(scope.stemsByDir) != 0 {
		t.Fatalf("changedRelatedLanguageScope() = %#v, want generated/vendor old paths excluded", scope.stemsByDir)
	}
}
