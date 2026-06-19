package probe

import "testing"

func TestDecodeReviewProbePlanJSONValidPlan(t *testing.T) {
	data := mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest())

	plan, err := DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := ValidateReviewProbePlan(plan); err != nil {
		t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
	}
	if got, want := plan.SchemaVersion, ReviewProbePlanSchemaVersionV2; got != want {
		t.Fatalf("SchemaVersion = %q, want %q", got, want)
	}
	if got, want := len(plan.Probes), 1; got != want {
		t.Fatalf("len(Probes) = %d, want %d", got, want)
	}
}

func TestDecodeReviewProbePlanJSONRejectsUnknownFieldsAndTrailingToken(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "unknown top-level field",
			json: mustMarshalMutatedReviewProbePlanJSONForTest(t, func(plan map[string]any) {
				plan["unexpected"] = true
			}),
		},
		{
			name: "unknown nested probe field",
			json: mustMarshalMutatedReviewProbePlanJSONForTest(t, func(plan map[string]any) {
				mustFirstReviewProbePlanProbeJSONForTest(t, plan)["unexpected"] = true
			}),
		},
		{
			name: "unknown nested command field",
			json: mustMarshalMutatedReviewProbePlanJSONForTest(t, func(plan map[string]any) {
				mustFirstReviewProbePlanCommandJSONForTest(t, plan)["unexpected"] = true
			}),
		},
		{
			name: "trailing JSON token",
			json: string(mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest())) + ` {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeReviewProbePlanJSON([]byte(tt.json))
			if err == nil {
				t.Fatal("DecodeReviewProbePlanJSON() error = nil, want error")
			}
		})
	}
}
