package promptreduction

import (
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

type reviewExternalDocAbsorptionRefSummary struct {
	owners        []string
	urls          map[string]struct{}
	fetchedAt     map[string]struct{}
	contentHashes map[string]struct{}
}

func reviewExternalDocAbsorptionRefs(report reviewreport.ReviewReport) (map[string]reviewExternalDocAbsorptionRefSummary, map[string]struct{}) {
	safeRefs := make(map[string]reviewExternalDocAbsorptionRefSummary)
	unsafeRefs := make(map[string]struct{})
	addRefs := func(refs []reviewreport.ReviewEvidenceRef, owner string, safe bool) {
		for _, ref := range refs {
			if ref.Kind != reviewreport.ReviewEvidenceKindExternalDoc {
				continue
			}
			key := reviewExternalDocSnippetAbsorptionKey(ref.DocID, ref.SnippetID)
			if key == "" {
				continue
			}
			if safe && strings.TrimSpace(ref.URL) != "" && strings.TrimSpace(ref.FetchedAt) != "" && strings.TrimSpace(ref.ContentHash) != "" {
				summary := safeRefs[key]
				summary.owners = append(summary.owners, owner)
				if summary.urls == nil {
					summary.urls = make(map[string]struct{})
				}
				if summary.fetchedAt == nil {
					summary.fetchedAt = make(map[string]struct{})
				}
				if summary.contentHashes == nil {
					summary.contentHashes = make(map[string]struct{})
				}
				summary.urls[strings.TrimSpace(ref.URL)] = struct{}{}
				summary.fetchedAt[strings.TrimSpace(ref.FetchedAt)] = struct{}{}
				summary.contentHashes[strings.TrimSpace(ref.ContentHash)] = struct{}{}
				safeRefs[key] = summary
				continue
			}
			unsafeRefs[key] = struct{}{}
		}
	}

	if report.ScopeCoverage != nil {
		for _, surface := range report.ScopeCoverage.ReviewedImpactSurfaces {
			owner := "scope_coverage.surface." + strings.TrimSpace(surface.SurfaceID)
			addRefs(surface.EvidenceRefs, owner, surface.Status == reviewreport.ReviewReportImpactSurfaceChecked)
		}
		for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
			owner := "scope_coverage.risk." + strings.TrimSpace(risk.RiskID)
			addRefs(risk.EvidenceRefs, owner, risk.Status == reviewreport.ReviewReportCandidateRiskDismissed)
		}
		for _, finding := range report.ScopeCoverage.NewFindingsFromReportPass {
			addRefs(finding.EvidenceRefs, "scope_coverage.new_finding", false)
		}
	}

	for _, surface := range report.CheckedSurfaces {
		addRefs(surface.EvidenceRefs, "checked_surfaces", false)
	}
	for _, surface := range report.UnverifiedSurfaces {
		addRefs(surface.EvidenceRefs, "unverified_surfaces", false)
	}
	for _, risk := range report.ResidualRisks {
		addRefs(risk.EvidenceRefs, "residual_risks", false)
	}
	for _, group := range report.RootCauseGroups {
		for _, finding := range group.Findings {
			addRefs(finding.EvidenceRefs, "root_cause.finding", false)
			for _, surface := range finding.CheckedSurfaces {
				addRefs(surface.EvidenceRefs, "root_cause.finding.checked_surface", false)
			}
			for _, surface := range finding.UnverifiedSurfaces {
				addRefs(surface.EvidenceRefs, "root_cause.finding.unverified_surface", false)
			}
			for _, risk := range finding.ResidualRisks {
				addRefs(risk.EvidenceRefs, "root_cause.finding.residual_risk", false)
			}
		}
		for _, surface := range group.CheckedSurfaces {
			addRefs(surface.EvidenceRefs, "root_cause.checked_surface", false)
		}
		for _, surface := range group.UnverifiedSurfaces {
			addRefs(surface.EvidenceRefs, "root_cause.unverified_surface", false)
		}
		for _, risk := range group.ResidualRisks {
			addRefs(risk.EvidenceRefs, "root_cause.residual_risk", false)
		}
	}

	for key, refs := range safeRefs {
		refs.owners = DedupeSortedReviewPromptAbsorptionRefs(refs.owners)
		safeRefs[key] = refs
	}
	return safeRefs, unsafeRefs
}

func (s reviewExternalDocAbsorptionRefSummary) matches(doc externaldoc.Evidence, snippet externaldoc.SnippetEvidence) bool {
	if len(s.urls) == 0 || len(s.fetchedAt) == 0 || len(s.contentHashes) == 0 {
		return false
	}
	if _, ok := s.urls[strings.TrimSpace(doc.URL)]; !ok {
		return false
	}
	if _, ok := s.fetchedAt[doc.FetchedAt.Format(time.RFC3339Nano)]; !ok {
		return false
	}
	if _, ok := s.contentHashes[strings.TrimSpace(snippet.ContentHash)]; !ok {
		return false
	}
	return true
}

func reviewExternalDocSnippetAbsorptionKey(docID, snippetID string) string {
	docID = strings.TrimSpace(docID)
	snippetID = strings.TrimSpace(snippetID)
	if docID == "" || snippetID == "" {
		return ""
	}
	return docID + "\x00" + snippetID
}
