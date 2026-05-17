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
			Files:       []string{"foo.go", "foo_test.go", "docs/foo.md"},
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
		"Target files: foo.go",
		"Read files: README.md",
		"Write files: foo_test.go",
		"Related files: docs/foo.md",
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
			ReadFiles:   []string{"foo.go"},
			WriteFiles:  []string{"foo_test.go"},
			Files:       []string{"README.md"},
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
	p.Steps[0].ReadFiles[0] = "mutated_read.go"
	p.Steps[0].WriteFiles[0] = "mutated_write.go"
	p.Steps[0].Files[0] = "mutated_related.go"

	input := handoff.normalModeInput()
	for _, want := range []string{
		"Summary: Ship a small change",
		"1. Update foo.go and tests",
		"Tools: apply_patch",
		"Target files: foo.go",
		"Read files: foo.go",
		"Write files: foo_test.go",
		"Related files: README.md",
	} {
		if !strings.Contains(input, want) {
			t.Fatalf("normalModeInput() = %q, want original fragment %q", input, want)
		}
	}
	for _, notWant := range []string{"mutated summary", "mutated description", "mutated_tool", "mutated.go", "mutated_read.go", "mutated_write.go", "mutated_related.go"} {
		if strings.Contains(input, notWant) {
			t.Fatalf("normalModeInput() = %q, should not contain mutated fragment %q", input, notWant)
		}
	}
}

func TestPlanModeImplementationHandoff_FileGroupsNormalizeAndPreserveRoles(t *testing.T) {
	handoff := newPlanModeImplementationHandoff("implement feature", &plan.Plan{
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Update files",
			TargetFiles: []string{" foo.go ", "foo.go", ""},
			ReadFiles:   []string{"foo.go", "bar.go", "bar.go"},
			WriteFiles:  []string{"bar.go"},
			Files:       []string{"foo.go", "bar.go", "baz.go", "baz.go"},
		}},
	})
	if handoff == nil {
		t.Fatal("handoff = nil")
	}

	input := handoff.normalModeInput()
	for _, want := range []string{
		"Target files: foo.go",
		"Read files: foo.go, bar.go",
		"Write files: bar.go",
		"Related files: baz.go",
	} {
		if !strings.Contains(input, want) {
			t.Fatalf("normalModeInput() = %q, want fragment %q", input, want)
		}
	}
	for _, notWant := range []string{
		"Target files: foo.go, foo.go",
		"Read files: foo.go, bar.go, bar.go",
		"Related files: foo.go",
		"Related files: bar.go",
	} {
		if strings.Contains(input, notWant) {
			t.Fatalf("normalModeInput() = %q, should not contain fragment %q", input, notWant)
		}
	}
}
