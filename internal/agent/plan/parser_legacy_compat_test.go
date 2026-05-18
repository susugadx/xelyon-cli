package plan

import "testing"

func TestParsePlan_LegacyFormatIgnoresNonObjectPlanField(t *testing.T) {
	jsonStr := `{
		"plan": "rollout",
		"summary": "Legacy plan",
		"steps": [
			{"id": 1, "description": "First step"}
		]
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}
	if plan.Summary != "Legacy plan" {
		t.Fatalf("Plan.Summary = %q, want legacy summary", plan.Summary)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Description != "First step" {
		t.Fatalf("Plan.Steps = %#v, want legacy step", plan.Steps)
	}
}

func TestParsePlan_LegacyFormatIgnoresNonWrapperPlanObject(t *testing.T) {
	jsonStr := `{
		"plan": {"name": "rollout"},
		"summary": "Legacy plan",
		"steps": [
			{"id": 1, "description": "First step"}
		]
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}
	if plan.Summary != "Legacy plan" {
		t.Fatalf("Plan.Summary = %q, want legacy summary", plan.Summary)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Description != "First step" {
		t.Fatalf("Plan.Steps = %#v, want legacy step", plan.Steps)
	}
}
