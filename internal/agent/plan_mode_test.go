package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestExtractPlanJSON_Mode(t *testing.T) {
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
			result := plan.ExtractPlanJSON(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractPlanJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParsePlan_V2Format(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSummary string
		wantSteps   int
		wantErr     bool
	}{
		{
			name:        "valid plan with wrapper",
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
			p, err := plan.ParsePlan(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePlan() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePlan() unexpected error: %v", err)
				return
			}
			if p.Summary != tt.wantSummary {
				t.Errorf("ParsePlan() summary = %q, want %q", p.Summary, tt.wantSummary)
			}
			if len(p.Steps) != tt.wantSteps {
				t.Errorf("ParsePlan() steps count = %d, want %d", len(p.Steps), tt.wantSteps)
			}
		})
	}
}

func TestPlanStep_Tools(t *testing.T) {
	input := `{"plan": {"summary": "Test", "steps": [{"id": 1, "description": "Write file", "tools": ["write_file", "str_replace"]}]}}`
	p, err := plan.ParsePlan(input)
	if err != nil {
		t.Fatalf("ParsePlan() error: %v", err)
	}

	if len(p.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(p.Steps))
	}

	step := p.Steps[0]
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

func TestRunPlanMode_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer

	agent := NewAgentWithRuntime("test-model", &mockProvider{name: "test"}, false, &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	})
	t.Cleanup(agent.Cleanup)

	if err := agent.RunPlanMode(context.Background(), "investigate only"); err != nil {
		t.Fatalf("RunPlanMode() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Investigation phase - researching the codebase") {
		t.Fatalf("expected runtime output to contain investigation header, got %q", output)
	}
	if !strings.Contains(output, "Investigation complete. No implementation needed.") {
		t.Fatalf("expected runtime output to contain completion message, got %q", output)
	}
}

func TestConfirmPlan_UsesRuntimePromptIO(t *testing.T) {
	var out bytes.Buffer

	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader("2\n"), &out, &out),
		},
	}

	approved, feedback := agent.confirmPlan()
	if approved {
		t.Fatal("confirmPlan() approved = true, want false")
	}
	if feedback != "" {
		t.Fatalf("confirmPlan() feedback = %q, want empty", feedback)
	}
	if !strings.Contains(out.String(), "Approve this plan?") {
		t.Fatalf("expected runtime output to contain confirmation prompt, got %q", out.String())
	}
}
