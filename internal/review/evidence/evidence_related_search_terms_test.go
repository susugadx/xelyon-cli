package evidence

import (
	"context"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewRelatedSearchTermsFromDeclSuppressesOnlyPackageInitFunc(t *testing.T) {
	if terms := reviewGoRelatedSearchTermsFromFuncDecl(&ast.FuncDecl{Name: ast.NewIdent("init")}); len(terms) != 0 {
		t.Fatalf("package-level init terms = %#v, want none", terms)
	}

	terms := reviewGoRelatedSearchTermsFromFuncDecl(&ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("Worker")}}},
		Name: ast.NewIdent("init"),
	})
	assertStringSlice(t, terms, []string{"init"})

	terms = reviewGoRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{Name: ast.NewIdent("init")}},
	})
	assertStringSlice(t, terms, []string{"init"})

	terms = reviewGoRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("init")},
		}},
	})
	assertStringSlice(t, terms, []string{"init"})

	terms = reviewGoRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("init")},
		}},
	})
	assertStringSlice(t, terms, []string{"init"})
}

func TestReviewRelatedSearchTermsFromDeclSkipsBlankIdentifiers(t *testing.T) {
	terms := reviewGoRelatedSearchTermsFromDecl(&ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("_"), ast.NewIdent("ReviewVarName")},
		}},
	})
	assertStringSlice(t, terms, []string{"ReviewVarName"})
}

func TestBuildReviewRelatedSearchTermsPreservesGoTestFileStem(t *testing.T) {
	termSet := buildReviewRelatedSearchTerms([]ReviewContextFileEvidence{{
		Path:    "pkg/foo_test.go",
		Content: "package pkg\n\nfunc TestFoo(t *testing.T) {}\n",
	}}, ReviewEvidenceLimits{MaxRelatedSearchTerms: 10})

	assertReviewRelatedSearchTerm(t, termSet.items, "foo_test", "file_stem:foo_test")
	if hasReviewRelatedSearchTerm(termSet.items, "foo", "file_stem:foo") {
		t.Fatalf("related search terms = %#v, want no broad file_stem:foo for changed Go test file", termSet.items)
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

func assertReviewRelatedSearchTerm(t *testing.T, terms []reviewRelatedSearchTerm, term, reason string) {
	t.Helper()
	if !hasReviewRelatedSearchTerm(terms, term, reason) {
		t.Fatalf("related search terms = %#v, want %s %s", terms, term, reason)
	}
}

func hasReviewRelatedSearchTerm(terms []reviewRelatedSearchTerm, term, reason string) bool {
	for _, item := range terms {
		if item.term == term && item.reason == reason {
			return true
		}
	}
	return false
}
