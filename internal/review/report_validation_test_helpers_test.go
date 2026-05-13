package review

import (
	"strings"
	"testing"
	"time"
)

type reviewReportValidationCase struct {
	name        string
	report      func() ReviewReport
	wantErr     bool
	errContains string
}

func runReviewReportValidationCases(t *testing.T, tests []reviewReportValidationCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewReport(tt.report())
			if tt.wantErr {
				if err == nil {
					t.Fatal("ValidateReviewReport() error = nil, want error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("ValidateReviewReport() error = %q, want substring %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateReviewReport() error = %v, want nil", err)
			}
		})
	}
}

func newValidReviewReportForValidationTest() ReviewReport {
	return ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV2,
		TargetKind:                TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: ReviewVerificationBlockedOrInconclusive,
		Verdict:                   ReviewVerdictBlocked,
		Summary:                   "review did not complete",
		RootCauseGroups:           []ReviewRootCauseGroup{},
		ProbeSummaries: []ReviewProbeSummary{
			{
				ProbeID: "probe-1",
				Mode:    ReviewProbeHostReadOnly,
				Status:  ReviewProbePassed,
				Commands: []ReviewProbeCommandSummary{
					{Command: "rg", Status: ReviewProbePassed},
				},
			},
		},
	}
}

func newCleanReportForValidationTest(overallStatus ReviewVerificationStatus) ReviewReport {
	report := newValidReviewReportForValidationTest()
	report.Verdict = ReviewVerdictClean
	report.OverallVerificationStatus = overallStatus
	report.Summary = ""
	report.RootCauseGroups = nil
	return report
}

func newBlockedReportForValidationTest(overallStatus ReviewVerificationStatus) ReviewReport {
	report := newValidReviewReportForValidationTest()
	report.Verdict = ReviewVerdictBlocked
	report.OverallVerificationStatus = overallStatus
	report.RootCauseGroups = nil
	report.Summary = "review did not complete"
	return report
}

func newHasFindingsReportForValidationTest(overallStatus ReviewVerificationStatus, groupStatus ReviewVerificationStatus) ReviewReport {
	report := newValidReviewReportForValidationTest()
	report.Verdict = ReviewVerdictHasFindings
	report.OverallVerificationStatus = overallStatus
	report.Summary = ""
	report.RootCauseGroups = newRootCauseGroupsForValidationTest(groupStatus)
	return report
}

func newRootCauseGroupsForValidationTest(status ReviewVerificationStatus) []ReviewRootCauseGroup {
	return []ReviewRootCauseGroup{
		newRootCauseGroupForValidationTest("rc-1", "finding-1", status),
	}
}

func newRootCauseGroupForValidationTest(id, findingID string, status ReviewVerificationStatus) ReviewRootCauseGroup {
	return ReviewRootCauseGroup{
		ID:                 id,
		Title:              "test group",
		Severity:           ReviewGroupSeverityLow,
		VerificationStatus: status,
		FixStrategy:        "fix root cause",
		VerificationPlan:   []string{"run focused validation"},
		Findings: []ReviewFinding{
			{
				ID:    findingID,
				Title: "finding",
				EvidenceRefs: []ReviewEvidenceRef{
					newFileEvidenceRefForValidationTest(),
				},
			},
		},
	}
}

func newFileEvidenceRefForValidationTest() ReviewEvidenceRef {
	return ReviewEvidenceRef{
		Kind: ReviewEvidenceKindFile,
		Path: "internal/review/report_validation.go",
		Line: 1,
	}
}

func newValidEvidenceRefForValidationTest() ReviewEvidenceRef {
	return ReviewEvidenceRef{
		Kind:         ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: ReviewCommandIndex(0),
		Path:         "internal/review/report_validation.go",
		Line:         1,
	}
}

func setCheckedSurfaceEvidenceRefForValidationTest(report *ReviewReport, ref ReviewEvidenceRef) {
	report.CheckedSurfaces = []ReviewSurfaceCoverage{
		{
			SurfaceID:    "surface-1",
			EvidenceRefs: []ReviewEvidenceRef{ref},
		},
	}
}
