package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestPlan_CanExecute(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "pending", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
		},
	}

	// Step 1 は依存なしで実行可能
	if !p.CanExecute(1) {
		t.Error("Expected step 1 to be executable")
	}

	// Step 2 は Step 1 が pending なので実行不可
	if p.CanExecute(2) {
		t.Error("Expected step 2 to be not executable (depends on step 1)")
	}

	// Step 1 を完了にする
	p.UpdateStatus(1, "completed", "Success")

	// Step 2, 3 が実行可能になる
	if !p.CanExecute(2) {
		t.Error("Expected step 2 to be executable after step 1 completed")
	}
	if !p.CanExecute(3) {
		t.Error("Expected step 3 to be executable after step 1 completed")
	}
}

func TestPlan_GetParallelSteps(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
			{ID: 4, Status: "pending", DependsOn: []int{2, 3}},
		},
	}

	parallelSteps := p.GetParallelSteps()
	if len(parallelSteps) != 2 {
		t.Errorf("Expected 2 parallel steps, got %d", len(parallelSteps))
	}

	// Step 2 と 3 は同じ depends_on を持つので並列実行可能
	if !containsStepID(parallelSteps, 2) || !containsStepID(parallelSteps, 3) {
		t.Errorf("Expected steps 2 and 3 to be parallel, got %v", parallelSteps)
	}
}

func TestPlan_GetParallelSteps_DifferentDeps(t *testing.T) {
	// 異なる depends_on を持つステップは並列実行されない
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "completed", DependsOn: []int{}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
			{ID: 4, Status: "pending", DependsOn: []int{2}},
		},
	}

	parallelSteps := p.GetParallelSteps()
	// Step 3 と 4 は異なる depends_on を持つので nil
	if parallelSteps != nil {
		t.Errorf("Expected nil for steps with different depends_on, got %v", parallelSteps)
	}
}

func TestPlan_GetNextStep(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{2}},
		},
	}

	nextStep := p.GetNextStep()
	if nextStep != 2 {
		t.Errorf("Expected next step to be 2, got %d", nextStep)
	}

	p.UpdateStatus(2, "completed", "Success")
	nextStep = p.GetNextStep()
	if nextStep != 3 {
		t.Errorf("Expected next step to be 3, got %d", nextStep)
	}

	p.UpdateStatus(3, "completed", "Success")
	nextStep = p.GetNextStep()
	if nextStep != -1 {
		t.Errorf("Expected next step to be -1 (all completed), got %d", nextStep)
	}
}

func TestPlan_IsCompleted(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if p.IsCompleted() {
		t.Error("Expected plan to be not completed")
	}

	p.UpdateStatus(2, "completed", "Success")
	if !p.IsCompleted() {
		t.Error("Expected plan to be completed")
	}
}

func TestPlan_HasFailed(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if p.HasFailed() {
		t.Error("Expected plan to have no failures")
	}

	p.UpdateStatus(2, "failed", "Error occurred")
	if !p.HasFailed() {
		t.Error("Expected plan to have failed")
	}
}

func TestFormatPlan(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{
				ID:          1,
				Description: "Read file",
				Tools:       []string{"read_file"},
				DependsOn:   []int{},
			},
			{
				ID:          2,
				Description: "Write file",
				Tools:       []string{"write_file"},
				DependsOn:   []int{1},
			},
		},
	}

	formatted := plan.FormatPlan(p)
	if !strings.Contains(formatted, "Plan:") {
		t.Error("Expected formatted plan to contain header")
	}
	if !strings.Contains(formatted, "Read file") {
		t.Error("Expected formatted plan to contain step description")
	}
	if !strings.Contains(formatted, "Depends on:") {
		t.Error("Expected formatted plan to contain depends_on info")
	}
}

func containsStepID(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
