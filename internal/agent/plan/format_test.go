package plan

import (
	"strings"
	"testing"
)

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

	checks := []string{"Plan:", "1.", "First step", "2.", "Second step", "Tools:", "Depends on:"}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("FormatPlan output should contain %q", check)
		}
	}
}
