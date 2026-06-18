package review

import "time"

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
