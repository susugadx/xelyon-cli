package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestPlanModeImplementationHandoff_NormalModeInputIncludesApprovedPlan(t *testing.T) {
	handoff := newPlanModeImplementationHandoff("implement feature", &plan.Plan{
		Summary: "Ship a small change",
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Update foo.go and tests",
			Tools:       []string{"read_file", "apply_patch"},
			TargetFiles: []string{"foo.go"},
			Files:       []string{"foo.go", "foo_test.go"},
			ReadFiles:   []string{"README.md"},
			WriteFiles:  []string{"foo_test.go"},
		}},
	})
	if handoff == nil {
		t.Fatal("handoff = nil")
	}

	input := handoff.normalModeInput()
	for _, want := range []string{
		"Implement the approved plan now.",
		"Original request:",
		"implement feature",
		"Summary: Ship a small change",
		"1. Update foo.go and tests",
		"Tools: read_file, apply_patch",
		"Files: foo.go, foo_test.go, README.md",
		"Use the plan as guidance",
	} {
		if !strings.Contains(input, want) {
			t.Fatalf("normalModeInput() = %q, want fragment %q", input, want)
		}
	}
}

func TestPlanModeImplementationHandoff_ClonesApprovedPlan(t *testing.T) {
	p := &plan.Plan{
		Summary: "Ship a small change",
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Update foo.go and tests",
			Tools:       []string{"apply_patch"},
			TargetFiles: []string{"foo.go"},
		}},
	}
	handoff := newPlanModeImplementationHandoff("implement feature", p)
	if handoff == nil {
		t.Fatal("handoff = nil")
	}

	p.Summary = "mutated summary"
	p.Steps[0].Description = "mutated description"
	p.Steps[0].Tools[0] = "mutated_tool"
	p.Steps[0].TargetFiles[0] = "mutated.go"

	input := handoff.normalModeInput()
	for _, want := range []string{
		"Summary: Ship a small change",
		"1. Update foo.go and tests",
		"Tools: apply_patch",
		"Files: foo.go",
	} {
		if !strings.Contains(input, want) {
			t.Fatalf("normalModeInput() = %q, want original fragment %q", input, want)
		}
	}
	for _, notWant := range []string{"mutated summary", "mutated description", "mutated_tool", "mutated.go"} {
		if strings.Contains(input, notWant) {
			t.Fatalf("normalModeInput() = %q, should not contain mutated fragment %q", input, notWant)
		}
	}
}
