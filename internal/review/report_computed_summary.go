package review

// ComputeReviewReportComputedSummary は validated report と runner trusted probe summaries から
// final report 用の派生 count を算出する。
func ComputeReviewReportComputedSummary(report ReviewReport, probeSummaries []ReviewProbeSummary) ReviewReportComputedSummary {
	summary := ReviewReportComputedSummary{
		RootCauseGroupCount: len(report.RootCauseGroups),
	}
	for _, group := range report.RootCauseGroups {
		summary.FindingCount += len(group.Findings)
	}

	if report.ScopeCoverage != nil {
		addReviewReportComputedScopeCounts(&summary, *report.ScopeCoverage)
	}
	addReviewReportComputedProbeCounts(&summary, probeSummaries)
	return summary
}

func addReviewReportComputedScopeCounts(summary *ReviewReportComputedSummary, coverage ReviewReportScopeCoverage) {
	for _, surface := range coverage.ReviewedImpactSurfaces {
		switch surface.Status {
		case ReviewReportImpactSurfaceChecked:
			summary.CheckedSurfaceCount++
		case ReviewReportImpactSurfaceFinding:
			summary.FindingSurfaceCount++
		case ReviewReportImpactSurfaceUnverified:
			summary.UnverifiedSurfaceCount++
		case ReviewReportImpactSurfaceResidualRisk:
			summary.ResidualSurfaceCount++
		}
	}

	summary.CandidateRiskCount = len(coverage.ReviewedCandidateRisks)
	for _, risk := range coverage.ReviewedCandidateRisks {
		switch risk.Status {
		case ReviewReportCandidateRiskDismissed:
			summary.DismissedRiskCount++
		case ReviewReportCandidateRiskFinding:
			summary.FindingRiskCount++
		case ReviewReportCandidateRiskUnverified:
			summary.UnverifiedRiskCount++
		case ReviewReportCandidateRiskResidualRisk:
			summary.ResidualRiskCount++
		}
	}
	summary.NewReportPassFindingCount = countReviewReportPassFindingIDs(coverage.NewFindingsFromReportPass)
}

func countReviewReportPassFindingIDs(findings []ReviewReportPassFindingCoverage) int {
	seen := make(map[string]struct{})
	for _, finding := range findings {
		for _, findingID := range finding.FindingIDs {
			if findingID == "" {
				continue
			}
			seen[findingID] = struct{}{}
		}
	}
	return len(seen)
}

func addReviewReportComputedProbeCounts(summary *ReviewReportComputedSummary, probeSummaries []ReviewProbeSummary) {
	summary.ProbeCount = len(probeSummaries)
	for _, probe := range probeSummaries {
		probe = canonicalizeReviewProbeSummaryMutationOutcome(probe)
		switch probe.Status {
		case ReviewProbePassed:
			summary.PassedProbeCount++
		case ReviewProbeFailed:
			summary.FailedProbeCount++
		case ReviewProbeTimedOut:
			summary.TimedOutProbeCount++
		case ReviewProbeBlocked:
			summary.BlockedProbeCount++
		case ReviewProbeMutatedWorktree:
			summary.MutatedWorktreeProbeCount++
		}
	}
}
