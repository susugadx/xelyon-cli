package review

import (
	"fmt"
	"strings"

	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

type reviewCoverageAuditContext struct {
	postPass1ExternalEvidence reviewreport.CoverageExternalEvidenceDelta
	externalSupport           reviewreport.CoverageExternalSupport
}

func buildReviewCoverageAuditContext(before ReviewWebSearchEvidence, bundle ReviewEvidenceBundle) reviewCoverageAuditContext {
	support := BuildReviewEvidenceModelInput(bundle).ExternalSupport
	externalSupport := reviewreport.CoverageExternalSupport{
		Level:                                string(support.Level),
		DocCount:                             support.DocCount,
		CitationCapableDocCount:              support.CitationCapableDocCount,
		CitationCapableSnippetCount:          support.CitationCapableSnippetCount,
		OfficialCandidateDocCount:            support.OfficialCandidateDocCount,
		OfficialCandidateCitationCapableDocs: support.OfficialCandidateCitationCapableDocCount,
		OfficialConfirmation:                 support.OfficialConfirmation,
		Warnings:                             append([]string(nil), support.Warnings...),
		Reasons:                              append([]string(nil), support.Reasons...),
	}
	return reviewCoverageAuditContext{
		postPass1ExternalEvidence: buildReviewCoverageExternalEvidenceDelta(before, bundle.WebSearchEvidence, externalSupport),
		externalSupport:           externalSupport,
	}
}

func buildReviewCoverageExternalEvidenceDelta(before, after ReviewWebSearchEvidence, support reviewreport.CoverageExternalSupport) reviewreport.CoverageExternalEvidenceDelta {
	delta := reviewreport.CoverageExternalEvidenceDelta{}

	addedQueries := addedReviewWebSearchEvidenceQueries(before, after)
	delta.AddedQueryCount = len(addedQueries)
	for _, query := range addedQueries {
		queryText := strings.TrimSpace(query.Query)
		if queryText != "" {
			delta.AddedQueries = append(delta.AddedQueries, queryText)
		}
		if strings.TrimSpace(query.Error) != "" {
			delta.AddedFailedQueryCount++
			if queryText != "" {
				delta.AddedFailedQueries = append(delta.AddedFailedQueries, queryText)
			}
		}
		if len(query.Results) == 0 {
			delta.AddedNoResultCount++
			if queryText != "" {
				delta.AddedNoResultQueries = append(delta.AddedNoResultQueries, queryText)
			}
		}
	}

	for _, doc := range addedReviewExternalDocs(before, after) {
		if docID := strings.TrimSpace(doc.DocID); docID != "" {
			delta.AddedDocIDs = append(delta.AddedDocIDs, docID)
		}
		if docURL := strings.TrimSpace(doc.URL); docURL != "" {
			delta.AddedDocURLs = append(delta.AddedDocURLs, docURL)
		}
		if strings.TrimSpace(doc.Error) != "" {
			delta.EvidenceError = true
		}
		if doc.Truncated {
			delta.Truncated = true
		}
	}
	if after.Truncated && !before.Truncated {
		delta.Truncated = true
	}
	if after.Inconclusive && (!before.Inconclusive || delta.AddedQueryCount > 0 || len(delta.AddedDocIDs) > 0 || len(delta.AddedDocURLs) > 0) {
		delta.Inconclusive = true
	}
	if strings.TrimSpace(after.Error) != "" && strings.TrimSpace(after.Error) != strings.TrimSpace(before.Error) {
		delta.EvidenceError = true
	}
	if delta.AddedQueryCount == 0 &&
		len(delta.AddedDocIDs) == 0 &&
		len(delta.AddedDocURLs) == 0 &&
		!delta.EvidenceError &&
		!delta.Inconclusive &&
		!delta.Truncated {
		return reviewreport.CoverageExternalEvidenceDelta{}
	}
	delta.Warnings = append([]string(nil), support.Warnings...)
	delta.Reasons = append([]string(nil), support.Reasons...)
	return delta
}

func addedReviewWebSearchEvidenceQueries(before, after ReviewWebSearchEvidence) []ReviewWebSearchEvidenceQuery {
	if len(after.Queries) <= len(before.Queries) {
		return nil
	}
	return append([]ReviewWebSearchEvidenceQuery(nil), after.Queries[len(before.Queries):]...)
}

func addedReviewExternalDocs(before, after ReviewWebSearchEvidence) []ReviewExternalDocEvidence {
	if len(after.ExternalDocs) <= len(before.ExternalDocs) {
		return nil
	}
	return append([]ReviewExternalDocEvidence(nil), after.ExternalDocs[len(before.ExternalDocs):]...)
}

func mergeReviewCoverageAuditIntoSaturationCheck(check ReviewSaturationCheck, plan ReviewProbePlan, finalizedReport ReviewReport, probeSummaries []ReviewProbeSummary, auditContext reviewCoverageAuditContext) (ReviewSaturationCheck, error) {
	planScope := reviewanalysis.PlanScopeFromProbePlan(plan)
	issues := reviewreport.AuditReviewReportCoverage(reviewreport.CoverageAuditInput{
		Plan:                      planScope,
		Report:                    finalizedReport,
		TrustedProbeSummaries:     probeSummaries,
		PostPass1ExternalEvidence: auditContext.postPass1ExternalEvidence,
		ExternalSupport:           auditContext.externalSupport,
	})
	merged := reviewreport.MergeCoverageIssuesIntoSaturationCheck(check, issues)
	if err := reviewreport.ValidateReviewSaturationCheck(merged, planScope, finalizedReport); err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("review runner merge coverage audit: %w", err)
	}
	return merged, nil
}
