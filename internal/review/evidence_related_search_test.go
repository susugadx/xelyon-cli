package review

import (
	"context"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_CurrentChangesCollectsMethodRelatedSearchHit(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "methodsearch", "type.go"), `package methodsearch

type Worker struct{}
`)
	writeTestFile(t, filepath.Join(repo, "methodsearch", "use.go"), `package methodsearch

func UseMethod(w Worker) int { return w.ReviewMethod() }
`)
	runGit(t, repo, "add", "methodsearch")
	runGit(t, repo, "commit", "-m", "add method search fixtures")

	writeTestFile(t, filepath.Join(repo, "methodsearch", "method.go"), `package methodsearch

func (w Worker) ReviewMethod() int { return 1 }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "methodsearch/use.go")
	if hit.Reason != "symbol:ReviewMethod" {
		t.Fatalf("method related search hit = %#v, want symbol:ReviewMethod", hit)
	}
	if !strings.Contains(hit.Snippet, "w.ReviewMethod()") {
		t.Fatalf("method related search hit = %#v, want method call snippet", hit)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesCollectsInitMethodRelatedSearchHit(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "initmethod", "type.go"), `package initmethod

type Worker struct{}
`)
	writeTestFile(t, filepath.Join(repo, "initmethod", "use.go"), `package initmethod

func UseMethod(w Worker) { w.init() }
`)
	runGit(t, repo, "add", "initmethod")
	runGit(t, repo, "commit", "-m", "add init method search fixtures")

	writeTestFile(t, filepath.Join(repo, "initmethod", "method.go"), `package initmethod

func (w Worker) init() {}
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "initmethod/use.go")
	if hit.Reason != "symbol:init" {
		t.Fatalf("init method related search hit = %#v, want symbol:init", hit)
	}
	if !strings.Contains(hit.Snippet, "w.init()") {
		t.Fatalf("init method related search hit = %#v, want init method call snippet", hit)
	}
}

func TestReviewRelatedSearchTermsFromDeclSuppressesOnlyPackageInitFunc(t *testing.T) {
	if terms := reviewRelatedSearchTermsFromFuncDecl(&ast.FuncDecl{Name: ast.NewIdent("init")}); len(terms) != 0 {
		t.Fatalf("package-level init terms = %#v, want none", terms)
	}

	terms := reviewRelatedSearchTermsFromFuncDecl(&ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("Worker")}}},
		Name: ast.NewIdent("init"),
	})
	assertStringSlice(t, terms, []string{"init"})

	terms = reviewRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{Name: ast.NewIdent("init")}},
	})
	assertStringSlice(t, terms, []string{"init"})

	terms = reviewRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("init")},
		}},
	})
	assertStringSlice(t, terms, []string{"init"})

	terms = reviewRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("init")},
		}},
	})
	assertStringSlice(t, terms, []string{"init"})
}

func TestReviewRelatedSearchTermsFromDeclSkipsBlankIdentifiers(t *testing.T) {
	terms := reviewRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("_"), ast.NewIdent("ReviewVarName")},
		}},
	})
	assertStringSlice(t, terms, []string{"ReviewVarName"})
}

