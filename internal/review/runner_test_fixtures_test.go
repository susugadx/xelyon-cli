package review

import (
	"path/filepath"
	"time"
)

func newRunnerEvidenceBundleForTest(repoRoot string) ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		TargetKind:  TargetCurrentChanges,
		RepoRoot:    repoRoot,
		CWD:         filepath.Join(repoRoot, "internal"),
		StatusShort: " M internal/review/runner.go\n",
		ChangedFiles: []ReviewChangedFile{
			{
				Path:     filepath.Join(repoRoot, "internal/review/runner.go"),
				Status:   "M",
				Unstaged: true,
			},
		},
		RelatedSearchHits: []ReviewRelatedSearchHit{
			{
				Path:    filepath.Join(repoRoot, "internal/review/runner_test.go"),
				Line:    1,
				Snippet: "func TestReviewRunnerRun",
				Reason:  "runner tests mention review orchestration",
			},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{filepath.Join(repoRoot, "internal/review/runner.go")},
		},
		Limits: DefaultReviewEvidenceLimits(),
	}
}

func newRunnerProbePlanForTest(ids ...string) ReviewProbePlan {
	probes := make([]ReviewPlannedProbe, 0, len(ids))
	for _, id := range ids {
		probes = append(probes, ReviewPlannedProbe{
			ID:         id,
			SurfaceIDs: []string{"surface-1"},
			RiskIDs:    []string{"risk-1"},
			Purpose:    "Confirm or falsify risk-1 for surface-1 with focused review checks.",
			Mode:       ReviewProbeHostReadOnly,
			Commands: []ReviewPlannedProbeCommand{
				{
					Command: "go",
					Args:    []string{"test", "./internal/review"},
					WorkDir: ".",
				},
			},
			TimeoutSeconds: 30,
			MaxOutputBytes: 4096,
		})
	}
	return ReviewProbePlan{
		SchemaVersion: ReviewProbePlanSchemaVersionV2,
		TargetKind:    TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Runner orchestration may need verification.",
				Category:        ReviewProbeImpactSurfaceChangedFile,
				EvidenceSummary: "Evidence references current review changes at internal/review/runner.go.",
				Status:          ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Run the planned probes in order.",
			},
		},
		CandidateRisks: []ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "A runner contract could regress.",
				Severity:             ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Runner tests cover probe orchestration.",
				VerificationStrategy: "Execute the focused runner probe.",
				Status:               ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: probes,
	}
}

func newRunnerNoProbePlanForTest() ReviewProbePlan {
	return markReviewProbePlanCheckedWithoutProbesForTest(newRunnerProbePlanForTest())
}

func newRunnerCleanReportForTest(probeSummaries []ReviewProbeSummary) ReviewReport {
	var reportProbeSummaries []ReviewProbeSummary
	if len(probeSummaries) > 0 {
		reportProbeSummaries = probeSummaries
	}
	return ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV2,
		TargetKind:                TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: ReviewVerificationVerified,
		Verdict:                   ReviewVerdictClean,
		ProbeSummaries:            reportProbeSummaries,
		ScopeCoverage:             newCleanScopeCoverageForTest(),
	}
}

func newRunnerCleanReportWithPassedProbeEvidenceForTest(probeID string) ReviewReport {
	report := newRunnerCleanReportForTest(nil)
	ref := ReviewEvidenceRef{
		Kind:    ReviewEvidenceKindProbe,
		ProbeID: probeID,
	}
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	return report
}

func newRunnerBlockedReportForTest(probeSummaries []ReviewProbeSummary) ReviewReport {
	report := newRunnerCleanReportForTest(probeSummaries)
	report.OverallVerificationStatus = ReviewVerificationBlockedOrInconclusive
	report.Verdict = ReviewVerdictBlocked
	report.Summary = "Review blocked by probe execution."
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
	report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskUnverified
	return report
}

func withComputedSummaryForRunnerTest(report ReviewReport, probeSummaries []ReviewProbeSummary) ReviewReport {
	computedSummary := ComputeReviewReportComputedSummary(report, probeSummaries)
	report.ComputedSummary = &computedSummary
	return report
}
