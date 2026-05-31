package report

import (
	"testing"
	"time"
)

func TestValidateReviewReportBasicContract(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "base blocked report is valid",
			report: func() ReviewReport {
				return newValidReviewReportForValidationTest()
			},
		},
		{
			name: "invalid schema_version",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.SchemaVersion = ReviewReportSchemaVersionV1
				return report
			},
			wantErr:     true,
			errContains: "schema_version",
		},
		{
			name: "invalid target_kind",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.TargetKind = TargetKind("workspace_snapshot")
				return report
			},
			wantErr:     true,
			errContains: "target_kind",
		},
		{
			name: "zero generated_at",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.GeneratedAt = time.Time{}
				return report
			},
			wantErr:     true,
			errContains: "generated_at",
		},
		{
			name: "invalid verdict",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.Verdict = ReviewVerdict("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "verdict",
		},
		{
			name: "invalid overall_verification_status",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.OverallVerificationStatus = ReviewVerificationStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "invalid group severity",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Severity = ReviewGroupSeverity("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].severity",
		},
		{
			name: "invalid group verification_status enum",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].VerificationStatus = ReviewVerificationStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].verification_status",
		},
		{
			name: "empty root_cause_group id",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].ID = ""
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].id",
		},
		{
			name: "root_cause_group id with leading whitespace is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].ID = " rc-1"
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].id",
		},
		{
			name: "root_cause_group id with trailing whitespace is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].ID = "rc-1 "
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].id",
		},
		{
			name: "root_cause_group id with internal whitespace is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].ID = "rc 1"
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].id",
		},
		{
			name: "duplicate root_cause_group id",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups = append(report.RootCauseGroups, newRootCauseGroupForValidationTest("rc-1", "finding-2", ReviewVerificationVerified))
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[1].id",
		},
		{
			name: "empty finding id is valid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].ID = ""
				return report
			},
		},
		{
			name: "finding id with whitespace-only input is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].ID = " "
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].id",
		},
		{
			name: "finding id with leading whitespace is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].ID = " finding-1"
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].id",
		},
		{
			name: "finding id with trailing whitespace is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].ID = "finding-1 "
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].id",
		},
		{
			name: "finding id with internal whitespace is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].ID = "finding 1"
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].id",
		},
		{
			name: "duplicate non-empty finding id across groups",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups = append(report.RootCauseGroups, newRootCauseGroupForValidationTest("rc-2", "finding-1", ReviewVerificationVerified))
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[1].findings[0].id",
		},
		{
			name: "missing probe_summaries probe_id",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries[0].ProbeID = ""
				return report
			},
			wantErr:     true,
			errContains: "probe_summaries[0].probe_id",
		},
		{
			name: "duplicate probe_summaries probe_id",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries = append(report.ProbeSummaries, ReviewProbeSummary{
					ProbeID: "probe-1",
					Mode:    ReviewProbeHostReadOnly,
					Status:  ReviewProbePassed,
				})
				return report
			},
			wantErr:     true,
			errContains: "probe_summaries[1].probe_id",
		},
		{
			name: "invalid probe mode",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries[0].Mode = ReviewProbeMode("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "probe_summaries[0].mode",
		},
		{
			name: "invalid probe status",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries[0].Status = ReviewProbeStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "probe_summaries[0].status",
		},
		{
			name: "invalid probe command status",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries[0].Commands[0].Status = ReviewProbeStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "probe_summaries[0].commands[0].status",
		},
	}

	runReviewReportValidationCases(t, tests)
}
