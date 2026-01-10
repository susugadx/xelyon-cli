package agent

import (
	"strings"
	"testing"
)

func TestParsePlan(t *testing.T) {
	jsonStr := `{
		"steps": [
			{
				"id": 1,
				"description": "Read main.go",
				"tools": ["read_file"],
				"depends_on": [],
				"parallel": false
			},
			{
				"id": 2,
				"description": "Run tests",
				"tools": ["bash"],
				"depends_on": [1],
				"parallel": false
			}
		]
	}`

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	step1 := plan.Steps[0]
	if step1.ID != 1 {
		t.Errorf("Expected step ID 1, got %d", step1.ID)
	}
	if step1.Description != "Read main.go" {
		t.Errorf("Expected description 'Read main.go', got '%s'", step1.Description)
	}
	if step1.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", step1.Status)
	}
	if len(step1.Tools) != 1 || step1.Tools[0] != "read_file" {
		t.Errorf("Expected tools ['read_file'], got %v", step1.Tools)
	}

	step2 := plan.Steps[1]
	if len(step2.DependsOn) != 1 || step2.DependsOn[0] != 1 {
		t.Errorf("Expected depends_on [1], got %v", step2.DependsOn)
	}
}

func TestExtractPlanJSON_WithCodeBlock(t *testing.T) {
	response := `Here is the plan:

` + "```json" + `
{
  "steps": [
    {
      "id": 1,
      "description": "Test step",
      "tools": ["bash"],
      "depends_on": [],
      "parallel": false
    }
  ]
}
` + "```" + `

This is the execution plan.`

	jsonStr, err := ExtractPlanJSON(response)
	if err != nil {
		t.Fatalf("Failed to extract JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"steps"`) {
		t.Errorf("Extracted JSON does not contain 'steps': %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"Test step"`) {
		t.Errorf("Extracted JSON does not contain 'Test step': %s", jsonStr)
	}
}

func TestExtractPlanJSON_WithNewlines(t *testing.T) {
	response := `I will create a plan:

{
  "steps": [
    {
      "id": 1,
      "description": "Step with newlines",
      "tools": ["read_file"],
      "depends_on": [],
      "parallel": true
    }
  ]
}

Done!`

	jsonStr, err := ExtractPlanJSON(response)
	if err != nil {
		t.Fatalf("Failed to extract JSON with newlines: %v", err)
	}

	if !strings.Contains(jsonStr, `"steps"`) {
		t.Errorf("Extracted JSON does not contain 'steps': %s", jsonStr)
	}
}

func TestExtractPlanJSON_NoJSON(t *testing.T) {
	response := "This response contains no JSON at all."

	_, err := ExtractPlanJSON(response)
	if err == nil {
		t.Error("Expected error for response without JSON, but got nil")
	}
}

func TestPlan_CanExecute(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "pending", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
		},
	}

	// Step 1 は依存なしで実行可能
	if !plan.CanExecute(1) {
		t.Error("Expected step 1 to be executable")
	}

	// Step 2 は Step 1 が pending なので実行不可
	if plan.CanExecute(2) {
		t.Error("Expected step 2 to be not executable (depends on step 1)")
	}

	// Step 1 を完了にする
	plan.UpdateStatus(1, "completed", "Success")

	// Step 2, 3 が実行可能になる
	if !plan.CanExecute(2) {
		t.Error("Expected step 2 to be executable after step 1 completed")
	}
	if !plan.CanExecute(3) {
		t.Error("Expected step 3 to be executable after step 1 completed")
	}
}

func TestPlan_GetParallelSteps(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}, Parallel: false},
			{ID: 2, Status: "pending", DependsOn: []int{1}, Parallel: true},
			{ID: 3, Status: "pending", DependsOn: []int{1}, Parallel: true},
			{ID: 4, Status: "pending", DependsOn: []int{2, 3}, Parallel: false},
		},
	}

	parallelSteps := plan.GetParallelSteps()
	if len(parallelSteps) != 2 {
		t.Errorf("Expected 2 parallel steps, got %d", len(parallelSteps))
	}

	// Step 2 と 3 が並列実行可能
	if !contains(parallelSteps, 2) || !contains(parallelSteps, 3) {
		t.Errorf("Expected steps 2 and 3 to be parallel, got %v", parallelSteps)
	}
}

func TestPlan_GetNextStep(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{2}},
		},
	}

	nextStep := plan.GetNextStep()
	if nextStep != 2 {
		t.Errorf("Expected next step to be 2, got %d", nextStep)
	}

	plan.UpdateStatus(2, "completed", "Success")
	nextStep = plan.GetNextStep()
	if nextStep != 3 {
		t.Errorf("Expected next step to be 3, got %d", nextStep)
	}

	plan.UpdateStatus(3, "completed", "Success")
	nextStep = plan.GetNextStep()
	if nextStep != -1 {
		t.Errorf("Expected next step to be -1 (all completed), got %d", nextStep)
	}
}

func TestPlan_IsCompleted(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if plan.IsCompleted() {
		t.Error("Expected plan to be not completed")
	}

	plan.UpdateStatus(2, "completed", "Success")
	if !plan.IsCompleted() {
		t.Error("Expected plan to be completed")
	}
}

func TestPlan_HasFailed(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if plan.HasFailed() {
		t.Error("Expected plan to have no failures")
	}

	plan.UpdateStatus(2, "failed", "Error occurred")
	if !plan.HasFailed() {
		t.Error("Expected plan to have failed")
	}
}

func TestFormatPlan(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{
				ID:          1,
				Description: "Read file",
				Tools:       []string{"read_file"},
				DependsOn:   []int{},
				Parallel:    false,
			},
			{
				ID:          2,
				Description: "Write file",
				Tools:       []string{"write_file"},
				DependsOn:   []int{1},
				Parallel:    true,
			},
		},
	}

	formatted := FormatPlan(plan)
	if !strings.Contains(formatted, "📋 Plan:") {
		t.Error("Expected formatted plan to contain header")
	}
	if !strings.Contains(formatted, "Read file") {
		t.Error("Expected formatted plan to contain step description")
	}
	if !strings.Contains(formatted, "[順次]") {
		t.Error("Expected formatted plan to contain sequential tag")
	}
	if !strings.Contains(formatted, "[並列]") {
		t.Error("Expected formatted plan to contain parallel tag")
	}
}

func TestFindClosingBrace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected int
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			start:    0,
			expected: 16,
		},
		{
			name:     "nested object",
			input:    `{"outer": {"inner": "value"}}`,
			start:    0,
			expected: 29, // closing brace at index 28, +1 = 29
		},
		{
			name:     "with string containing brace",
			input:    `{"key": "value with } brace"}`,
			start:    0,
			expected: 29, // closing brace at index 28, +1 = 29
		},
		{
			name:     "with escaped quote",
			input:    `{"key": "value with \" quote"}`,
			start:    0,
			expected: 30, // closing brace at index 29, +1 = 30
		},
		{
			name:     "incomplete object",
			input:    `{"key": "value"`,
			start:    0,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findClosingBrace(tt.input, tt.start)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d for input: %s", tt.expected, result, tt.input)
			}
		})
	}
}

// Helper function
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