func TestReviewEvidenceBuilder_CurrentChangesCollectsConstRelatedSearchHit(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "constsearch", "use.go"), `package constsearch

func UseConst() string { return ReviewConstName }
`)
	runGit(t, repo, "add", "constsearch")
	runGit(t, repo, "commit", "-m", "add const search fixtures")

	writeTestFile(t, filepath.Join(repo, "constsearch", "const.go"), `package constsearch

const ReviewConstName = "enabled"
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "constsearch/use.go")
	if hit.Reason != "symbol:ReviewConstName" {
		t.Fatalf("const related search hit = %#v, want symbol:ReviewConstName", hit)
	}
	if !strings.Contains(hit.Snippet, "ReviewConstName") {
		t.Fatalf("const related search hit = %#v, want const reference snippet", hit)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesCollectsVarRelatedSearchHit(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "varsearch", "use.go"), `package varsearch

func UseVar() int { return ReviewVarName }
`)
	runGit(t, repo, "add", "varsearch")
	runGit(t, repo, "commit", "-m", "add var search fixtures")

	writeTestFile(t, filepath.Join(repo, "varsearch", "var.go"), `package varsearch

var ReviewVarName = 42
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "varsearch/use.go")
	if hit.Reason != "symbol:ReviewVarName" {
		t.Fatalf("var related search hit = %#v, want symbol:ReviewVarName", hit)
	}
	if !strings.Contains(hit.Snippet, "ReviewVarName") {
		t.Fatalf("var related search hit = %#v, want var reference snippet", hit)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesBoundsRelatedSearchFiles(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "budget", "a.go"), `package budget

func FirstCandidate() SearchNeedle { return SearchNeedle{} }
`)
	writeTestFile(t, filepath.Join(repo, "budget", "budget.go"), `package budget

type SearchNeedle struct{}
`)
	writeTestFile(t, filepath.Join(repo, "budget", "z.go"), `package budget

func LaterCandidate() SearchNeedle { return SearchNeedle{} }
`)
	runGit(t, repo, "add", "budget")
	runGit(t, repo, "commit", "-m", "add search budget fixtures")

	writeTestFile(t, filepath.Join(repo, "budget", "budget.go"), `package budget

type SearchNeedle struct{}

func Changed() SearchNeedle { return SearchNeedle{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      1,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       20,
		MaxRelatedSearchTerms:      10,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if !hasReviewSearchHit(bundle.RelatedSearchHits, "budget/a.go") {
		t.Fatalf("RelatedSearchHits = %#v, want first candidate within budget", bundle.RelatedSearchHits)
	}
	if hasReviewSearchHit(bundle.RelatedSearchHits, "budget/z.go") {
		t.Fatalf("RelatedSearchHits = %#v, want later candidate beyond file budget omitted", bundle.RelatedSearchHits)
	}
	if !bundle.RelatedSearchTruncated {
		t.Fatal("RelatedSearchTruncated = false, want true when related search file budget omits candidates")
	}
}

func TestReviewEvidenceBuilder_CurrentChangesDoesNotCountMissingRelatedSearchCandidates(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "aaa_sparse", "a_missing.go"), `package aaa_sparse

func MissingReference() SearchNeedle { return SearchNeedle{} }
`)
	writeTestFile(t, filepath.Join(repo, "aaa_sparse", "seed.go"), `package aaa_sparse

type SearchNeedle struct{}
`)
	writeTestFile(t, filepath.Join(repo, "aaa_sparse", "use.go"), `package aaa_sparse

func ExistingReference() SearchNeedle { return SearchNeedle{} }
`)
	runGit(t, repo, "add", "aaa_sparse")
	runGit(t, repo, "commit", "-m", "add sparse related search fixtures")
	runGit(t, repo, "update-index", "--skip-worktree", "aaa_sparse/a_missing.go")
	if err := os.Remove(filepath.Join(repo, "aaa_sparse", "a_missing.go")); err != nil {
		t.Fatalf("Remove aaa_sparse/a_missing.go error = %v", err)
	}

	writeTestFile(t, filepath.Join(repo, "aaa_sparse", "seed.go"), `package aaa_sparse

type SearchNeedle struct{}

func Changed() SearchNeedle { return SearchNeedle{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      1,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       20,
		MaxRelatedSearchTerms:      10,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "aaa_sparse/use.go")
	if hit.Reason != "symbol:SearchNeedle" {
		t.Fatalf("related search hit = %#v, want symbol:SearchNeedle", hit)
	}
	if hasReviewSearchHit(bundle.RelatedSearchHits, "aaa_sparse/a_missing.go") {
		t.Fatalf("RelatedSearchHits = %#v, want missing candidate omitted", bundle.RelatedSearchHits)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesReportsRelatedSearchFileTruncation(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "trunc", "large.go"), `package trunc

// filler keeps the symbol reference outside the related search prefix.
// filler keeps the symbol reference outside the related search prefix.
// filler keeps the symbol reference outside the related search prefix.

func ExistingReference() SearchNeedle { return SearchNeedle{} }
`)
	writeTestFile(t, filepath.Join(repo, "trunc", "seed.go"), `package trunc

type SearchNeedle struct{}
`)
	runGit(t, repo, "add", "trunc")
	runGit(t, repo, "commit", "-m", "add related search truncation fixtures")

	writeTestFile(t, filepath.Join(repo, "trunc", "seed.go"), `package trunc

type SearchNeedle struct{}

func Changed() SearchNeedle { return SearchNeedle{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      10,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  32,
		MaxRelatedSearchHits:       20,
		MaxRelatedSearchTerms:      10,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if !bundle.RelatedSearchTruncated {
		t.Fatal("RelatedSearchTruncated = false, want true when related search reads only a file prefix")
	}
	if hasReviewSearchHit(bundle.RelatedSearchHits, "trunc/large.go") {
		t.Fatalf("RelatedSearchHits = %#v, want reference after prefix omitted", bundle.RelatedSearchHits)
	}

	input := BuildReviewEvidenceModelInput(bundle)
	if !input.TruncationFlags.RelatedSearch {
		t.Fatal("TruncationFlags.RelatedSearch = false, want true")
	}
	markdown := RenderReviewEvidenceMarkdown(bundle)
	if !strings.Contains(markdown, `"related_search": true`) {
		t.Fatalf("markdown = %q, want related search truncation flag", markdown)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesReportsRelatedSearchHitCapTruncation(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "hitcap", "seed.go"), `package hitcap

type SearchNeedle struct{}
`)
	writeTestFile(t, filepath.Join(repo, "hitcap", "use.go"), `package hitcap

func UseOne() SearchNeedle { return SearchNeedle{} }

func UseTwo() SearchNeedle { return SearchNeedle{} }
`)
	runGit(t, repo, "add", "hitcap")
	runGit(t, repo, "commit", "-m", "add related search hit cap fixtures")

	writeTestFile(t, filepath.Join(repo, "hitcap", "seed.go"), `package hitcap

type SearchNeedle struct{}

func Changed() SearchNeedle { return SearchNeedle{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      10,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       1,
		MaxRelatedSearchTerms:      10,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if got := len(bundle.RelatedSearchHits); got != 1 {
		t.Fatalf("RelatedSearchHits length = %d, want capped single hit: %#v", got, bundle.RelatedSearchHits)
	}
	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "hitcap/use.go")
	if !strings.Contains(hit.Snippet, "UseOne") {
		t.Fatalf("related search hit = %#v, want first reference before cap", hit)
	}
	if !bundle.RelatedSearchTruncated {
		t.Fatal("RelatedSearchTruncated = false, want true when hit cap omits later matches")
	}

	input := BuildReviewEvidenceModelInput(bundle)
	if !input.TruncationFlags.RelatedSearch {
		t.Fatal("TruncationFlags.RelatedSearch = false, want true")
	}
	markdown := RenderReviewEvidenceMarkdown(bundle)
	if !strings.Contains(markdown, `"related_search": true`) {
		t.Fatalf("markdown = %q, want related search truncation flag", markdown)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesReportsRelatedSearchTermCapTruncation(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "termcap", "seed.go"), `package termcap

type FirstSymbol struct{}
type OmittedSymbol struct{}
`)
	writeTestFile(t, filepath.Join(repo, "termcap", "use.go"), `package termcap

func UseOmitted() OmittedSymbol { return OmittedSymbol{} }
`)
	runGit(t, repo, "add", "termcap")
	runGit(t, repo, "commit", "-m", "add related search term cap fixtures")

	writeTestFile(t, filepath.Join(repo, "termcap", "seed.go"), `package termcap

type FirstSymbol struct{}
type OmittedSymbol struct{}

func Changed() FirstSymbol { return FirstSymbol{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      10,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       20,
		MaxRelatedSearchTerms:      1,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if hasReviewSearchHit(bundle.RelatedSearchHits, "termcap/use.go") {
		t.Fatalf("RelatedSearchHits = %#v, want omitted term reference absent", bundle.RelatedSearchHits)
	}
	if !bundle.RelatedSearchTruncated {
		t.Fatal("RelatedSearchTruncated = false, want true when term cap omits search terms")
	}

	input := BuildReviewEvidenceModelInput(bundle)
	if !input.TruncationFlags.RelatedSearch {
		t.Fatal("TruncationFlags.RelatedSearch = false, want true")
	}
	markdown := RenderReviewEvidenceMarkdown(bundle)
	if !strings.Contains(markdown, `"related_search": true`) {
		t.Fatalf("markdown = %q, want related search truncation flag", markdown)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesReportsRelatedSearchTermCapWithMethodConstVarTerms(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "methodtermcap", "type.go"), `package methodtermcap

type CapWorker struct{}
`)
	writeTestFile(t, filepath.Join(repo, "methodtermcap", "use.go"), `package methodtermcap

func UseMethod(w CapWorker) int { return w.CapMethod() }

func UseConst() string { return CapConst }

func UseVar() CapWorker { return CapVar }
`)
	runGit(t, repo, "add", "methodtermcap")
	runGit(t, repo, "commit", "-m", "add method const var term cap fixtures")

	writeTestFile(t, filepath.Join(repo, "methodtermcap", "seed.go"), `package methodtermcap

func (w CapWorker) CapMethod() int { return 1 }

const CapConst = "cap"

var CapVar = CapWorker{}
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      10,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       20,
		MaxRelatedSearchTerms:      2,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if !hasReviewSearchHitWithReason(bundle.RelatedSearchHits, "methodtermcap/use.go", "symbol:CapMethod") {
		t.Fatalf("RelatedSearchHits = %#v, want in-cap method term hit", bundle.RelatedSearchHits)
	}
	if !hasReviewSearchHitWithReason(bundle.RelatedSearchHits, "methodtermcap/use.go", "symbol:CapConst") {
		t.Fatalf("RelatedSearchHits = %#v, want in-cap const term hit", bundle.RelatedSearchHits)
	}
	if hasReviewSearchHitWithReason(bundle.RelatedSearchHits, "methodtermcap/use.go", "symbol:CapVar") {
		t.Fatalf("RelatedSearchHits = %#v, want cap-excluded var term hit omitted", bundle.RelatedSearchHits)
	}
	if !bundle.RelatedSearchTruncated {
		t.Fatal("RelatedSearchTruncated = false, want true when method/const/var terms exceed term cap")
	}
}

func TestReviewEvidenceBuilder_CurrentChangesReportsRelatedSearchSnippetTruncation(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "snippet", "seed.go"), `package snippet

type SnippetNeedle struct{}
`)
	writeTestFile(t, filepath.Join(repo, "snippet", "use.go"), `package snippet

func UseSnippet() SnippetNeedle { return SnippetNeedle{} } // suffix that should be truncated
`)
	runGit(t, repo, "add", "snippet")
	runGit(t, repo, "commit", "-m", "add related search snippet truncation fixtures")

	writeTestFile(t, filepath.Join(repo, "snippet", "seed.go"), `package snippet

type SnippetNeedle struct{}

func Changed() SnippetNeedle { return SnippetNeedle{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      10,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       20,
		MaxRelatedSearchTerms:      10,
		MaxSearchSnippetBytes:      24,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "snippet/use.go")
	if got := int64(len(hit.Snippet)); got != 24 {
		t.Fatalf("snippet length = %d, want truncated 24-byte snippet: %#v", got, hit)
	}
	if !bundle.RelatedSearchTruncated {
		t.Fatal("RelatedSearchTruncated = false, want true when search snippet is truncated")
	}

	input := BuildReviewEvidenceModelInput(bundle)
	if !input.TruncationFlags.RelatedSearch {
		t.Fatal("TruncationFlags.RelatedSearch = false, want true")
	}
	markdown := RenderReviewEvidenceMarkdown(bundle)
	if !strings.Contains(markdown, `"related_search": true`) {
		t.Fatalf("markdown = %q, want related search truncation flag", markdown)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesPrioritizesSymbolSearchHitsOverPackageDeclarations(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "dom", "a.go"), `package dom

func A() {}
`)
	writeTestFile(t, filepath.Join(repo, "dom", "b.go"), `package dom

func B() {}
`)
	writeTestFile(t, filepath.Join(repo, "dom", "dom.go"), `package dom

type SearchNeedle struct{}
`)
	writeTestFile(t, filepath.Join(repo, "dom", "use.go"), `package dom

func UseNeedle() SearchNeedle { return SearchNeedle{} }
`)
	runGit(t, repo, "add", "dom")
	runGit(t, repo, "commit", "-m", "add related search priority fixtures")

	writeTestFile(t, filepath.Join(repo, "dom", "dom.go"), `package dom

type SearchNeedle struct{}

func Changed() SearchNeedle { return SearchNeedle{} }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      10,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       2,
		MaxRelatedSearchTerms:      10,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	symbolHit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "dom/use.go")
	if symbolHit.Reason != "symbol:SearchNeedle" {
		t.Fatalf("symbol hit = %#v, want symbol:SearchNeedle", symbolHit)
	}
	for _, hit := range bundle.RelatedSearchHits {
		if strings.TrimSpace(hit.Snippet) == "package dom" {
			t.Fatalf("RelatedSearchHits = %#v, want package declarations filtered from ranked hits", bundle.RelatedSearchHits)
		}
	}
}

func TestReviewEvidenceBuilder_CurrentChangesFiltersPackageDeclarationsForSymbolNameCollisions(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "cmdapp", "a.go"), `package main

func A() {}
`)
	writeTestFile(t, filepath.Join(repo, "cmdapp", "b.go"), `package main

func B() {}
`)
	writeTestFile(t, filepath.Join(repo, "cmdapp", "main.go"), `package main

func main() {}
`)
	writeTestFile(t, filepath.Join(repo, "cmdapp", "use.go"), `package main

func callMain() { main() }
`)
	runGit(t, repo, "add", "cmdapp")
	runGit(t, repo, "commit", "-m", "add package name collision fixtures")

	writeTestFile(t, filepath.Join(repo, "cmdapp", "main.go"), `package main

func main() {}

func changedMain() { main() }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxContextFileBytes:        1024,
		MaxTotalContextBytes:       4096,
		MaxContextFiles:            10,
		MaxRelatedSearchFiles:      10,
		MaxTotalRelatedSearchBytes: 4096,
		MaxRelatedSearchFileBytes:  1024,
		MaxRelatedSearchHits:       1,
		MaxRelatedSearchTerms:      10,
		MaxSearchSnippetBytes:      120,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	hit := requireReviewSearchHit(t, bundle.RelatedSearchHits, "cmdapp/use.go")
	if hit.Reason != "symbol:main" {
		t.Fatalf("main symbol hit = %#v, want symbol:main", hit)
	}
	if strings.TrimSpace(hit.Snippet) == "package main" {
		t.Fatalf("main symbol hit = %#v, want non-package declaration hit", hit)
	}
}
