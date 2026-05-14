package review

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeReviewReportJSONValidReport(t *testing.T) {
	data := mustMarshalReviewReportForDecodeTest(t, newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified))

	report, err := DecodeReviewReportJSON(data)
	if err != nil {
		t.Fatalf("DecodeReviewReportJSON() error = %v, want nil", err)
	}
	if err := ValidateReviewReport(report); err != nil {
		t.Fatalf("ValidateReviewReport() error = %v, want nil", err)
	}
	if got, want := report.SchemaVersion, ReviewReportSchemaVersionV2; got != want {
		t.Fatalf("SchemaVersion = %q, want %q", got, want)
	}
	if got, want := len(report.RootCauseGroups), 1; got != want {
		t.Fatalf("len(RootCauseGroups) = %d, want %d", got, want)
	}
}

func TestDecodeReviewReportJSONRejectsUnknownFieldsAndTrailingToken(t *testing.T) {
	base := mustMarshalReviewReportForDecodeTest(t, newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified))

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "unknown top-level field",
			data: appendReviewReportTopLevelFieldForDecodeTest(t, base, `"unexpected":true`),
		},
		{
			name: "unknown nested field",
			data: replaceReviewReportJSONForDecodeTest(t, base, `"root_cause_groups":[{`, `"root_cause_groups":[{"unexpected":true,`),
		},
		{
			name: "trailing JSON token",
			data: []byte(string(base) + ` {}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeReviewReportJSON(tt.data)
			if err == nil {
				t.Fatal("DecodeReviewReportJSON() error = nil, want error")
			}
		})
	}
}

func TestDecodeReviewReportJSONRejectsInvalidValidatedReport(t *testing.T) {
	report := newValidReviewReportForValidationTest()
	report.SchemaVersion = ReviewReportSchemaVersionV1
	data := mustMarshalReviewReportForDecodeTest(t, report)

	_, err := DecodeReviewReportJSON(data)
	if err == nil {
		t.Fatal("DecodeReviewReportJSON() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("DecodeReviewReportJSON() error = %q, want schema_version", err.Error())
	}
}

func TestDecodeReviewReportJSONAcceptsComputedSummaryInFinalReport(t *testing.T) {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.ComputedSummary = &ReviewReportComputedSummary{
		RootCauseGroupCount: 1,
		FindingCount:        1,
	}
	data := mustMarshalReviewReportForDecodeTest(t, report)

	got, err := DecodeReviewReportJSON(data)
	if err != nil {
		t.Fatalf("DecodeReviewReportJSON() error = %v, want nil", err)
	}
	if got.ComputedSummary == nil {
		t.Fatal("ComputedSummary = nil, want decoded final report computed_summary")
	}
	if got.ComputedSummary.FindingCount != 1 {
		t.Fatalf("ComputedSummary.FindingCount = %d, want 1", got.ComputedSummary.FindingCount)
	}
}

func TestDecodeReviewReportJSONRejectsScopeCoverageSemanticContract(t *testing.T) {
	report := newCleanReportForValidationTest(ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskFinding
	data := mustMarshalReviewReportForDecodeTest(t, report)

	_, err := DecodeReviewReportJSON(data)
	if err == nil {
		t.Fatal("DecodeReviewReportJSON() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `verdict "clean" requires scope_coverage.reviewed_candidate_risks[0].status`) {
		t.Fatalf("DecodeReviewReportJSON() error = %q, want scope coverage verdict error", err.Error())
	}
}

func TestDecodeReviewReportJSONRejectsIncompleteNestedReportEntries(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		errContains string
	}{
		{
			name: "blocked report with empty residual risk",
			data: replaceReviewReportJSONForDecodeTest(
				t,
				mustMarshalReviewReportForDecodeTest(t, newBlockedReportWithResidualRiskForDecodeTest()),
				`"residual_risks":[{"summary":"blocked reason"}]`,
				`"residual_risks":[{}]`,
			),
			errContains: "residual_risks[0].summary",
		},
		{
			name:        "has_findings report with empty finding",
			data:        mustMarshalReviewReportForDecodeTest(t, newHasFindingsReportWithEmptyFindingForDecodeTest()),
			errContains: "root_cause_groups[0].findings[0].title",
		},
		{
			name: "clean report with empty checked surface",
			data: replaceReviewReportJSONForDecodeTest(
				t,
				mustMarshalReviewReportForDecodeTest(t, newCleanReportWithCheckedSurfaceForDecodeTest()),
				`"checked_surfaces":[{"surface_id":"surface-1","summary":"checked"}]`,
				`"checked_surfaces":[{}]`,
			),
			errContains: "checked_surfaces[0].surface_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeReviewReportJSON(tt.data)
			if err == nil {
				t.Fatal("DecodeReviewReportJSON() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("DecodeReviewReportJSON() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func mustMarshalReviewReportForDecodeTest(t *testing.T, report ReviewReport) []byte {
	t.Helper()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func newBlockedReportWithResidualRiskForDecodeTest() ReviewReport {
	report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
	report.Summary = ""
	report.ResidualRisks = []ReviewResidualRisk{{Summary: "blocked reason"}}
	return report
}

func newCleanReportWithCheckedSurfaceForDecodeTest() ReviewReport {
	report := newCleanReportForValidationTest(ReviewVerificationPartiallyVerified)
	report.CheckedSurfaces = []ReviewSurfaceCoverage{{SurfaceID: "surface-1", Summary: "checked"}}
	return report
}

func newHasFindingsReportWithEmptyFindingForDecodeTest() ReviewReport {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.RootCauseGroups[0].Findings[0] = ReviewFinding{}
	return report
}

func appendReviewReportTopLevelFieldForDecodeTest(t *testing.T, data []byte, field string) []byte {
	t.Helper()

	text := string(data)
	if !strings.HasSuffix(text, "}") {
		t.Fatalf("report JSON must end with object close: %q", text)
	}
	return []byte(strings.TrimSuffix(text, "}") + "," + field + "}")
}

func replaceReviewReportJSONForDecodeTest(t *testing.T, data []byte, old, new string) []byte {
	t.Helper()

	text := string(data)
	replaced := strings.Replace(text, old, new, 1)
	if replaced == text {
		t.Fatalf("report JSON did not contain %q: %s", old, text)
	}
	return []byte(replaced)
}
