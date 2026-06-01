package review

import (
	"context"
	"strings"
	"testing"
)

func TestReviewRunnerRunUsesPostPass1WebSearchEvidenceForReport(t *testing.T) {
	events := []string{}
	initialBundle := newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")
	initialBundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled: true,
		Queries: []ReviewWebSearchEvidenceQuery{
			{Query: "OpenAI API web_search official documentation", Reason: "pre-pass1"},
		},
		Inconclusive: true,
	}
	mergedEvidence := initialBundle.WebSearchEvidence
	mergedEvidence.Queries = append(mergedEvidence.Queries, ReviewWebSearchEvidenceQuery{
		Query:  "OAuth 2.0 redirect URI specification",
		Reason: "intent=spec; expected_source_type=technical_specification; confidence=high; reason=pass1 plan protocol/spec signal",
	})
	mergedEvidence.ExternalDocs = []ReviewExternalDocEvidence{
		{
			DocID:             "external-doc-post",
			URL:               "https://docs.example.test/oauth",
			SourceCredibility: ReviewExternalDocSourceCredibilityUnknown,
			Snippets: []ReviewExternalDocSnippetEvidence{
				{
					SnippetID:   "external-doc-post-snippet-1",
					Content:     "Post-pass1 OAuth redirect URI snippet.",
					ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
		},
	}
	mergedEvidence.Inconclusive = false
	evidence := &runnerPostPass1WebSearchEvidenceBuilder{
		runnerFakeEvidenceBuilder: runnerFakeEvidenceBuilder{
			bundle: initialBundle,
			events: &events,
		},
		postEvidence: mergedEvidence,
	}
	plan := newRunnerNoProbePlanForTest()
	report := newRunnerCleanReportForTest(nil)
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
		events: &events,
	}
	runner := newReviewRunnerForTest(t, evidence, &runnerFakeProbeRunner{events: &events}, model)

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	assertStringSliceEqualForRunnerTest(t, events, []string{
		"evidence",
		"model:probe_plan",
		"post_search",
		"model:report",
		"model:saturation_check",
	})
	if evidence.postCalls != 1 {
		t.Fatalf("postCalls = %d, want 1", evidence.postCalls)
	}
	if evidence.seenPlan.SchemaVersion != ReviewProbePlanSchemaVersionV2 {
		t.Fatalf("seen plan schema = %q, want %q", evidence.seenPlan.SchemaVersion, ReviewProbePlanSchemaVersionV2)
	}
	if strings.Contains(model.requests[0].Prompt, "external-doc-post") {
		t.Fatalf("Pass1 prompt contains post-pass1 evidence:\n%s", model.requests[0].Prompt)
	}
	if !strings.Contains(model.requests[1].Prompt, "external-doc-post") || !strings.Contains(model.requests[1].Prompt, "OAuth 2.0 redirect URI specification") {
		t.Fatalf("Pass2 prompt missing merged post-pass1 evidence:\n%s", model.requests[1].Prompt)
	}
}

type runnerPostPass1WebSearchEvidenceBuilder struct {
	runnerFakeEvidenceBuilder
	postEvidence ReviewWebSearchEvidence
	postCalls    int
	seenPlan     ReviewProbePlan
}

func (b *runnerPostPass1WebSearchEvidenceBuilder) CollectPostPass1WebSearchEvidence(_ context.Context, _ ReviewEvidenceBundle, plan ReviewProbePlan) ReviewWebSearchEvidence {
	b.postCalls++
	b.seenPlan = plan
	if b.events != nil {
		*b.events = append(*b.events, "post_search")
	}
	return b.postEvidence
}
