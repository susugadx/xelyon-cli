package ui

import (
	"strings"
	"testing"
)

func TestPlanDisplay_RenderDetailSections(t *testing.T) {
	p := NewPlanDisplay("Plan Review").
		SetSummary("Ship a reviewable plan").
		AddDetailSection("調査結果", []string{" plan/handoff.go owns implementation handoff ", "plan/handoff.go owns implementation handoff", ""}).
		AddDetailSection("根拠", []string{"internal/agent/plan/handoff.go: ImplementationHandoff.NormalModeInput"}).
		AddDetailSection("制約", []string{"Do not carry raw investigation history"}).
		AddPlanStep(PlanStep{ID: 1, Description: "Update handoff"})

	result := p.Render()
	for _, want := range []string{
		"調査結果",
		"  - plan/handoff.go owns implementation handoff",
		"根拠",
		"  - internal/agent/plan/handoff.go: ImplementationHandoff.NormalModeInput",
		"制約",
		"  - Do not carry raw investigation history",
		"ステップ",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("PlanDisplay.Render() = %q, want fragment %q", result, want)
		}
	}
	if strings.Count(result, "plan/handoff.go owns implementation handoff") != 1 {
		t.Fatalf("PlanDisplay.Render() = %q, duplicated detail value", result)
	}
}
