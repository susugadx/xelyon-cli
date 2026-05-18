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

	if plan, err := ParsePlan(jsonStr); err == nil {
		t.Fatalf("ParsePlan(%q) returned plan %#v, want retry parse error", jsonStr, plan)
	}
}

func assertStringSliceEqual(t *testing.T, label string, got, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}
