package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestDecodeReviewSaturationCheckJSONAcceptsSaturatedCheck(t *testing.T) {
	check := newSaturatedReviewSaturationCheckForTest()
	data := mustMarshalReviewSaturationCheckForTest(t, check)

	got, err := DecodeReviewSaturationCheckJSON(data, newNoProbePlanScopeForTest(), newPlanAwareCleanReportForValidationTest())
	if err != nil {
		t.Fatalf("DecodeReviewSaturationCheckJSON() error = %v, want nil", err)
	}
	if got.Status != ReviewSaturationStatusSaturated {
		t.Fatalf("Status = %q, want %q", got.Status, ReviewSaturationStatusSaturated)
	}
}

func TestDecodeReviewSaturationCheckJSONRejectsUnknownFieldAndTrailingJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "unknown field",
			input:       `{"schema_version":"review_saturation_check.v1","status":"saturated","checked_summary":"checked","unknown":true}`,
			errContains: "unknown field",
		},
		{
			name:        "trailing JSON",
			input:       `{"schema_version":"review_saturation_check.v1","status":"saturated","checked_summary":"checked"} {}`,
			errContains: "single JSON value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeReviewSaturationCheckJSON([]byte(tt.input), newNoProbePlanScopeForTest(), newPlanAwareCleanReportForValidationTest())
			if err == nil {
				t.Fatal("DecodeReviewSaturationCheckJSON() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("DecodeReviewSaturationCheckJSON() error = %q, want %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewSaturationCheckNeedsRevisionForMissingRisk(t *testing.T) {
	check := ReviewSaturationCheck{
		SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
		Status:               ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "risk-1 was not handled.",
		MissingRiskIDs:       []string{"risk-1"},
		RevisionInstructions: "Classify risk-1 in scope_coverage.",
	}

	if err := ValidateReviewSaturationCheck(check, newNoProbePlanScopeForTest(), newPlanAwareCleanReportForValidationTest()); err != nil {
		t.Fatalf("ValidateReviewSaturationCheck() error = %v, want nil", err)
	}
}

func TestValidateReviewSaturationCheckNeedsRevisionForAdditionalFindingCandidate(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ProbeSummaries = []ReviewProbeSummary{
		{
			ProbeID: "probe-1",
			Mode:    domain.ReviewProbeHostReadOnly,
			Status:  domain.ReviewProbePassed,
			Commands: []ReviewProbeCommandSummary{
				{Command: "go", Status: domain.ReviewProbePassed},
			},
		},
	}
	check := ReviewSaturationCheck{
		SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
		Status:         ReviewSaturationStatusNeedsRevision,
		CheckedSummary: "A probe-backed candidate was omitted.",
		AdditionalFindingCandidates: []ReviewSaturationAdditionalFindingCandidate{
			{
				Summary: "probe output indicates a finding",
				EvidenceRefs: []ReviewEvidenceRef{
					{
						Kind:         ReviewEvidenceKindProbeCommand,
						ProbeID:      "probe-1",
						CommandIndex: ReviewCommandIndex(0),
					},
				},
				Reason: "The candidate is grounded in supplied probe output.",
			},
		},
		RevisionInstructions: "Add the probe-backed candidate if it remains valid.",
	}

	if err := ValidateReviewSaturationCheck(check, newNoProbePlanScopeForTest(), report); err != nil {
		t.Fatalf("ValidateReviewSaturationCheck() error = %v, want nil", err)
	}
}

func TestValidateReviewSaturationCheckRejectsInvalidStatusContracts(t *testing.T) {
	tests := []struct {
		name        string
		check       ReviewSaturationCheck
		errContains string
	}{
		{
			name: "saturated with missing risk",
			check: ReviewSaturationCheck{
				SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
				Status:         ReviewSaturationStatusSaturated,
				CheckedSummary: "checked",
				MissingRiskIDs: []string{"risk-1"},
			},
			errContains: "missing_risk_ids must be empty",
		},
		{
			name: "needs revision without item",
			check: ReviewSaturationCheck{
				SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
				Status:               ReviewSaturationStatusNeedsRevision,
				CheckedSummary:       "checked",
				RevisionInstructions: "Revise report.",
			},
			errContains: "requires missing_surface_ids",
		},
		{
			name: "needs revision without instructions",
			check: ReviewSaturationCheck{
				SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
				Status:         ReviewSaturationStatusNeedsRevision,
				CheckedSummary: "checked",
				MissingRiskIDs: []string{"risk-1"},
			},
			errContains: "revision_instructions must be non-empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewSaturationCheck(tt.check, newNoProbePlanScopeForTest(), newPlanAwareCleanReportForValidationTest())
			if err == nil {
				t.Fatal("ValidateReviewSaturationCheck() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewSaturationCheck() error = %q, want %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewSaturationCheckRejectsMissingIDsOutsidePass1Plan(t *testing.T) {
	tests := []struct {
		name        string
		check       ReviewSaturationCheck
		errContains string
	}{
		{
			name: "unknown surface",
			check: ReviewSaturationCheck{
				SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
				Status:               ReviewSaturationStatusNeedsRevision,
				CheckedSummary:       "surface missing",
				MissingSurfaceIDs:    []string{"surface-unknown"},
				RevisionInstructions: "Classify surface.",
			},
			errContains: "unknown Pass1 impact surface ID",
		},
		{
			name: "duplicate risk",
			check: ReviewSaturationCheck{
				SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
				Status:               ReviewSaturationStatusNeedsRevision,
				CheckedSummary:       "risk missing",
				MissingRiskIDs:       []string{"risk-1", "risk-1"},
				RevisionInstructions: "Classify risk.",
			},
			errContains: "duplicates candidate risk ID",
		},
		{
			name: "non canonical risk",
			check: ReviewSaturationCheck{
				SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
				Status:               ReviewSaturationStatusNeedsRevision,
				CheckedSummary:       "risk missing",
				MissingRiskIDs:       []string{"risk 1"},
				RevisionInstructions: "Classify risk.",
			},
			errContains: "must not include whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewSaturationCheck(tt.check, newNoProbePlanScopeForTest(), newPlanAwareCleanReportForValidationTest())
			if err == nil {
				t.Fatal("ValidateReviewSaturationCheck() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewSaturationCheck() error = %q, want %q", err.Error(), tt.errContains)
			}
		})
	}
}

func newSaturatedReviewSaturationCheckForTest() ReviewSaturationCheck {
	return ReviewSaturationCheck{
		SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
		Status:         ReviewSaturationStatusSaturated,
		CheckedSummary: "Final report covers Pass1 surfaces and risks.",
	}
}

func mustMarshalReviewSaturationCheckForTest(t *testing.T, check ReviewSaturationCheck) []byte {
	t.Helper()

	data, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}
