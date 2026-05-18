package plan

import "testing"

func TestParsePlan_V2FormatPreservesHandoffFields(t *testing.T) {
	jsonStr := `{
		"plan": {
			"summary": "Test plan",
			"findings": ["plan_handoff.go owns implementation handoff"],
			"evidence": ["internal/agent/plan_handoff.go: normalModeInput"],
			"constraints": ["Do not carry raw investigation history"],
			"steps": [
				{"id": 1, "description": "Update handoff files"}
			]
		}
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	assertStringSliceEqual(t, "Plan.Findings", plan.Findings, []string{"plan_handoff.go owns implementation handoff"})
	assertStringSliceEqual(t, "Plan.Evidence", plan.Evidence, []string{"internal/agent/plan_handoff.go: normalModeInput"})
	assertStringSliceEqual(t, "Plan.Constraints", plan.Constraints, []string{"Do not carry raw investigation history"})
}

func TestParsePlan_V2FormatAllowsHandoffFieldsWithoutSteps(t *testing.T) {
	jsonStr := `{
		"plan": {
			"findings": ["No code change is needed"],
			"evidence": ["README.md documents the existing behavior"],
			"constraints": ["Keep current CLI output unchanged"],
			"steps": []
		}
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	assertStringSliceEqual(t, "Plan.Findings", plan.Findings, []string{"No code change is needed"})
	assertStringSliceEqual(t, "Plan.Evidence", plan.Evidence, []string{"README.md documents the existing behavior"})
	assertStringSliceEqual(t, "Plan.Constraints", plan.Constraints, []string{"Keep current CLI output unchanged"})
	if len(plan.Steps) != 0 {
		t.Fatalf("len(Plan.Steps) = %d, want 0", len(plan.Steps))
	}
}

func TestExtractPlanJSON_WrapperHandoffFieldsArePlanShape(t *testing.T) {
	response := `Here is a no-op plan:
{"plan":{"findings":["Existing behavior already satisfies the request"],"evidence":["README.md documents it"],"constraints":["Do not change CLI output"],"steps":[]}}
Done.`

	parsed := mustParsePlanJSON(t, mustExtractPlanJSON(t, response))
	assertStringSliceEqual(t, "parsed findings", parsed.Findings, []string{"Existing behavior already satisfies the request"})
	assertStringSliceEqual(t, "parsed evidence", parsed.Evidence, []string{"README.md documents it"})
	assertStringSliceEqual(t, "parsed constraints", parsed.Constraints, []string{"Do not change CLI output"})
}
