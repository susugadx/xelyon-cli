package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExtractPlanJSON(t *testing.T) {
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
			result := ExtractPlanJSON(tt.input)
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
			plan, err := ParsePlan(tt.input)
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
			if plan.Summary != tt.wantSummary {
				t.Errorf("ParsePlan() summary = %q, want %q", plan.Summary, tt.wantSummary)
			}
			if len(plan.Steps) != tt.wantSteps {
				t.Errorf("ParsePlan() steps count = %d, want %d", len(plan.Steps), tt.wantSteps)
			}
		})
	}
}

func TestPlanStep_Tools(t *testing.T) {
	input := `{"plan": {"summary": "Test", "steps": [{"id": 1, "description": "Write file", "tools": ["write_file", "str_replace"]}]}}`
	plan, err := ParsePlan(input)
	if err != nil {
		t.Fatalf("ParsePlan() error: %v", err)
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

func TestHashToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		toolCalls []*tools.ToolCall
		want      string
	}{
		{
			name:      "empty",
			toolCalls: []*tools.ToolCall{},
			want:      "",
		},
		{
			name: "single tool",
			toolCalls: []*tools.ToolCall{
				{Tool: "read_file", Args: map[string]string{"path": "/test.go"}},
			},
			want: "read_file:map[path:/test.go]",
		},
		{
			name: "multiple tools sorted",
			toolCalls: []*tools.ToolCall{
				{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
				{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
			},
			want: "read_file:map[path:/a.go]|read_file:map[path:/b.go]",
		},
		{
			name: "different tools",
			toolCalls: []*tools.ToolCall{
				{Tool: "search_code", Args: map[string]string{"pattern": "func"}},
				{Tool: "read_file", Args: map[string]string{"path": "/test.go"}},
			},
			want: "read_file:map[path:/test.go]|search_code:map[pattern:func]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hashToolCalls(tt.toolCalls)
			if got != tt.want {
				t.Errorf("hashToolCalls() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHashToolCalls_OrderIndependent(t *testing.T) {
	// 同じツールセットは順序に関係なく同じハッシュを返す
	calls1 := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
	}
	calls2 := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
	}

	hash1 := hashToolCalls(calls1)
	hash2 := hashToolCalls(calls2)

	if hash1 != hash2 {
		t.Errorf("hashToolCalls() should be order-independent: %q != %q", hash1, hash2)
	}
}
