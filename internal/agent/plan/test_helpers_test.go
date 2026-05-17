package plan

import (
	"slices"
	"testing"
)

func mustExtractPlanJSON(t *testing.T, response string) string {
	t.Helper()

	result := ExtractPlanJSON(response)
	if result == "" {
		t.Fatalf("ExtractPlanJSON returned empty string for response %q", response)
	}
	return result
}

func mustParsePlanJSON(t *testing.T, jsonStr string) *Plan {
	t.Helper()

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("Extracted JSON is not valid: %v", err)
	}
	return plan
}

func assertPlanJSONNeedsRetry(t *testing.T, jsonStr string) {
	t.Helper()

	plan, err := ParsePlan(jsonStr)
	if err == nil && len(plan.Steps) > 0 {
		t.Fatalf("ParsePlan(%q) returned executable plan %#v, want retry candidate", jsonStr, plan)
	}
}

func assertStringSliceEqual(t *testing.T, label string, got, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}
