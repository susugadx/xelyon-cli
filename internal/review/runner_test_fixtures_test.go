package review

import (
	"path/filepath"
	"time"

	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func newRunnerEvidenceBundleForTest(repoRoot string) reviewevidence.ReviewEvidenceBundle {
	return reviewevidence.ReviewEvidenceBundle{
		TargetKind:  reviewdomain.TargetCurrentChanges,
		RepoRoot:    repoRoot,
		CWD:         filepath.Join(repoRoot, "internal"),
		StatusShort: " M internal/review/runner.go\n",
		ChangedFiles: []reviewevidence.ReviewChangedFile{
			{
				Path:     filepath.Join(repoRoot, "internal/review/runner.go"),
				Status:   "M",
				Unstaged: true,
			},
		},
		RelatedSearchHits: []reviewevidence.ReviewRelatedSearchHit{
			{
				Path:    filepath.Join(repoRoot, "internal/review/runner_test.go"),
				Line:    1,
				Snippet: "func TestReviewRunnerRun",
				Reason:  "runner tests mention review orchestration",
			},
		},
		Inventory: reviewevidence.ReviewChangeInventory{
			Production: []string{filepath.Join(repoRoot, "internal/review/runner.go")},
		},
		Limits: reviewevidence.DefaultReviewEvidenceLimits(),
	}
}

func newRunnerProbePlanForTest(ids ...string) reviewprobeplan.ReviewProbePlan {
	probes := make([]reviewprobeplan.ReviewPlannedProbe, 0, len(ids))
	for _, id := range ids {
		probes = append(probes, reviewprobeplan.ReviewPlannedProbe{
			ID:         id,
			SurfaceIDs: []string{"surface-1"},
			RiskIDs:    []string{"risk-1"},
			Purpose:    "Confirm or falsify risk-1 for surface-1 with focused review checks.",
			Mode:       reviewprobeplan.ReviewProbeHostReadOnly,
			Commands: []reviewprobeplan.ReviewPlannedProbeCommand{
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
	return reviewprobeplan.ReviewProbePlan{
		SchemaVersion: reviewprobeplan.ReviewProbePlanSchemaVersionV2,
		TargetKind:    reviewdomain.TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []reviewprobeplan.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Runner orchestration may need verification.",
				Category:        reviewprobeplan.ReviewProbeImpactSurfaceChangedFile,
				EvidenceSummary: "Evidence references current review changes at internal/review/runner.go.",
				Status:          reviewprobeplan.ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Run the planned probes in order.",
			},
		},
		CandidateRisks: []reviewprobeplan.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "A runner contract could regress.",
				Severity:             reviewprobeplan.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Runner tests cover probe orchestration.",
				VerificationStrategy: "Execute the focused runner probe.",
				Status:               reviewprobeplan.ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: probes,
	}
}

func newRunnerNoProbePlanForTest() reviewprobeplan.ReviewProbePlan {
	return markReviewProbePlanCheckedWithoutProbesForTest(newRunnerProbePlanForTest())
}

func newRunnerCleanReportForTest(probeSummaries []reviewreport.ReviewProbeSummary) reviewreport.ReviewReport {
	var reportProbeSummaries []reviewreport.ReviewProbeSummary
	if len(probeSummaries) > 0 {
		reportProbeSummaries = probeSummaries
	}
	return reviewreport.ReviewReport{
		SchemaVersion:             reviewreport.ReviewReportSchemaVersionV2,
		TargetKind:                reviewdomain.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: reviewreport.ReviewVerificationVerified,
		Verdict:                   reviewreport.ReviewVerdictClean,
		ProbeSummaries:            reportProbeSummaries,
		ScopeCoverage:             newCleanScopeCoverageForTest(),
	}
}

func newRunnerCleanReportWithPassedProbeEvidenceForTest(probeID string) reviewreport.ReviewReport {
	report := newRunnerCleanReportForTest(nil)
	ref := reviewreport.ReviewEvidenceRef{
		Kind:    reviewreport.ReviewEvidenceKindProbe,
		ProbeID: probeID,
	}
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
	return report
}

func newRunnerBlockedReportForTest(probeSummaries []reviewreport.ReviewProbeSummary) reviewreport.ReviewReport {
	report := newRunnerCleanReportForTest(probeSummaries)
	report.OverallVerificationStatus = reviewreport.ReviewVerificationBlockedOrInconclusive
	report.Verdict = reviewreport.ReviewVerdictBlocked
	report.Summary = "Review blocked by probe execution."
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = reviewreport.ReviewReportImpactSurfaceUnverified
	report.ScopeCoverage.ReviewedCandidateRisks[0].Status = reviewreport.ReviewReportCandidateRiskUnverified
	return report
}

func withComputedSummaryForRunnerTest(report reviewreport.ReviewReport, probeSummaries []reviewreport.ReviewProbeSummary) reviewreport.ReviewReport {
	computedSummary := reviewreport.ComputeReviewReportComputedSummary(report, probeSummaries)
	report.ComputedSummary = &computedSummary
	return report
}
