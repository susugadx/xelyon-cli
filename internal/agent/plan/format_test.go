package plan

import (
	"strings"
	"testing"
)

func TestFormatPlan(t *testing.T) {
	plan := &Plan{
		Summary:            "Test plan",
		AcceptanceCriteria: []string{"Plan v2 display keeps criteria"},
		Findings:           []string{"plan parser preserves handoff fields"},
		Evidence:           []string{"internal/agent/plan/parser.go"},
		Constraints:        []string{"Keep legacy plan JSON compatible"},
		OpenQuestions:      []string{"Should legacy format stay accepted?"},
		Steps: []PlanStep{
			{ID: 1, Description: "First step", Purpose: "Understand current behavior", Tools: []string{"read_file"}, Files: []string{"foo.go"}},
			{ID: 2, Description: "Second step", Tools: []string{"write_file"}, Verification: []string{"go test ./internal/agent/plan"}, DependsOn: []int{1}},
		},
	}

	result := FormatPlan(plan)

	if result == "" {
		t.Error("FormatPlan returned empty string")
	}

	checks := []string{"Plan:", "Acceptance criteria:", "Plan v2 display keeps criteria", "Findings:", "Evidence:", "Constraints:", "Open questions:", "Should legacy format stay accepted?", "1.", "First step", "Purpose:", "Files:", "2.", "Second step", "Tools:", "Verification:", "Depends on:"}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("FormatPlan output should contain %q", check)
		}
	}
}
