package review

import "fmt"

func validateReviewReportTrustedProbeSummaries(report ReviewReport, trustedProbeSummaries []ReviewProbeSummary) error {
	if len(report.ProbeSummaries) != len(trustedProbeSummaries) {
		return fmt.Errorf("probe_summaries must match trusted probe summaries count: got %d, want %d", len(report.ProbeSummaries), len(trustedProbeSummaries))
	}
	for i, trusted := range trustedProbeSummaries {
		if report.ProbeSummaries[i].ProbeID != trusted.ProbeID {
			return fmt.Errorf("probe_summaries[%d].probe_id must match trusted probe summary ID %q: got %q", i, trusted.ProbeID, report.ProbeSummaries[i].ProbeID)
		}
	}
	return nil
}

func validateReviewReportProbeOutcomesAgainstPlan(report ReviewReport, plan ReviewProbePlan, trustedProbeSummaries []ReviewProbeSummary) error {
	trustedByID := make(map[string]ReviewProbeSummary, len(trustedProbeSummaries))
	for _, summary := range trustedProbeSummaries {
		trustedByID[summary.ProbeID] = summary
	}

	if report.Verdict == ReviewVerdictClean {
		for i, summary := range trustedProbeSummaries {
			if isReviewProbeSummaryNonPassingForScopeValidation(summary) {
				return fmt.Errorf("verdict %q is not allowed when trusted probe_summaries[%d] status is %q", ReviewVerdictClean, i, canonicalReviewProbeSummaryStatusForValidation(summary))
			}
		}
	}

	linkage := newReviewReportProbePlanLinkageIndex(plan)
	if err := validateReviewReportImpactSurfaceProbeOutcomeCoverage(report.ScopeCoverage, plan, trustedByID, linkage); err != nil {
		return err
	}
	if err := validateReviewReportCandidateRiskProbeOutcomeCoverage(report.ScopeCoverage, plan, trustedByID, linkage); err != nil {
		return err
	}
	return nil
}

type reviewReportProbePlanLinkageIndex struct {
	surfaceProbeIDs map[string]map[string]struct{}
	riskProbeIDs    map[string]map[string]struct{}
}

func newReviewReportProbePlanLinkageIndex(plan ReviewProbePlan) reviewReportProbePlanLinkageIndex {
	index := reviewReportProbePlanLinkageIndex{
		surfaceProbeIDs: make(map[string]map[string]struct{}),
		riskProbeIDs:    make(map[string]map[string]struct{}),
	}
	for _, probe := range plan.Probes {
		for _, surfaceID := range probe.SurfaceIDs {
			addReviewReportLinkedProbeID(index.surfaceProbeIDs, surfaceID, probe.ID)
		}
		for _, riskID := range probe.RiskIDs {
			addReviewReportLinkedProbeID(index.riskProbeIDs, riskID, probe.ID)
		}
	}
	return index
}

func addReviewReportLinkedProbeID(index map[string]map[string]struct{}, itemID, probeID string) {
	if _, exists := index[itemID]; !exists {
		index[itemID] = make(map[string]struct{})
	}
	index[itemID][probeID] = struct{}{}
}

func validateReviewReportImpactSurfaceProbeOutcomeCoverage(coverage *ReviewReportScopeCoverage, plan ReviewProbePlan, trustedByID map[string]ReviewProbeSummary, linkage reviewReportProbePlanLinkageIndex) error {
	coverageByID := make(map[string]ReviewReportImpactSurfaceCoverage, len(coverage.ReviewedImpactSurfaces))
	for _, surface := range coverage.ReviewedImpactSurfaces {
		coverageByID[surface.SurfaceID] = surface
	}
	for i, surface := range plan.ImpactSurfaces {
		covered := coverageByID[surface.ID]
		linkedProbeIDs := linkage.surfaceProbeIDs[surface.ID]
		if covered.Status == ReviewReportImpactSurfaceChecked && hasReviewReportNonPassingLinkedProbe(linkedProbeIDs, trustedByID) {
			return fmt.Errorf("scope_coverage.reviewed_impact_surfaces for impact_surfaces[%d].id %q must not be %q when a linked trusted probe did not pass", i, surface.ID, ReviewReportImpactSurfaceChecked)
		}
		if surface.Status == ReviewProbeImpactSurfaceNeedsProbe || surface.Status == ReviewProbeImpactSurfaceUnverified {
			if covered.Status == ReviewReportImpactSurfaceChecked && !hasReviewReportPassedLinkedProbeEvidence(covered.EvidenceRefs, linkedProbeIDs, trustedByID) {
				return fmt.Errorf("scope_coverage.reviewed_impact_surfaces for impact_surfaces[%d].id %q requires passed linked probe evidence_ref before status %q", i, surface.ID, ReviewReportImpactSurfaceChecked)
			}
		}
	}
	return nil
}

