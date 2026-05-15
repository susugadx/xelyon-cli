package review

import (
	"strings"
	"testing"
)

func TestValidateReviewProbePlanProbeLinkageContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "probe requires at least one surface or risk link",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].SurfaceIDs = nil
				plan.Probes[0].RiskIDs = nil
				return plan
			},
			errContains: "probes[0].surface_ids",
		},
		{
			name: "probe rejects unknown surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].SurfaceIDs = []string{"missing-surface"}
				return plan
			},
			errContains: "probes[0].surface_ids[0]",
		},
		{
			name: "probe rejects unknown risk id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].RiskIDs = []string{"missing-risk"}
				return plan
			},
			errContains: "probes[0].risk_ids[0]",
		},
		{
			name: "needs_probe surface must be referenced by a probe",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].SurfaceIDs = nil
				return plan
			},
			errContains: "impact_surfaces[0].id",
		},
		{
			name: "unverified surface must be referenced by a probe",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceUnverified
				plan.Probes[0].SurfaceIDs = nil
				return plan
			},
			errContains: "impact_surfaces[0].id",
		},
		{
			name: "needs_probe risk must be referenced by a probe",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].RiskIDs = nil
				return plan
			},
			errContains: "candidate_risks[0].id",
		},
		{
			name: "unverified risk must be referenced by a probe",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskUnverified
				plan.Probes[0].RiskIDs = nil
				return plan
			},
			errContains: "candidate_risks[0].id",
		},
		{
			name: "checked surfaces and checked_by_evidence risks do not require probe links",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces = append(plan.ImpactSurfaces, ReviewProbeImpactSurface{
					ID:              "surface-checked",
					Summary:         "Existing evidence covers this surface.",
					Category:        ReviewProbeImpactSurfaceChangedFile,
					EvidenceSummary: "Existing evidence is sufficient.",
					Status:          ReviewProbeImpactSurfaceChecked,
					Reason:          "No additional probe is needed for surface-checked.",
				})
				plan.CandidateRisks = append(plan.CandidateRisks, ReviewProbeCandidateRisk{
					ID:                   "risk-checked",
					Summary:              "Existing evidence already checks this risk.",
					Severity:             ReviewGroupSeverityLow,
					SurfaceIDs:           []string{"surface-checked"},
					EvidenceSummary:      "Existing evidence is sufficient.",
					VerificationStrategy: "No additional probe is needed for risk-checked.",
					Status:               ReviewProbeCandidateRiskCheckedByEvidence,
				})
				return plan
			},
		},
		{
			name: "valid linked probe plan",
			plan: newValidReviewProbePlanForTest,
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

func TestValidateReviewProbePlanRequiredScopeTextContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "surface summary must be non-empty after trim",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Summary = " \t"
				return plan
			},
			errContains: "impact_surfaces[0].summary",
		},
		{
			name: "surface reason must be non-empty after trim",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Reason = " \n"
				return plan
			},
			errContains: "impact_surfaces[0].reason",
		},
		{
			name: "risk summary must be non-empty after trim",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].Summary = " "
				return plan
			},
			errContains: "candidate_risks[0].summary",
		},
		{
			name: "risk verification_strategy must be non-empty after trim",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].VerificationStrategy = "\t"
				return plan
			},
			errContains: "candidate_risks[0].verification_strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewProbePlan(tt.plan())
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}
