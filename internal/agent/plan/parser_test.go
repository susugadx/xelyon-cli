package plan

import (
	"testing"
)

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

func TestExtractPlanJSON_V2Pattern(t *testing.T) {
	response := `Here is the plan:
{"plan": {"summary": "test", "steps": [{"id": 1, "description": "Step 1"}]}}
End of response.`

	result := ExtractPlanJSON(response)

	if result == "" {
		t.Fatal("ExtractPlanJSON returned empty string")
	}

	// Verify it's valid JSON
	_, err := ParsePlan(result)
	if err != nil {
		t.Errorf("Extracted JSON is not valid: %v", err)
	}
}

func TestExtractPlanJSON_CodeBlock(t *testing.T) {
	response := "Here is the plan:\n```json\n{\"summary\": \"test\", \"steps\": [{\"id\": 1, \"description\": \"Step 1\"}]}\n```"

	result := ExtractPlanJSON(response)

	if result == "" {
		t.Fatal("ExtractPlanJSON returned empty string for code block")
	}
}

func TestExtractPlanJSON_StepsPattern(t *testing.T) {
	response := `I'll create a plan:
{"steps": [{"id": 1, "description": "First step"}]}
Done.`

	result := ExtractPlanJSON(response)

	if result == "" {
		t.Fatal("ExtractPlanJSON should find steps pattern")
	}
}

func TestExtractPlanJSON_NoJSON(t *testing.T) {
	response := "This is just plain text without any JSON."

	result := ExtractPlanJSON(response)

	if result != "" {
		t.Errorf("Expected empty string for text without JSON, got %q", result)
	}
}

func TestExtractPlanJSON_ToolCallExcluded(t *testing.T) {
	response := `I'll use a tool:
` + "```json\n" + `{"tool": "read_file", "path": "main.go"}` + "\n```"

	result := ExtractPlanJSON(response)

	if result != "" {
		t.Errorf("Tool call JSON should be excluded, got %q", result)
	}
}

func TestIsToolCallJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		{
			name:     "tool call",
			json:     `{"tool": "read_file", "path": "main.go"}`,
			expected: true,
		},
		{
			name:     "tool with space",
			json:     `{ "tool": "write_file" }`,
			expected: true,
		},
		{
			name:     "plan JSON",
			json:     `{"summary": "test", "steps": []}`,
			expected: false,
		},
		{
			name:     "empty object",
			json:     `{}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolCallJSON(tt.json)
			if result != tt.expected {
				t.Errorf("isToolCallJSON(%q) = %v, want %v", tt.json, result, tt.expected)
			}
		})
	}
}

func TestFindClosingBrace(t *testing.T) {
	tests := []struct {
		name     string
		response string
		start    int
		expected int
	}{
		{
			name:     "simple object",
			response: `{"key": "value"}`,
			start:    0,
			expected: 16,
		},
		{
			name:     "nested object",
			response: `{"outer": {"inner": "value"}}`,
			start:    0,
			expected: 29,
		},
		{
			name:     "with escaped quotes",
			response: `{"key": "value with \"quotes\""}`,
			start:    0,
			expected: 32, // Returns position after closing brace
		},
		{
			name:     "unclosed",
			response: `{"key": "value"`,
			start:    0,
			expected: -1,
		},
		{
			name:     "brace in string",
			response: `{"key": "{}"}`,
			start:    0,
			expected: 13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findClosingBrace(tt.response, tt.start)
			if result != tt.expected {
				t.Errorf("findClosingBrace(%q, %d) = %d, want %d", tt.response, tt.start, result, tt.expected)
			}
		})
	}
}

func TestFormatPlan(t *testing.T) {
	plan := &Plan{
		Summary: "Test plan",
		Steps: []PlanStep{
			{ID: 1, Description: "First step", Tools: []string{"read_file"}},
			{ID: 2, Description: "Second step", Tools: []string{"write_file"}, DependsOn: []int{1}},
		},
	}

	result := FormatPlan(plan)

	if result == "" {
		t.Error("FormatPlan returned empty string")
	}

	// Check that key elements are present
	checks := []string{"Plan:", "1.", "First step", "2.", "Second step", "Tools:", "Depends on:"}
	for _, check := range checks {
		if !containsSubstring(result, check) {
			t.Errorf("FormatPlan output should contain %q", check)
		}
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
