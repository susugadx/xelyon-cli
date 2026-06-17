package plan

import "testing"

func TestParsePlan_V2Format(t *testing.T) {
	jsonStr := `{
		"plan": {
			"summary": "Test plan",
			"steps": [
				{"id": 1, "description": "Step 1", "tools": ["read_file"]},
				{"id": 2, "description": "Step 2", "tools": ["write_file"], "depends_on": [1]}
			]
		}
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if plan.Summary != "Test plan" {
		t.Errorf("Expected summary 'Test plan', got %q", plan.Summary)
	}
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Status != "pending" {
		t.Errorf("Expected status 'pending', got %q", plan.Steps[0].Status)
	}
}

func TestParsePlan_V2FormatPreservesStepFiles(t *testing.T) {
	jsonStr := `{
		"plan": {
			"summary": "Test plan",
			"steps": [
				{
					"id": 1,
					"description": "Update handoff files",
					"purpose": "Keep implementation handoff readable",
					"tools": ["str_replace"],
					"files": ["internal/agent/plan/handoff.go", "internal/agent/plan/handoff_test.go"],
					"verification": ["go test ./internal/agent"]
				}
			]
		}
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if plan.Steps[0].Purpose != "Keep implementation handoff readable" {
		t.Fatalf("PlanStep.Purpose = %q, want purpose", plan.Steps[0].Purpose)
	}
	assertStringSliceEqual(t, "PlanStep.Files", plan.Steps[0].Files, []string{"internal/agent/plan/handoff.go", "internal/agent/plan/handoff_test.go"})
	assertStringSliceEqual(t, "PlanStep.Verification", plan.Steps[0].Verification, []string{"go test ./internal/agent"})
}

func TestParsePlan_V2FormatInvalidStepsReturnsError(t *testing.T) {
	jsonStr := `{"plan":{"summary":"Fix","steps":{"id":1,"description":"Do it"}}}`

	if _, err := ParsePlan(jsonStr); err == nil {
		t.Fatal("ParsePlan() error = nil, want wrapper schema error")
	}
}

func TestParsePlan_V2FormatEmptyWrapperReturnsError(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "empty summary",
			json: `{"plan":{"summary":""}}`,
		},
		{
			name: "whitespace summary",
			json: `{"plan":{"summary":" ","steps":[]}}`,
		},
		{
			name: "empty handoff fields",
			json: `{"plan":{"findings":[""],"evidence":[],"constraints":[],"steps":[]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParsePlan(tt.json); err == nil {
				t.Fatal("ParsePlan() error = nil, want empty wrapper error")
			}
		})
	}
}
