package report

import "testing"

func TestValidateReviewReportEvidenceContract(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "unknown evidence kind",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Kind = "unexpected"
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].kind",
		},
		{
			name: "probe_command evidence without probe_id",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.ProbeID = ""
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].probe_id",
		},
		{
			name: "probe_command evidence without command_index",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.CommandIndex = nil
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].command_index",
		},
		{
			name: "command_index without probe_id",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Kind = ReviewEvidenceKindFile
				ref.ProbeID = ""
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].command_index",
		},
		{
			name: "command_index out of range",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.CommandIndex = ReviewCommandIndex(1)
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].command_index",
		},
		{
			name: "command_index zero is valid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.CommandIndex = ReviewCommandIndex(0)
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
		},
		{
			name: "line must be >= 0",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Line = -1
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].line",
		},
		{
			name: "line > 0 requires path",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = ""
				ref.Line = 10
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "file evidence requires path",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := ReviewEvidenceRef{Kind: ReviewEvidenceKindFile}
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "diff evidence requires path",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := ReviewEvidenceRef{Kind: ReviewEvidenceKindDiff}
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "rule_file evidence requires path",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := ReviewEvidenceRef{Kind: ReviewEvidenceKindRuleFile}
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "git_status evidence without path is valid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := ReviewEvidenceRef{Kind: ReviewEvidenceKindGitStatus}
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
		},
		{
			name: "probe evidence without path is valid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := ReviewEvidenceRef{
					Kind:    ReviewEvidenceKindProbe,
					ProbeID: "probe-1",
				}
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
		},
		{
			name: "probe_command evidence without path is valid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = ""
				ref.Line = 0
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
		},
		{
			name: "absolute evidence path is invalid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = "/tmp/review/report_validation.go"
				ref.Line = 0
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "evidence path with parent escape is invalid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = "../outside.go"
				ref.Line = 0
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "evidence path with windows-style parent escape is invalid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				ref.Line = 0
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].evidence_refs[0].path",
		},
	}

	runReviewReportValidationCases(t, tests)
}
