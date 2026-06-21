package modelinput

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
	"time"

	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func TestReviewEvidenceMarkdownIncludesWebSearchEvidenceSection(t *testing.T) {
	bundle := newReviewWebSearchEvidenceModelInputTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled:  true,
		Provider: "gemini",
		Queries:  []externaldoc.WebSearchEvidenceQuery{{Query: "OpenAI API web_search official documentation", Reason: "test"}},
		ExternalDocs: []externaldoc.Evidence{
			{
				DocID:                   "external-doc-1",
				URL:                     "https://example.test/openai-reference",
				SourceDomain:            "example.test",
				SourceCredibility:       externaldoc.SourceCredibilityUnknown,
				SourceCredibilityReason: "unknown: source domain does not match trusted domains for the query subject",
				Snippets: []externaldoc.SnippetEvidence{
					{
						SnippetID:   "external-doc-1-snippet-1",
						Content:     "OpenAI API request example.",
						ContentHash: "hash-1",
					},
				},
			},
		},
	}

	got := RenderReviewEvidenceMarkdown(bundle)

	for _, want := range []string{
		"## review web search evidence",
		`"provider": "gemini"`,
		`"OpenAI API web_search official documentation"`,
		`"source_credibility": "unknown"`,
		`"source_credibility_reason": "unknown: source domain does not match trusted domains for the query subject"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}

func TestBuildReviewEvidenceModelInputIncludesExternalSupportSummary(t *testing.T) {
	bundle := newReviewWebSearchEvidenceModelInputTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		ExternalDocs: []externaldoc.Evidence{
			newReviewExternalSupportDocForModelInputTest("external-doc-1", externaldoc.SourceCredibilityOfficialCandidate, "first official snippet"),
			newReviewExternalSupportDocForModelInputTest("external-doc-2", externaldoc.SourceCredibilityOfficialCandidate, "second official snippet"),
		},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if input.ExternalSupport.Level != externaldoc.ExternalSupportLevelAdequate {
		t.Fatalf("ExternalSupport.Level = %q, want adequate", input.ExternalSupport.Level)
	}
	if !input.ExternalSupport.OfficialConfirmation {
		t.Fatal("ExternalSupport.OfficialConfirmation = false, want true")
	}
	if input.ExternalSupport.OfficialCandidateCitationCapableDocCount != 2 {
		t.Fatalf("OfficialCandidateCitationCapableDocCount = %d, want 2", input.ExternalSupport.OfficialCandidateCitationCapableDocCount)
	}
	if input.ExternalSupport.OfficialCandidateUniqueCitationCapableSourceCount != 2 {
		t.Fatalf("OfficialCandidateUniqueCitationCapableSourceCount = %d, want 2", input.ExternalSupport.OfficialCandidateUniqueCitationCapableSourceCount)
	}
}

func TestReviewEvidenceMarkdownIncludesExternalSupportSummary(t *testing.T) {
	bundle := newReviewWebSearchEvidenceModelInputTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		ExternalDocs: []externaldoc.Evidence{
			newReviewExternalSupportDocForModelInputTest("external-doc-1", externaldoc.SourceCredibilityOfficialCandidate, "first official snippet"),
			newReviewExternalSupportDocForModelInputTest("external-doc-2", externaldoc.SourceCredibilityOfficialCandidate, "second official snippet"),
		},
	}

	got := RenderReviewEvidenceMarkdown(bundle)

	for _, want := range []string{
		"## external support summary",
		`"level": "adequate"`,
		`"citation_capable_doc_count": 2`,
		`"official_candidate_citation_capable_doc_count": 2`,
		`"official_candidate_unique_citation_capable_source_count": 2`,
		`"official_confirmation": true`,
		"## review web search evidence",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}

func TestExternalSupportSummaryDoesNotMutateWebSearchEvidence(t *testing.T) {
	bundle := newReviewWebSearchEvidenceModelInputTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{
				Query: "OpenAI API web_search official documentation",
				Results: []externaldoc.WebSearchEvidenceResult{
					{Title: "OpenAI docs", URL: "https://platform.openai.com/docs"},
				},
			},
		},
		ExternalDocs: []externaldoc.Evidence{
			newReviewExternalSupportDocForModelInputTest("external-doc-1", externaldoc.SourceCredibilityOfficialCandidate, "first official snippet"),
		},
	}
	original := cloneReviewWebSearchEvidenceForModelInputTest(bundle.WebSearchEvidence)

	_ = BuildReviewEvidenceModelInput(bundle)
	_ = RenderReviewEvidenceMarkdown(bundle)

	if !reflect.DeepEqual(bundle.WebSearchEvidence, original) {
		t.Fatalf("WebSearchEvidence mutated:\n got %#v\nwant %#v", bundle.WebSearchEvidence, original)
	}
}

func TestReviewPressureSignalsIncludeWebSearchEvidenceStates(t *testing.T) {
	input := BuildReviewEvidenceModelInput(newReviewWebSearchEvidenceModelInputTestBundle())
	disabledSignals := BuildReviewPressureSignalInputs(input)
	if !reviewPressureSignalsContain(disabledSignals, "web_search_evidence_disabled_for_external_contract_change") {
		t.Fatalf("signals = %#v, want disabled external contract signal", disabledSignals)
	}

	bundle := newReviewWebSearchEvidenceModelInputTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled:      true,
		Error:        "fetch failed",
		Truncated:    true,
		Inconclusive: true,
	}
	enabledSignals := BuildReviewPressureSignalInputs(BuildReviewEvidenceModelInput(bundle))
	for _, signal := range []string{
		"web_search_evidence_failed",
		"web_search_evidence_truncated",
		"web_search_evidence_inconclusive",
	} {
		if !reviewPressureSignalsContain(enabledSignals, signal) {
			t.Fatalf("signals = %#v, want %s", enabledSignals, signal)
		}
	}
}

func newReviewWebSearchEvidenceModelInputTestBundle() reviewevidence.ReviewEvidenceBundle {
	return reviewevidence.ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
		RepoRoot:   "/tmp/repo",
		CWD:        "/tmp/repo",
		ChangedFiles: []reviewevidence.ReviewChangedFile{
			{Path: "internal/api/providers/openai/web_search.go", Status: "M", Unstaged: true},
		},
		Diffs: []reviewevidence.ReviewDiffEvidence{
			{
				Source: "unstaged",
				Stat:   "internal/api/providers/openai/web_search.go | 2 +",
				Diff:   "+ Tools: []map[string]any{{\"type\":\"web_search\"}}",
			},
		},
		Inventory: reviewevidence.ReviewChangeInventory{
			Production: []string{"internal/api/providers/openai/web_search.go"},
		},
		GenericImpactCandidates: reviewevidence.ReviewGenericImpactCandidates{
			Tokens: []string{"web_search"},
		},
		Limits: reviewevidence.DefaultReviewEvidenceLimits(),
	}
}

func newReviewExternalSupportDocForModelInputTest(docID string, credibility externaldoc.SourceCredibility, content string) externaldoc.Evidence {
	return externaldoc.Evidence{
		DocID:             docID,
		URL:               "https://platform.openai.com/docs/" + docID,
		SourceDomain:      "platform.openai.com",
		SourceCredibility: credibility,
		FetchedAt:         time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		ContentHash:       reviewExternalSupportHashForModelInputTest("doc:" + docID),
		Snippets: []externaldoc.SnippetEvidence{
			{
				SnippetID:   docID + "-snippet-1",
				Content:     content,
				ContentHash: reviewExternalSupportHashForModelInputTest("snippet:" + content),
			},
		},
	}
}

func reviewExternalSupportHashForModelInputTest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneReviewWebSearchEvidenceForModelInputTest(evidence externaldoc.WebSearchEvidence) externaldoc.WebSearchEvidence {
	clone := evidence
	clone.Queries = append([]externaldoc.WebSearchEvidenceQuery(nil), evidence.Queries...)
	for i := range clone.Queries {
		clone.Queries[i].Results = append([]externaldoc.WebSearchEvidenceResult(nil), evidence.Queries[i].Results...)
	}
	clone.ExternalDocs = append([]externaldoc.Evidence(nil), evidence.ExternalDocs...)
	for i := range clone.ExternalDocs {
		clone.ExternalDocs[i].Snippets = append([]externaldoc.SnippetEvidence(nil), evidence.ExternalDocs[i].Snippets...)
	}
	return clone
}

func reviewPressureSignalsContain(signals []reviewanalysis.PressureSignal, want string) bool {
	for _, signal := range signals {
		if signal.Signal == want {
			return true
		}
	}
	return false
}
