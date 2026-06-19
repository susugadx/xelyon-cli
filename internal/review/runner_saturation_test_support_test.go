package review

import (
	"testing"
	"time"
)

func saturatedRunnerModelResponseForTest(t *testing.T) runnerFakeModelResponse {
	t.Helper()

	return runnerFakeModelResponse{
		content: string(mustMarshalReviewSaturationCheckForTest(t, newSaturatedReviewSaturationCheckForTest())),
	}
}

func needsRevisionMissingRiskCheckForRunnerTest() ReviewSaturationCheck {
	return ReviewSaturationCheck{
		SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
		Status:               ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "risk-1 was not fully represented in the finalized report.",
		MissingRiskIDs:       []string{"risk-1"},
		RevisionInstructions: "Revise the report so risk-1 is explicitly classified in scope_coverage.",
	}
}

func needsRevisionAdditionalCandidateCheckForRunnerTest() ReviewSaturationCheck {
	return ReviewSaturationCheck{
		SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
		Status:         ReviewSaturationStatusNeedsRevision,
		CheckedSummary: "A file-backed candidate was not represented in the finalized report.",
		AdditionalFindingCandidates: []ReviewSaturationAdditionalFindingCandidate{
			{
				Summary: "A report-pass finding candidate is grounded in existing file evidence.",
				EvidenceRefs: []ReviewEvidenceRef{
					newFileEvidenceRefForValidationTest(),
				},
				Reason: "The candidate uses existing evidence only and does not require additional exploration.",
			},
		},
		RevisionInstructions: "Revise the report to include or explicitly dismiss the file-backed candidate.",
	}
}

func newRunnerEvidenceBundleWithWebSearchForSaturationAuditTest(repoRoot string) ReviewEvidenceBundle {
	bundle := newRunnerEvidenceBundleForTest(repoRoot)
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:      true,
		Inconclusive: true,
	}
	return bundle
}

func newRunnerEvidenceBundleWithAdequateWebSearchSupportForSaturationAuditTest(repoRoot string) ReviewEvidenceBundle {
	bundle := newRunnerEvidenceBundleForTest(repoRoot)
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled: true,
		ExternalDocs: []ReviewExternalDocEvidence{
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

func newRunnerOfficialCandidateExternalDocForSaturationAuditTest(docID, url, content, contentHash string) ReviewExternalDocEvidence {
	return ReviewExternalDocEvidence{
		DocID:             docID,
		URL:               url,
		SourceCredibility: ReviewExternalDocSourceCredibilityOfficialCandidate,
		FetchedAt:         time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		ContentHash:       contentHash,
		Snippets: []ReviewExternalDocSnippetEvidence{
			{
				SnippetID:   docID + "-snippet-1",
				Content:     content,
				ContentHash: contentHash,
			},
		},
	}
}

func newExternalDocEvidenceRefForSaturationCompactTest(doc ReviewExternalDocEvidence) ReviewEvidenceRef {
	snippet := doc.Snippets[0]
	return ReviewEvidenceRef{
		Kind:        ReviewEvidenceKindExternalDoc,
		DocID:       doc.DocID,
		SnippetID:   snippet.SnippetID,
		URL:         doc.URL,
		FetchedAt:   doc.FetchedAt.Format(time.RFC3339Nano),
		ContentHash: snippet.ContentHash,
	}
}

func findReviewPromptReductionItemForTest(runner *ReviewRunner, id string, phase ReviewModelPhase) *ReviewPromptReductionItem {
	if runner == nil || runner.promptReductionState == nil {
		return nil
	}
	for i := range runner.promptReductionState.Items {
		if runner.promptReductionState.Items[i].ID == id && runner.promptReductionState.Items[i].Phase == phase {
			return &runner.promptReductionState.Items[i]
		}
	}
	return nil
}

func newRunnerPostPass1WeakExternalEvidenceForSaturationAuditTest() ReviewWebSearchEvidence {
	return ReviewWebSearchEvidence{
		Enabled: true,
		Queries: []ReviewWebSearchEvidenceQuery{
			{
				Query:  "OAuth 2.0 redirect URI specification",
				Reason: "intent=spec; expected_source_type=technical_specification; confidence=high; reason=pass1 plan protocol/spec signal",
				Results: []ReviewWebSearchEvidenceResult{
					{Title: "OAuth 2.0 redirect URI specification", URL: "https://docs.example.test/oauth"},
				},
			},
		},
		ExternalDocs: []ReviewExternalDocEvidence{
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
		},
	}
}
