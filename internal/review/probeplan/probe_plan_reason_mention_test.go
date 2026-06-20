package probeplan

import (
	"strings"
	"testing"
)

func TestValidateReviewProbePlanReasonIDMentionContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "riskless reason rejects surface id prefix match",
			plan: func() ReviewProbePlan {
				plan := markReviewProbePlanRisklessForTest(newValidReviewProbePlanForTest())
				plan = appendReviewProbePlanCheckedSurfaceForReasonMentionTest(plan, "surface-10")
				plan.Probes[0].SurfaceIDs = []string{"surface-1", "surface-10"}
				plan.NoCandidateRiskReason = "surface-10 was checked from available evidence and leaves no material candidate risk."
				return plan
			},
			errContains: "no_candidate_risk_reason",
		},
		{
			name: "riskless reason accepts exact prefix-related surface ids",
			plan: func() ReviewProbePlan {
				plan := markReviewProbePlanRisklessForTest(newValidReviewProbePlanForTest())
				plan = appendReviewProbePlanCheckedSurfaceForReasonMentionTest(plan, "surface-10")
				plan.Probes[0].SurfaceIDs = []string{"surface-1", "surface-10"}
				plan.NoCandidateRiskReason = "surface-1, surface-10 were checked from available evidence and leave no material candidate risk."
				return plan
			},
		},
		{
			name: "no-probe reason rejects surface id prefix match",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan = appendReviewProbePlanCheckedSurfaceForReasonMentionTest(plan, "surface-10")
				plan.NoProbeReason = "surface-10 and risk-1 are checked by existing evidence."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "no-probe reason rejects risk id prefix match",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan = appendReviewProbePlanCheckedRiskForReasonMentionTest(plan, "risk-10")
				plan.NoProbeReason = "surface-1 and risk-10 are checked by existing evidence."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "no-probe reason accepts exact prefix-related surface and risk ids",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan = appendReviewProbePlanCheckedSurfaceForReasonMentionTest(plan, "surface-10")
				plan = appendReviewProbePlanCheckedRiskForReasonMentionTest(plan, "risk-10")
				plan.NoProbeReason = "surface-1, surface-10, risk-1, and risk-10 are checked by existing evidence."
				return plan
			},
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

func appendReviewProbePlanCheckedSurfaceForReasonMentionTest(plan ReviewProbePlan, id string) ReviewProbePlan {
	plan.ImpactSurfaces = append(plan.ImpactSurfaces, ReviewProbeImpactSurface{
		ID:              id,
		Summary:         "Existing evidence covers " + id + ".",
		Category:        ReviewProbeImpactSurfaceChangedFile,
		EvidenceSummary: "Existing evidence is sufficient.",
		Status:          ReviewProbeImpactSurfaceChecked,
		Reason:          "Existing evidence covers " + id + ".",
	})
	return plan
}

func appendReviewProbePlanCheckedRiskForReasonMentionTest(plan ReviewProbePlan, id string) ReviewProbePlan {
	surfaceID := "surface-1"
	if len(plan.ImpactSurfaces) > 0 {
		surfaceID = plan.ImpactSurfaces[0].ID
	}
	plan.CandidateRisks = append(plan.CandidateRisks, ReviewProbeCandidateRisk{
		ID:                   id,
		Summary:              "Existing evidence already checks " + id + ".",
		Severity:             ReviewGroupSeverityLow,
		SurfaceIDs:           []string{surfaceID},
		EvidenceSummary:      "Existing evidence is sufficient.",
		VerificationStrategy: "No additional probe is needed for " + id + ".",
		Status:               ReviewProbeCandidateRiskCheckedByEvidence,
	})
	return plan
}
