package probeplan

import (
	"strings"
	"testing"
)

func TestValidateReviewProbePlanIDTokenContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "allows ASCII letters digits hyphen and underscore",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].ID = "surface_1-A"
				plan.CandidateRisks[0].ID = "risk_1-A"
				plan.CandidateRisks[0].SurfaceIDs = []string{"surface_1-A"}
				plan.Probes[0].ID = "probe_1-A"
				plan.Probes[0].SurfaceIDs = []string{"surface_1-A"}
				plan.Probes[0].RiskIDs = []string{"risk_1-A"}
				plan.Probes[0].Purpose = "Confirm or falsify risk_1-A for surface_1-A by running focused review tests."
				return plan
			},
		},
		{
			name: "rejects punctuated surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].ID = "surface.1"
				return plan
			},
			errContains: "impact_surfaces[0].id",
		},
		{
			name: "rejects punctuated risk id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].ID = "risk.1"
				return plan
			},
			errContains: "candidate_risks[0].id",
		},
		{
			name: "rejects punctuated probe id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].ID = "probe.1"
				return plan
			},
			errContains: "probes[0].id",
		},
		{
			name: "rejects punctuated surface reference before mention matching",
			plan: func() ReviewProbePlan {
				plan := markReviewProbePlanRisklessForTest(newValidReviewProbePlanForTest())
				plan.ImpactSurfaces[0].ID = "surface.1"
				plan.Probes[0].SurfaceIDs = []string{"surface.1"}
				plan.NoCandidateRiskReason = "surface.1.extra was checked from available evidence and leaves no material candidate risk."
				return plan
			},
			errContains: "impact_surfaces[0].id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewProbePlan(tt.plan())
			if tt.errContains == "" {
				if err != nil {
					t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}
