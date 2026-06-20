package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDecodeReviewReportModelOutputStrictJSONRejectsCleanCoverageGaps(t *testing.T) {
	model := newCleanReviewReportModelOutputForTest()
	model.CoverageGaps = []ReviewReportModelCoverageGap{
		{
			Surface:          "surface-1",
			Reason:           ReviewReportModelGapMissingEvidence,
			RecommendedCheck: "Run focused report validation.",
		},
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := DecodeReviewReportModelOutputStrictJSON(data); err == nil || !strings.Contains(err.Error(), "coverage_gaps") {
		t.Fatalf("DecodeReviewReportModelOutputStrictJSON() error = %v, want clean coverage_gaps rejection", err)
	}
}

func TestDecodeReviewReportModelOutputStrictJSONRejectsUnverifiedSuggestedFinding(t *testing.T) {
	model := newCleanReviewReportModelOutputForTest()
	model.Verdict = ReviewVerdictHasFindings
	model.SuggestedFindings = []ReviewReportModelSuggestedFinding{
		{
			ID:                   "finding-1",
			Severity:             ReviewReportModelSeverityP1,
			Status:               ReviewReportModelFindingStatus("unverified"),
			Confidence:           ReviewReportModelConfidenceLow,
			Title:                "Missing verification only",
			AffectedBehavior:     "The report would turn a coverage gap into a finding.",
			CausalChain:          "No causal chain was established.",
			EvidenceRefs:         []ReviewEvidenceRef{{Kind: ReviewEvidenceKindFile, Path: "internal/review/report/report_model_types.go"}},
			RemediationDirection: "Represent this as coverage_gaps and unverified scope.",
		},
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := DecodeReviewReportModelOutputStrictJSON(data); err == nil || !strings.Contains(err.Error(), `status must be one of "confirmed", "probable"`) {
		t.Fatalf("DecodeReviewReportModelOutputStrictJSON() error = %v, want unverified suggested finding rejection", err)
	}
}

func TestDecodeReviewReportModelOutputStrictJSONValidatesCoverageGapScopeCoverageLinkage(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*ReviewReportModelOutput)
		wantErrPart string
	}{
		{
			name: "unknown surface",
			mutate: func(model *ReviewReportModelOutput) {
				model.CoverageGaps[0].Surface = "surface-unknown"
			},
			wantErrPart: "must reference scope_coverage.reviewed_impact_surfaces",
		},
		{
			name: "checked surface",
			mutate: func(model *ReviewReportModelOutput) {
				model.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceChecked
			},
			wantErrPart: "requires scope_coverage.reviewed_impact_surfaces status",
		},
		{
			name: "finding surface",
			mutate: func(model *ReviewReportModelOutput) {
				model.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceFinding
			},
			wantErrPart: "requires scope_coverage.reviewed_impact_surfaces status",
		},
		{
			name: "unverified surface",
			mutate: func(model *ReviewReportModelOutput) {
				model.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
			},
		},
		{
			name: "residual risk surface",
			mutate: func(model *ReviewReportModelOutput) {
				model.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceResidualRisk
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newBlockedReviewReportModelOutputWithCoverageGapForTest()
			tt.mutate(&model)
			data, err := json.Marshal(model)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			_, err = DecodeReviewReportModelOutputStrictJSON(data)
			if tt.wantErrPart != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
					t.Fatalf("DecodeReviewReportModelOutputStrictJSON() error = %v, want %q", err, tt.wantErrPart)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeReviewReportModelOutputStrictJSON() error = %v, want nil", err)
			}
		})
	}
}

func TestReviewReportModelFindingStatusBypassDoesNotPromoteUnknownToPartiallyVerified(t *testing.T) {
	got := reviewVerificationStatusFromModelFindingStatus(ReviewReportModelFindingStatus("unverified"))
	if got != ReviewVerificationUnverified {
		t.Fatalf("reviewVerificationStatusFromModelFindingStatus(unverified) = %q, want %q", got, ReviewVerificationUnverified)
	}
}

func newCleanReviewReportModelOutputForTest() ReviewReportModelOutput {
	return ReviewReportModelOutput{
		SchemaVersion:             ReviewReportModelSchemaVersionV2,
		TargetKind:                TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: ReviewVerificationVerified,
		Verdict:                   ReviewVerdictClean,
		Summary:                   "No findings.",
		SuggestedFindings:         []ReviewReportModelSuggestedFinding{},
		CoverageGaps:              []ReviewReportModelCoverageGap{},
		ProbeSummaries:            []ReviewProbeSummary{},
		ScopeCoverage: &ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
				{SurfaceID: "surface-1", Status: ReviewReportImpactSurfaceChecked, Summary: "surface-1 checked."},
			},
			ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
				{RiskID: "risk-1", Status: ReviewReportCandidateRiskDismissed, Summary: "risk-1 dismissed."},
			},
		},
	}
}

func newBlockedReviewReportModelOutputWithCoverageGapForTest() ReviewReportModelOutput {
	model := newCleanReviewReportModelOutputForTest()
	model.OverallVerificationStatus = ReviewVerificationBlockedOrInconclusive
	model.Verdict = ReviewVerdictBlocked
	model.Summary = "Review blocked by missing evidence."
	model.CoverageGaps = []ReviewReportModelCoverageGap{
		{
			Surface:          "surface-1",
			Reason:           ReviewReportModelGapMissingEvidence,
			RecommendedCheck: "Run focused report validation.",
		},
	}
	model.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
	model.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 lacks evidence."
	model.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskUnverified
	model.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 lacks evidence."
	return model
}
