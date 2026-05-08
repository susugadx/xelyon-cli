package review

import (
	"strings"
	"testing"
)

func TestValidateReviewReportRejectsNonCanonicalProbeID(t *testing.T) {
	tests := []struct {
		name        string
		report      func() ReviewReport
		errContains string
	}{
		{
			name: "probe_summaries probe_id with leading whitespace",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries[0].ProbeID = " probe-1"
				return report
			},
			errContains: "probe_summaries[0].probe_id",
		},
		{
			name: "probe_summaries probe_id with trailing whitespace",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries[0].ProbeID = "probe-1 "
				return report
			},
			errContains: "probe_summaries[0].probe_id",
		},
		{
			name: "duplicate probe_id with whitespace variation is rejected",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ProbeSummaries = append(report.ProbeSummaries, ReviewProbeSummary{
					ProbeID: "probe-1 ",
					Mode:    ReviewProbeHostReadOnly,
					Status:  ReviewProbePassed,
				})
				return report
			},
			errContains: "probe_summaries[1].probe_id",
		},
		{
			name: "evidence ref probe_id with leading whitespace",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.ProbeID = " probe-1"
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			errContains: "checked_surfaces[0].evidence_refs[0].probe_id",
		},
		{
			name: "evidence ref probe_id with trailing whitespace",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.ProbeID = "probe-1 "
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			errContains: "checked_surfaces[0].evidence_refs[0].probe_id",
		},
		{
			name: "unknown probe_id with leading whitespace rejects canonical form before lookup",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.ProbeID = " unknown-probe"
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			errContains: "checked_surfaces[0].evidence_refs[0].probe_id",
		},
		{
			name: "optional probe_id with whitespace-only input is invalid",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Kind = ReviewEvidenceKindFile
				ref.ProbeID = " "
				ref.CommandIndex = nil
				setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
				return report
			},
			errContains: "checked_surfaces[0].evidence_refs[0].probe_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewReport(tt.report())
			if err == nil {
				t.Fatal("ValidateReviewReport() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewReport() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewReportRejectsAmbiguousEvidencePath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "windows drive path", path: `C:\repo\internal\review\report_validation.go`},
		{name: "unc path", path: `\\server\share\report_validation.go`},
		{name: "relative current dir prefix", path: "./internal/review/report_validation.go"},
		{name: "non canonical with trailing slash", path: "internal/review/"},
		{name: "non canonical parent segment", path: "internal/review/../report_validation.go"},
		{name: "path with leading whitespace", path: " internal/review/report_validation.go"},
		{name: "path with trailing whitespace", path: "internal/review/report_validation.go "},
		{name: "path with whitespace-only", path: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newValidReviewReportForValidationTest()
			ref := newValidEvidenceRefForValidationTest()
			ref.Path = tt.path
			setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)

			err := ValidateReviewReport(report)
			if err == nil {
				t.Fatal("ValidateReviewReport() error = nil, want error")
			}
			if !strings.Contains(err.Error(), "checked_surfaces[0].evidence_refs[0].path") {
				t.Fatalf("ValidateReviewReport() error = %q, want path field", err.Error())
			}
		})
	}
}

func TestValidateReviewReportAllowsEvidencePathWithSpaces(t *testing.T) {
	report := newValidReviewReportForValidationTest()
	ref := newValidEvidenceRefForValidationTest()
	ref.Path = "docs/release notes.md"
	setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)

	if err := ValidateReviewReport(report); err != nil {
		t.Fatalf("ValidateReviewReport() error = %v, want nil", err)
	}
}

func TestValidateReviewReportValidatesEvidenceRefsAcrossAllContainers(t *testing.T) {
	tests := []struct {
		name        string
		report      func() ReviewReport
		errContains string
	}{
		{
			name: "top-level unverified_surfaces evidence_refs",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.UnverifiedSurfaces = []ReviewSurfaceCoverage{{
					SurfaceID:    "surface-1",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "unverified_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "top-level residual_risks evidence_refs",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.ResidualRisks = []ReviewResidualRisk{{
					Summary:      "risk",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "residual_risks[0].evidence_refs[0].path",
		},
		{
			name: "root_cause_group checked_surfaces evidence_refs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.RootCauseGroups[0].CheckedSurfaces = []ReviewSurfaceCoverage{{
					SurfaceID:    "surface-1",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "root_cause_groups[0].checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "root_cause_group unverified_surfaces evidence_refs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.RootCauseGroups[0].UnverifiedSurfaces = []ReviewSurfaceCoverage{{
					SurfaceID:    "surface-1",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "root_cause_groups[0].unverified_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "root_cause_group residual_risks evidence_refs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.RootCauseGroups[0].ResidualRisks = []ReviewResidualRisk{{
					Summary:      "risk",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "root_cause_groups[0].residual_risks[0].evidence_refs[0].path",
		},
		{
			name: "finding evidence_refs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.RootCauseGroups[0].Findings[0].EvidenceRefs = []ReviewEvidenceRef{ref}
				return report
			},
			errContains: "root_cause_groups[0].findings[0].evidence_refs[0].path",
		},
		{
			name: "finding checked_surfaces evidence_refs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.RootCauseGroups[0].Findings[0].CheckedSurfaces = []ReviewSurfaceCoverage{{
					SurfaceID:    "surface-1",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "root_cause_groups[0].findings[0].checked_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "finding unverified_surfaces evidence_refs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.RootCauseGroups[0].Findings[0].UnverifiedSurfaces = []ReviewSurfaceCoverage{{
					SurfaceID:    "surface-1",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "root_cause_groups[0].findings[0].unverified_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "finding residual_risks evidence_refs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				ref := newValidEvidenceRefForValidationTest()
				ref.Path = `..\outside.go`
				report.RootCauseGroups[0].Findings[0].ResidualRisks = []ReviewResidualRisk{{
					Summary:      "risk",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				}}
				return report
			},
			errContains: "root_cause_groups[0].findings[0].residual_risks[0].evidence_refs[0].path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewReport(tt.report())
			if err == nil {
				t.Fatal("ValidateReviewReport() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewReport() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewReportBlockedReasonByUnverifiedSurfaces(t *testing.T) {
	report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
	report.Summary = ""
	report.ProbeSummaries = nil
	report.UnverifiedSurfaces = []ReviewSurfaceCoverage{{SurfaceID: "surface-1", Summary: "未検証"}}

	if err := ValidateReviewReport(report); err != nil {
		t.Fatalf("ValidateReviewReport() error = %v, want nil", err)
	}
}
