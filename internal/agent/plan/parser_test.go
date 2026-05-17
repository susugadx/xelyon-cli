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
					"files": ["internal/agent/plan_handoff.go", "internal/agent/plan_handoff_test.go"],
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
	assertStringSliceEqual(t, "PlanStep.Files", plan.Steps[0].Files, []string{"internal/agent/plan_handoff.go", "internal/agent/plan_handoff_test.go"})
	assertStringSliceEqual(t, "PlanStep.Verification", plan.Steps[0].Verification, []string{"go test ./internal/agent"})
}

func TestParsePlan_LegacyFormat(t *testing.T) {
	jsonStr := `{
		"summary": "Legacy plan",
		"steps": [
			{"id": 1, "description": "First step"},
			{"id": 2, "description": "Second step"}
		]
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if plan.Summary != "Legacy plan" {
		t.Errorf("Expected summary 'Legacy plan', got %q", plan.Summary)
	}
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}
	for i, step := range plan.Steps {
		if step.Status != "pending" {
			t.Errorf("Step %d: expected status 'pending', got %q", i, step.Status)
		}
	}
}

func TestParsePlan_InvalidJSON(t *testing.T) {
	jsonStr := `{invalid json}`

	_, err := ParsePlan(jsonStr)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestParsePlan_EmptySteps(t *testing.T) {
	jsonStr := `{"summary": "Empty", "steps": []}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(plan.Steps))
	}
}

func TestParsePlan_DependsOnStrings(t *testing.T) {
	// Gemini が depends_on を文字列配列で返すケース
	jsonStr := `{
		"plan": {
			"summary": "Test depends_on strings",
			"steps": [
				{"id": 1, "description": "Step 1"},
				{"id": 2, "description": "Step 2", "depends_on": ["1"]},
				{"id": 3, "description": "Step 3", "depends_on": ["1", "2"]}
			]
		}
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if len(plan.Steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(plan.Steps))
	}
	// Step 2: depends_on: ["1"] → [1]
	if len(plan.Steps[1].DependsOn) != 1 || plan.Steps[1].DependsOn[0] != 1 {
		t.Errorf("Step 2 DependsOn: expected [1], got %v", plan.Steps[1].DependsOn)
	}
	// Step 3: depends_on: ["1", "2"] → [1, 2]
	if len(plan.Steps[2].DependsOn) != 2 {
		t.Errorf("Step 3 DependsOn: expected 2 items, got %v", plan.Steps[2].DependsOn)
	}
}

func TestParsePlan_DependsOnSingleInt(t *testing.T) {
	// depends_on が単一 int で返されるケース
	jsonStr := `{
		"summary": "Test single int",
		"steps": [
			{"id": 1, "description": "Step 1"},
			{"id": 2, "description": "Step 2", "depends_on": 1}
		]
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if len(plan.Steps[1].DependsOn) != 1 || plan.Steps[1].DependsOn[0] != 1 {
		t.Errorf("Step 2 DependsOn: expected [1], got %v", plan.Steps[1].DependsOn)
	}
}

func TestParsePlan_DependsOnIntArray(t *testing.T) {
	// 従来通りの []int（回帰テスト）
	jsonStr := `{
		"summary": "Test int array",
		"steps": [
			{"id": 1, "description": "Step 1"},
			{"id": 2, "description": "Step 2", "depends_on": [1]}
		]
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if len(plan.Steps[1].DependsOn) != 1 || plan.Steps[1].DependsOn[0] != 1 {
		t.Errorf("Step 2 DependsOn: expected [1], got %v", plan.Steps[1].DependsOn)
	}
}

func TestParsePlan_DependsOnNull(t *testing.T) {
	// depends_on が null のケース
	jsonStr := `{
		"summary": "Test null",
		"steps": [
			{"id": 1, "description": "Step 1", "depends_on": null}
		]
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if len(plan.Steps[0].DependsOn) != 0 {
		t.Errorf("Step 1 DependsOn: expected empty, got %v", plan.Steps[0].DependsOn)
	}
}
