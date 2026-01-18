package agent

import (
	"testing"
)

func TestExtractPlanV2JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic plan",
			input: `Based on my investigation, here is my plan:
{"plan": {"summary": "Add new feature", "steps": [{"id": 1, "description": "Step 1", "tools": ["write_file"]}]}}`,
			expected: `{"plan": {"summary": "Add new feature", "steps": [{"id": 1, "description": "Step 1", "tools": ["write_file"]}]}}`,
		},
		{
			name: "plan with space",
			input: `I'll implement this:
{ "plan": {"summary": "Fix bug", "steps": [{"id": 1, "description": "Fix", "tools": []}]}}`,
			expected: `{ "plan": {"summary": "Fix bug", "steps": [{"id": 1, "description": "Fix", "tools": []}]}}`,
		},
		{
			name:     "no plan",
			input:    "This is just a regular response without any plan.",
			expected: "",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPlanV2JSON(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractPlanV2JSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParsePlanV2(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSummary string
		wantSteps   int
		wantErr     bool
	}{
		{
			name:        "valid plan",
			input:       `{"plan": {"summary": "Add feature X", "steps": [{"id": 1, "description": "Step 1", "tools": ["write_file"]}, {"id": 2, "description": "Step 2", "tools": ["str_replace"]}]}}`,
			wantSummary: "Add feature X",
			wantSteps:   2,
			wantErr:     false,
		},
		{
			name:        "empty steps",
			input:       `{"plan": {"summary": "Research only", "steps": []}}`,
			wantSummary: "Research only",
			wantSteps:   0,
			wantErr:     false,
		},
		{
			name:        "no summary",
			input:       `{"plan": {"steps": [{"id": 1, "description": "Step 1", "tools": []}]}}`,
			wantSummary: "",
			wantSteps:   1,
			wantErr:     false,
		},
		{
			name:    "invalid json",
			input:   `{"plan": invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := ParsePlanV2(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePlanV2() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePlanV2() unexpected error: %v", err)
				return
			}
			if plan.Summary != tt.wantSummary {
				t.Errorf("ParsePlanV2() summary = %q, want %q", plan.Summary, tt.wantSummary)
			}
			if len(plan.Steps) != tt.wantSteps {
				t.Errorf("ParsePlanV2() steps count = %d, want %d", len(plan.Steps), tt.wantSteps)
			}
		})
	}
}

func TestPlanStepV2_Tools(t *testing.T) {
	input := `{"plan": {"summary": "Test", "steps": [{"id": 1, "description": "Write file", "tools": ["write_file", "str_replace"]}]}}`
	plan, err := ParsePlanV2(input)
	if err != nil {
		t.Fatalf("ParsePlanV2() error: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	if step.ID != 1 {
		t.Errorf("Step ID = %d, want 1", step.ID)
	}
	if step.Description != "Write file" {
		t.Errorf("Step Description = %q, want %q", step.Description, "Write file")
	}
	if len(step.Tools) != 2 {
		t.Errorf("Step Tools count = %d, want 2", len(step.Tools))
	}
	if step.Tools[0] != "write_file" || step.Tools[1] != "str_replace" {
		t.Errorf("Step Tools = %v, want [write_file, str_replace]", step.Tools)
	}
}