func validateReviewReportCandidateRiskProbeOutcomeCoverage(coverage *ReviewReportScopeCoverage, plan ReviewProbePlan, trustedByID map[string]ReviewProbeSummary, linkage reviewReportProbePlanLinkageIndex) error {
	coverageByID := make(map[string]ReviewReportCandidateRiskCoverage, len(coverage.ReviewedCandidateRisks))
	for _, risk := range coverage.ReviewedCandidateRisks {
		coverageByID[risk.RiskID] = risk
	}
	for i, risk := range plan.CandidateRisks {
		covered := coverageByID[risk.ID]
		linkedProbeIDs := linkage.riskProbeIDs[risk.ID]
		if covered.Status == ReviewReportCandidateRiskDismissed && hasReviewReportNonPassingLinkedProbe(linkedProbeIDs, trustedByID) {
			return fmt.Errorf("scope_coverage.reviewed_candidate_risks for candidate_risks[%d].id %q must not be %q when a linked trusted probe did not pass", i, risk.ID, ReviewReportCandidateRiskDismissed)
		}
		if risk.Status == ReviewProbeCandidateRiskNeedsProbe || risk.Status == ReviewProbeCandidateRiskUnverified {
			if covered.Status == ReviewReportCandidateRiskDismissed && !hasReviewReportPassedLinkedProbeEvidence(covered.EvidenceRefs, linkedProbeIDs, trustedByID) {
				return fmt.Errorf("scope_coverage.reviewed_candidate_risks for candidate_risks[%d].id %q requires passed linked probe evidence_ref before status %q", i, risk.ID, ReviewReportCandidateRiskDismissed)
			}
		}
	}
	return nil
}

func hasReviewReportPassedLinkedProbeEvidence(refs []ReviewEvidenceRef, linkedProbeIDs map[string]struct{}, trustedByID map[string]ReviewProbeSummary) bool {
	for _, ref := range refs {
		if ref.Kind != ReviewEvidenceKindProbe && ref.Kind != ReviewEvidenceKindProbeCommand {
			continue
		}
		if _, linked := linkedProbeIDs[ref.ProbeID]; !linked {
			continue
		}
		summary, exists := trustedByID[ref.ProbeID]
		if !exists {
			continue
		}
		if summary.Status == ReviewProbePassed && !isReviewProbeSummaryMutationOutcome(summary) {
			return true
		}
	}
	return false
}

func hasReviewReportNonPassingLinkedProbe(linkedProbeIDs map[string]struct{}, trustedByID map[string]ReviewProbeSummary) bool {
	for probeID := range linkedProbeIDs {
		summary, exists := trustedByID[probeID]
		if !exists {
			continue
		}
		if isReviewProbeSummaryNonPassingForScopeValidation(summary) {
			return true
		}
	}
	return false
}

func isReviewProbeSummaryNonPassingForScopeValidation(summary ReviewProbeSummary) bool {
	if isReviewProbeSummaryMutationOutcome(summary) {
		return true
	}
	switch summary.Status {
	case ReviewProbeFailed, ReviewProbeBlocked, ReviewProbeTimedOut, ReviewProbeMutatedWorktree:
		return true
	default:
		return false
	}
}

func canonicalReviewProbeSummaryStatusForValidation(summary ReviewProbeSummary) ReviewProbeStatus {
	if isReviewProbeSummaryMutationOutcome(summary) {
		return ReviewProbeMutatedWorktree
	}
	return summary.Status
}
