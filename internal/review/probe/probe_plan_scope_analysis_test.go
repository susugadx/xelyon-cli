package probe

import (
	"strings"
	"testing"
)

func TestValidateReviewProbePlanScopeAnalysisContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "missing impact_surfaces",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces = nil
				return plan
			},
			errContains: "impact_surfaces",
		},
		{
			name: "duplicate surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces = append(plan.ImpactSurfaces, plan.ImpactSurfaces[0])
				return plan
			},
			errContains: "impact_surfaces[1].id",
		},
		{
			name: "duplicate risk id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks = append(plan.CandidateRisks, plan.CandidateRisks[0])
				return plan
			},
			errContains: "candidate_risks[1].id",
		},
		{
			name: "risk references unknown surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].SurfaceIDs = []string{"missing-surface"}
				return plan
			},
			errContains: "candidate_risks[0].surface_ids[0]",
		},
		{
			name: "risk requires at least one surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].SurfaceIDs = nil
				return plan
			},
			errContains: "candidate_risks[0].surface_ids",
		},
		{
			name: "unknown surface category",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Category = ReviewProbeImpactSurfaceCategory("unknown")
				return plan
			},
			errContains: "impact_surfaces[0].category",
		},
		{
			name: "unknown surface status",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceStatus("unknown")
				return plan
			},
			errContains: "impact_surfaces[0].status",
		},
		{
			name: "unknown risk severity",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].Severity = ReviewGroupSeverity("unknown")
				return plan
			},
			errContains: "candidate_risks[0].severity",
		},
		{
			name: "unknown risk status",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskStatus("unknown")
				return plan
			},
			errContains: "candidate_risks[0].status",
		},
		{
			name: "surface requires evidence summary or refs",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = nil
				return plan
			},
			errContains: "impact_surfaces[0] requires evidence_summary or evidence_refs",
		},
		{
			name: "risk requires evidence summary or refs",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].EvidenceSummary = ""
				plan.CandidateRisks[0].EvidenceRefs = nil
				return plan
			},
			errContains: "candidate_risks[0] requires evidence_summary or evidence_refs",
		},
		{
			name: "scope evidence rejects probe kind",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindProbe}}
				return plan
			},
			errContains: "impact_surfaces[0].evidence_refs[0].kind",
		},
		{
			name: "scope evidence rejects probe_id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindGitStatus, ProbeID: "probe-1"}}
				return plan
			},
			errContains: "impact_surfaces[0].evidence_refs[0].probe_id",
		},
		{
			name: "scope evidence rejects command_index",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindGitStatus, CommandIndex: ReviewCommandIndex(0)}}
				return plan
			},
			errContains: "impact_surfaces[0].evidence_refs[0].command_index",
		},
		{
			name: "scope evidence accepts file ref",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{
					Kind: ReviewEvidenceKindFile,
					Path: "internal/review/probe_plan.go",
					Line: 12,
				}}
				return plan
			},
		},
		{
			name: "candidate risks may be empty",
			plan: func() ReviewProbePlan {
				return markReviewProbePlanRisklessForTest(newValidReviewProbePlanForTest())
			},
		},
		{
			name: "candidate risks empty requires reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks = nil
				plan.Probes[0].RiskIDs = nil
				return plan
			},
			errContains: "no_candidate_risk_reason",
		},
		{
			name: "candidate risks empty rejects reason missing surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks = nil
				plan.Probes[0].RiskIDs = nil
				plan.NoCandidateRiskReason = "The diff evidence leaves no material candidate risk."
				return plan
			},
			errContains: "no_candidate_risk_reason",
		},
		{
			name: "candidate risks non-empty rejects no_candidate_risk_reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.NoCandidateRiskReason = "surface-1 has no material candidate risk."
				return plan
			},
			errContains: "no_candidate_risk_reason",
		},
		{
			name: "no-probe rejects needs_probe surface",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceNeedsProbe
				return plan
			},
			errContains: "impact_surfaces[0].status",
		},
		{
			name: "no-probe rejects unverified risk",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskUnverified
				return plan
			},
			errContains: "candidate_risks[0].status",
		},
		{
			name: "no-probe rejects reason missing surface id",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.NoProbeReason = "risk-1 is checked by existing evidence."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "no-probe rejects reason missing risk id",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.NoProbeReason = "surface-1 is checked by existing evidence."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "no-probe accepts fully checked scope",
			plan: newNoProbeReviewProbePlanForTest,
		},
		{
			name: "no-probe accepts riskless fully checked scope",
			plan: newNoProbeRisklessReviewProbePlanForTest,
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
