package analysis

import (
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

type reviewExternalDocSnippetRefTarget struct {
	url         string
	fetchedAt   time.Time
	contentHash string
}

// ValidateProbePlanExternalDocRefs は probe plan の external_doc evidence refs が取得済み snippet を指すことを検証する。
func ValidateProbePlanExternalDocRefs(plan reviewprobe.ReviewProbePlan, docs []externaldoc.Evidence) error {
	for i, surface := range plan.ImpactSurfaces {
		if err := ValidateExternalDocEvidenceRefs(fmt.Sprintf("impact_surfaces[%d].evidence_refs", i), surface.EvidenceRefs, docs); err != nil {
			return err
		}
	}
	for i, risk := range plan.CandidateRisks {
		if err := ValidateExternalDocEvidenceRefs(fmt.Sprintf("candidate_risks[%d].evidence_refs", i), risk.EvidenceRefs, docs); err != nil {
			return err
		}
	}
	return nil
}

// ValidateReportExternalDocRefs は report の external_doc evidence refs が取得済み snippet を指すことを検証する。
func ValidateReportExternalDocRefs(report reviewreport.ReviewReport, docs []externaldoc.Evidence) error {
	return reviewreport.WalkReviewReportEvidenceRefs(report, func(field string, refs []reviewreport.ReviewEvidenceRef) error {
		return ValidateExternalDocEvidenceRefs(field, refs, docs)
	})
}

// ValidateSaturationExternalDocRefs は saturation check の external_doc evidence refs が取得済み snippet を指すことを検証する。
func ValidateSaturationExternalDocRefs(check reviewreport.ReviewSaturationCheck, docs []externaldoc.Evidence) error {
	return reviewreport.WalkReviewSaturationEvidenceRefs(check, func(field string, refs []reviewreport.ReviewEvidenceRef) error {
		return ValidateExternalDocEvidenceRefs(field, refs, docs)
	})
}

// ValidateExternalDocEvidenceRefs は external_doc evidence refs と fetched snippet index を照合する。
func ValidateExternalDocEvidenceRefs(field string, refs []reviewreport.ReviewEvidenceRef, docs []externaldoc.Evidence) error {
	index := indexReviewExternalDocSnippetRefs(docs)
	for i, ref := range refs {
		if ref.Kind != reviewreport.ReviewEvidenceKindExternalDoc {
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

func indexReviewExternalDocSnippetRefs(docs []externaldoc.Evidence) map[string]reviewExternalDocSnippetRefTarget {
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
