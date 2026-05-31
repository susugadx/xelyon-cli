package review

import (
	"fmt"
	"time"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

type reviewExternalDocSnippetRefTarget struct {
	url         string
	fetchedAt   time.Time
	contentHash string
}

func validateReviewProbePlanExternalDocRefsAgainstEvidence(plan ReviewProbePlan, bundle ReviewEvidenceBundle) error {
	for i, surface := range plan.ImpactSurfaces {
		if err := validateExternalDocEvidenceRefsAgainstBundle(fmt.Sprintf("impact_surfaces[%d].evidence_refs", i), surface.EvidenceRefs, bundle); err != nil {
			return err
		}
	}
	for i, risk := range plan.CandidateRisks {
		if err := validateExternalDocEvidenceRefsAgainstBundle(fmt.Sprintf("candidate_risks[%d].evidence_refs", i), risk.EvidenceRefs, bundle); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewReportExternalDocRefsAgainstEvidence(report ReviewReport, bundle ReviewEvidenceBundle) error {
	return reviewreport.WalkReviewReportEvidenceRefs(report, func(field string, refs []ReviewEvidenceRef) error {
		return validateExternalDocEvidenceRefsAgainstBundle(field, refs, bundle)
	})
}

func validateReviewSaturationExternalDocRefsAgainstEvidence(check ReviewSaturationCheck, bundle ReviewEvidenceBundle) error {
	return reviewreport.WalkReviewSaturationEvidenceRefs(check, func(field string, refs []ReviewEvidenceRef) error {
		return validateExternalDocEvidenceRefsAgainstBundle(field, refs, bundle)
	})
}

func validateExternalDocEvidenceRefsAgainstBundle(field string, refs []ReviewEvidenceRef, bundle ReviewEvidenceBundle) error {
	index := indexReviewExternalDocSnippetRefs(bundle.WebSearchEvidence.ExternalDocs)
	for i, ref := range refs {
		if ref.Kind != ReviewEvidenceKindExternalDoc {
			continue
		}
		refField := fmt.Sprintf("%s[%d]", field, i)
		key := reviewExternalDocSnippetRefKey(ref.DocID, ref.SnippetID)
		target, exists := index[key]
		if !exists {
			return fmt.Errorf("%s references unknown fetched external_doc snippet %q/%q", refField, ref.DocID, ref.SnippetID)
		}
		if ref.URL != target.url {
			return fmt.Errorf("%s.url does not match fetched external_doc URL", refField)
		}
		fetchedAt, err := time.Parse(time.RFC3339Nano, ref.FetchedAt)
		if err != nil {
			return fmt.Errorf("%s.fetched_at must be RFC3339: %w", refField, err)
		}
		if !fetchedAt.Equal(target.fetchedAt) {
			return fmt.Errorf("%s.fetched_at does not match fetched external_doc timestamp", refField)
		}
		if ref.ContentHash != target.contentHash {
			return fmt.Errorf("%s.content_hash does not match fetched external_doc snippet hash", refField)
		}
	}
	return nil
}

func indexReviewExternalDocSnippetRefs(docs []ReviewExternalDocEvidence) map[string]reviewExternalDocSnippetRefTarget {
	index := make(map[string]reviewExternalDocSnippetRefTarget)
	for _, doc := range docs {
		if doc.Error != "" || doc.DocID == "" || doc.URL == "" || doc.FetchedAt.IsZero() {
			continue
		}
		for _, snippet := range doc.Snippets {
			if snippet.SnippetID == "" || snippet.ContentHash == "" {
				continue
			}
			index[reviewExternalDocSnippetRefKey(doc.DocID, snippet.SnippetID)] = reviewExternalDocSnippetRefTarget{
				url:         doc.URL,
				fetchedAt:   doc.FetchedAt,
				contentHash: snippet.ContentHash,
			}
		}
	}
	return index
}

func reviewExternalDocSnippetRefKey(docID, snippetID string) string {
	return docID + "\x00" + snippetID
}
