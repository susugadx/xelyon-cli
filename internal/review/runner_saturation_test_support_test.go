package review

import (
	"testing"
	"time"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func saturatedRunnerModelResponseForTest(t *testing.T) runnerFakeModelResponse {
	t.Helper()

	return runnerFakeModelResponse{
		content: string(mustMarshalReviewSaturationCheckForTest(t, newSaturatedReviewSaturationCheckForTest())),
	}
}

func needsRevisionMissingRiskCheckForRunnerTest() reviewreport.ReviewSaturationCheck {
	return reviewreport.ReviewSaturationCheck{
		SchemaVersion:        reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:               reviewreport.ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "risk-1 was not fully represented in the finalized report.",
		MissingRiskIDs:       []string{"risk-1"},
		RevisionInstructions: "Revise the report so risk-1 is explicitly classified in scope_coverage.",
	}
}

func needsRevisionAdditionalCandidateCheckForRunnerTest() reviewreport.ReviewSaturationCheck {
	return reviewreport.ReviewSaturationCheck{
		SchemaVersion:  reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:         reviewreport.ReviewSaturationStatusNeedsRevision,
		CheckedSummary: "A file-backed candidate was not represented in the finalized report.",
		AdditionalFindingCandidates: []reviewreport.ReviewSaturationAdditionalFindingCandidate{
			{
				Summary: "A report-pass finding candidate is grounded in existing file evidence.",
				EvidenceRefs: []reviewreport.ReviewEvidenceRef{
					newFileEvidenceRefForValidationTest(),
				},
				Reason: "The candidate uses existing evidence only and does not require additional exploration.",
			},
		},
		RevisionInstructions: "Revise the report to include or explicitly dismiss the file-backed candidate.",
	}
}

func newRunnerEvidenceBundleWithWebSearchForSaturationAuditTest(repoRoot string) reviewevidence.ReviewEvidenceBundle {
	bundle := newRunnerEvidenceBundleForTest(repoRoot)
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled:      true,
		Inconclusive: true,
	}
	return bundle
}

func newRunnerEvidenceBundleWithAdequateWebSearchSupportForSaturationAuditTest(repoRoot string) reviewevidence.ReviewEvidenceBundle {
	bundle := newRunnerEvidenceBundleForTest(repoRoot)
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		ExternalDocs: []externaldoc.Evidence{
			newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
				"external-doc-official-1",
				"https://docs.example.test/oauth",
				"OAuth redirect URI official behavior.",
				"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			),
			newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
				"external-doc-official-2",
				"https://reference.example.test/oauth",
				"OAuth redirect URI reference behavior.",
				"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			),
		},
	}
	return bundle
}

func newRunnerOfficialCandidateExternalDocForSaturationAuditTest(docID, url, content, contentHash string) externaldoc.Evidence {
	return externaldoc.Evidence{
		DocID:             docID,
		URL:               url,
		SourceCredibility: externaldoc.SourceCredibilityOfficialCandidate,
		FetchedAt:         time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		ContentHash:       contentHash,
		Snippets: []externaldoc.SnippetEvidence{
			{
				SnippetID:   docID + "-snippet-1",
				Content:     content,
				ContentHash: contentHash,
			},
		},
	}
}

func newExternalDocEvidenceRefForSaturationCompactTest(doc externaldoc.Evidence) reviewreport.ReviewEvidenceRef {
	snippet := doc.Snippets[0]
	return reviewreport.ReviewEvidenceRef{
		Kind:        reviewreport.ReviewEvidenceKindExternalDoc,
		DocID:       doc.DocID,
		SnippetID:   snippet.SnippetID,
		URL:         doc.URL,
		FetchedAt:   doc.FetchedAt.Format(time.RFC3339Nano),
		ContentHash: snippet.ContentHash,
	}
}

func findReviewPromptReductionItemForTest(runner *ReviewRunner, id string, phase ReviewModelPhase) *reviewpromptreduction.ReviewPromptReductionItem {
	if runner == nil || runner.promptReductionState == nil {
		return nil
	}
	reductionPhase := reviewPromptReductionPhase(phase)
	for i := range runner.promptReductionState.Items {
		if runner.promptReductionState.Items[i].ID == id && runner.promptReductionState.Items[i].Phase == reductionPhase {
			return &runner.promptReductionState.Items[i]
		}
	}
	return nil
}

func newRunnerPostPass1WeakExternalEvidenceForSaturationAuditTest() externaldoc.WebSearchEvidence {
	return externaldoc.WebSearchEvidence{
		Enabled: true,
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{
				Query:  "OAuth 2.0 redirect URI specification",
				Reason: "intent=spec; expected_source_type=technical_specification; confidence=high; reason=pass1 plan protocol/spec signal",
				Results: []externaldoc.WebSearchEvidenceResult{
					{Title: "OAuth 2.0 redirect URI specification", URL: "https://docs.example.test/oauth"},
				},
			},
		},
		ExternalDocs: []externaldoc.Evidence{
			{
				DocID:             "external-doc-post",
				URL:               "https://docs.example.test/oauth",
				SourceCredibility: externaldoc.SourceCredibilityUnknown,
				Snippets: []externaldoc.SnippetEvidence{
					{
						SnippetID:   "external-doc-post-snippet-1",
						Content:     "Post-pass1 OAuth redirect URI snippet.",
						ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
					},
				},
			},
		},
	}
}
