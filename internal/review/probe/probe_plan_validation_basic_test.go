package probe

import (
	"strings"
	"testing"
)

func TestValidateReviewProbePlanBasicContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "valid plan",
			plan: newValidReviewProbePlanForTest,
		},
		{
			name: "invalid schema_version",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.SchemaVersion = ReviewProbePlanSchemaVersionV1
				return plan
			},
			errContains: "schema_version",
		},
		{
			name: "invalid target_kind",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.TargetKind = TargetKind("workspace_snapshot")
				return plan
			},
			errContains: "target_kind",
		},
		{
			name: "duplicate probe id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes = append(plan.Probes, plan.Probes[0])
				return plan
			},
			errContains: "probes[1].id",
		},
		{
			name: "probe id with leading whitespace",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].ID = " probe-1"
				return plan
			},
			errContains: "probes[0].id",
		},
		{
			name: "probe id with internal whitespace",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].ID = "probe 1"
				return plan
			},
			errContains: "probes[0].id",
		},
		{
			name: "invalid mode",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].Mode = ReviewProbeMode("unknown")
				return plan
			},
			errContains: "probes[0].mode",
		},
		{
			name: "probes empty requires no_probe_reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes = nil
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "probes non-empty rejects no_probe_reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.NoProbeReason = "All relevant surfaces were already checked."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "too many probes",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes = make([]ReviewPlannedProbe, 0, MaxReviewProbePlanProbes+1)
				for i := 0; i < MaxReviewProbePlanProbes+1; i++ {
					probe := newValidReviewProbePlanForTest().Probes[0]
					probe.ID = "probe-" + string(rune('a'+i))
					plan.Probes = append(plan.Probes, probe)
				}
				return plan
			},
			errContains: "probes",
		},
		{
			name: "negative timeout_seconds",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].TimeoutSeconds = -1
				return plan
			},
			errContains: "timeout_seconds",
		},
		{
			name: "too large timeout_seconds",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].TimeoutSeconds = MaxReviewProbePlanTimeoutSeconds + 1
				return plan
			},
			errContains: "timeout_seconds",
		},
		{
			name: "negative max_output_bytes",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].MaxOutputBytes = -1
				return plan
			},
			errContains: "max_output_bytes",
		},
		{
			name: "too large max_output_bytes",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].MaxOutputBytes = MaxReviewProbePlanMaxOutputBytes + 1
				return plan
			},
			errContains: "max_output_bytes",
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
