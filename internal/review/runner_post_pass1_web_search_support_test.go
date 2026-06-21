package review

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
)

type runnerPostPass1WebSearchEvidenceBuilder struct {
	runnerFakeEvidenceBuilder
	postEvidence externaldoc.WebSearchEvidence
	postCalls    int
	seenPlan     reviewprobeplan.ReviewProbePlan
}

func (b *runnerPostPass1WebSearchEvidenceBuilder) CollectPostPass1WebSearchEvidence(_ context.Context, _ reviewevidence.ReviewEvidenceBundle, plan reviewprobeplan.ReviewProbePlan) externaldoc.WebSearchEvidence {
	b.postCalls++
	b.seenPlan = plan
	if b.events != nil {
		*b.events = append(*b.events, "post_search")
	}
	return b.postEvidence
}

func readReviewRunArtifactForTest(t *testing.T, dir, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", name, err)
	}
	return string(content)
}

func reviewWebSearchDiscoveryCompactEvidenceForTest(snippet string, docs []externaldoc.Evidence) externaldoc.WebSearchEvidence {
	return externaldoc.WebSearchEvidence{
		Enabled:  true,
		Provider: "gemini",
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{
				Query:  "OpenAI Responses API previous_response_id official docs",
				Reason: "test",
				Results: []externaldoc.WebSearchEvidenceResult{
					{
						Title:        "OpenAI Responses API docs",
						URL:          "https://platform.openai.com/docs/responses",
						SourceDomain: "platform.openai.com",
						Snippet:      snippet,
					},
				},
			},
		},
		ExternalDocs: docs,
	}
}

func reviewExternalDocEvidenceForDiscoveryCompactTest(docID string, credibility externaldoc.SourceCredibility, truncated bool, content string) externaldoc.Evidence {
	return externaldoc.Evidence{
		DocID:                   docID,
		URL:                     "https://platform.openai.com/docs/" + docID,
		SourceDomain:            "platform.openai.com",
		SourceCredibility:       credibility,
		SourceCredibilityReason: "test credibility reason",
		FetchedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		ContentHash:             "sha256:" + strings.Repeat("a", 64-len(docID)) + docID,
		Truncated:               truncated,
		Snippets: []externaldoc.SnippetEvidence{
			{
				SnippetID:   docID + "-snippet-1",
				Content:     content,
				ContentHash: "sha256:" + strings.Repeat("b", 64-len(docID)) + docID,
				Truncated:   truncated,
			},
		},
	}
}
