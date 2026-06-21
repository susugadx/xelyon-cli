package evidence

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
)

func TestReviewEvidenceBuilderPostPass1ProviderBoundary(t *testing.T) {
	existing := externaldoc.WebSearchEvidence{
		Enabled:  true,
		Provider: "existing-provider",
		Queries: []externaldoc.WebSearchEvidenceQuery{{
			Query:  "existing query",
			Reason: "pre-pass",
		}},
	}
	plan := reviewprobeplan.ReviewProbePlan{
		SchemaVersion: reviewprobeplan.ReviewProbePlanSchemaVersionV2,
		Summary:       "post-pass plan",
	}

	t.Run("unsupported provider preserves existing evidence", func(t *testing.T) {
		provider := &prePassOnlyWebSearchProvider{evidence: externaldoc.WebSearchEvidence{Provider: "pre-pass-only"}}
		builder := NewReviewEvidenceBuilder("", "", WithReviewWebSearchEvidenceProvider(provider))

		got := builder.CollectPostPass1WebSearchEvidence(context.Background(), ReviewEvidenceBundle{WebSearchEvidence: existing}, plan)

		if got.Provider != existing.Provider || len(got.Queries) != 1 || got.Queries[0].Query != "existing query" {
			t.Fatalf("post-pass evidence = %#v, want existing evidence preserved", got)
		}
		if provider.prePassCalls != 0 {
			t.Fatalf("prePassCalls = %d, want 0 for post-pass collection", provider.prePassCalls)
		}
	})

	t.Run("post-pass provider receives bundle and plan", func(t *testing.T) {
		want := externaldoc.WebSearchEvidence{
			Enabled:  true,
			Provider: "post-pass-provider",
			Queries: []externaldoc.WebSearchEvidenceQuery{{
				Query:  "post-pass query",
				Reason: "plan-derived",
			}},
		}
		provider := &postPassWebSearchProvider{returnEvidence: want}
		builder := NewReviewEvidenceBuilder("", "", WithReviewWebSearchEvidenceProvider(provider))

		got := builder.CollectPostPass1WebSearchEvidence(context.Background(), ReviewEvidenceBundle{
			RepoRoot:          "/repo",
			WebSearchEvidence: existing,
		}, plan)

		if got.Provider != want.Provider || len(got.Queries) != 1 || got.Queries[0].Query != "post-pass query" {
			t.Fatalf("post-pass evidence = %#v, want provider result %#v", got, want)
		}
		if provider.postPassCalls != 1 {
			t.Fatalf("postPassCalls = %d, want 1", provider.postPassCalls)
		}
		if provider.bundle.RepoRoot != "/repo" || provider.bundle.WebSearchEvidence.Provider != existing.Provider {
			t.Fatalf("provider bundle = %#v, want original bundle with existing evidence", provider.bundle)
		}
		if provider.plan.Summary != plan.Summary {
			t.Fatalf("provider plan = %#v, want %#v", provider.plan, plan)
		}
	})
}

type prePassOnlyWebSearchProvider struct {
	evidence     externaldoc.WebSearchEvidence
	prePassCalls int
}

func (p *prePassOnlyWebSearchProvider) CollectWebSearchEvidence(context.Context, ReviewEvidenceBundle) externaldoc.WebSearchEvidence {
	p.prePassCalls++
	return p.evidence
}

type postPassWebSearchProvider struct {
	prePassOnlyWebSearchProvider
	returnEvidence externaldoc.WebSearchEvidence
	postPassCalls  int
	bundle         ReviewEvidenceBundle
	plan           reviewprobeplan.ReviewProbePlan
}

func (p *postPassWebSearchProvider) CollectPostPass1WebSearchEvidence(_ context.Context, bundle ReviewEvidenceBundle, plan reviewprobeplan.ReviewProbePlan) externaldoc.WebSearchEvidence {
	p.postPassCalls++
	p.bundle = bundle
	p.plan = plan
	return p.returnEvidence
}
