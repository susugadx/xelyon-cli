package report

import "fmt"

func auditNonPassingProbeCoverage(input CoverageAuditInput) []CoverageIssue {
	if input.Report.ScopeCoverage == nil {
		return nil
	}

	summaries := input.TrustedProbeSummaries
	if summaries == nil {
		summaries = input.Report.ProbeSummaries
	}
	if len(summaries) == 0 {
		return nil
	}

	linkage := newReviewReportProbePlanLinkageIndex(input.Plan)
	surfaces := indexCoverageImpactSurfaces(input.Report.ScopeCoverage)
	risks := indexCoverageCandidateRisks(input.Report.ScopeCoverage)
	var issues []CoverageIssue
	for _, summary := range summaries {
		if !isReviewProbeSummaryNonPassingForScopeValidation(summary) {
			continue
		}
		for _, surface := range input.Plan.ImpactSurfaces {
			if _, linked := linkage.surfaceProbeIDs[surface.ID][summary.ProbeID]; !linked {
				continue
			}
			coverage, ok := surfaces[surface.ID]
			if !ok || coverage.Status != ReviewReportImpactSurfaceChecked {
				continue
			}
			if scopeCoverageReflectsProbeOutcome(coverage.Summary, coverage.EvidenceRefs, summary, input.Report) {
				continue
			}
			issues = append(issues, newProbeOutcomeCoverageIssue(
				[]string{surface.ID},
				nil,
				summary,
				fmt.Sprintf("impact surface %q is %q", surface.ID, coverage.Status),
			))
		}
		for _, risk := range input.Plan.CandidateRisks {
			if _, linked := linkage.riskProbeIDs[risk.ID][summary.ProbeID]; !linked {
				continue
			}
			coverage, ok := risks[risk.ID]
			if !ok || coverage.Status != ReviewReportCandidateRiskDismissed {
				continue
			}
			if scopeCoverageReflectsProbeOutcome(coverage.Summary, coverage.EvidenceRefs, summary, input.Report) {
				continue
			}
			issues = append(issues, newProbeOutcomeCoverageIssue(
				nil,
				[]string{risk.ID},
				summary,
				fmt.Sprintf("candidate risk %q is %q", risk.ID, coverage.Status),
			))
		}
	}
	return issues
}

func indexCoverageImpactSurfaces(coverage *ReviewReportScopeCoverage) map[string]ReviewReportImpactSurfaceCoverage {
	byID := make(map[string]ReviewReportImpactSurfaceCoverage, len(coverage.ReviewedImpactSurfaces))
	for _, surface := range coverage.ReviewedImpactSurfaces {
		byID[surface.SurfaceID] = surface
	}
	return byID
}

func indexCoverageCandidateRisks(coverage *ReviewReportScopeCoverage) map[string]ReviewReportCandidateRiskCoverage {
	byID := make(map[string]ReviewReportCandidateRiskCoverage, len(coverage.ReviewedCandidateRisks))
	for _, risk := range coverage.ReviewedCandidateRisks {
		byID[risk.RiskID] = risk
	}
	return byID
}

func newProbeOutcomeCoverageIssue(surfaceIDs, riskIDs []string, summary ReviewProbeSummary, scope string) CoverageIssue {
	status := canonicalReviewProbeSummaryStatusForValidation(summary)
	return CoverageIssue{
		Kind:         CoverageIssueKindUnreflectedProbeOutcome,
		Severity:     CoverageIssueSeverityHigh,
		SurfaceIDs:   surfaceIDs,
		RiskIDs:      riskIDs,
		ProbeID:      summary.ProbeID,
		EvidenceRefs: []ReviewEvidenceRef{{Kind: ReviewEvidenceKindProbe, ProbeID: summary.ProbeID}},
		Summary: fmt.Sprintf(
			"Linked probe %q ended with %q, but %s without probe evidence or explanation.",
			summary.ProbeID,
			status,
			scope,
		),
		RevisionInstruction: fmt.Sprintf(
			"Revisit the scope linked to non-passing probe %q; do not leave it clean/dismissed unless the report explains the outcome with supplied probe evidence, otherwise classify it as finding, residual_risk, unverified, or blocked.",
			summary.ProbeID,
		),
	}
}

func scopeCoverageReflectsProbeOutcome(summary string, refs []ReviewEvidenceRef, probe ReviewProbeSummary, report ReviewReport) bool {
	if hasProbeEvidenceRef(refs, probe.ProbeID) {
		return true
	}
	if textReflectsProbeOutcome(summary, probe) {
		return true
	}
	for _, text := range collectReviewReportCoverageTexts(report) {
		if textReflectsProbeOutcome(text, probe) {
			return true
		}
	}
	return false
}

func hasProbeEvidenceRef(refs []ReviewEvidenceRef, probeID string) bool {
	for _, ref := range refs {
		if ref.ProbeID != probeID {
			continue
		}
		if ref.Kind == ReviewEvidenceKindProbe || ref.Kind == ReviewEvidenceKindProbeCommand {
			return true
		}
	}
	return false
}
