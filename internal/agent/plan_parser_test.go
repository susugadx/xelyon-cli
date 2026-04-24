package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestParsePlan(t *testing.T) {
	jsonStr := `{
		"steps": [
			{
				"id": 1,
				"description": "Read main.go",
				"tools": ["read_file"],
				"depends_on": []
			},
			{
				"id": 2,
				"description": "Run tests",
				"tools": ["bash"],
				"depends_on": [1]
			}
		]
	}`

	p, err := plan.ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(p.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(p.Steps))
	}

	step1 := p.Steps[0]
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

	step2 := p.Steps[1]
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
      "depends_on": []
    }
  ]
}
` + "```" + `

This is the execution plan.`

	jsonStr := plan.ExtractPlanJSON(response)
	if jsonStr == "" {
		t.Fatal("Failed to extract JSON: got empty string")
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
      "depends_on": []
    }
  ]
}

Done!`

	jsonStr := plan.ExtractPlanJSON(response)
	if jsonStr == "" {
		t.Fatal("Failed to extract JSON with newlines: got empty string")
	}

	if !strings.Contains(jsonStr, `"steps"`) {
		t.Errorf("Extracted JSON does not contain 'steps': %s", jsonStr)
	}
}

func TestExtractPlanJSON_NoJSON(t *testing.T) {
	response := "This response contains no JSON at all."

	jsonStr := plan.ExtractPlanJSON(response)
	if jsonStr != "" {
		t.Errorf("Expected empty string for response without JSON, but got: %s", jsonStr)
	}
}
