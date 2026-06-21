package evidence

import (
	"context"
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
