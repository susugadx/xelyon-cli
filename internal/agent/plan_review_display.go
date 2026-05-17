package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func buildPlanReviewDisplay(p *plan.Plan) *ui.PlanDisplay {
	display := ui.NewPlanReviewDisplay()
	if p == nil {
		return display
	}

	display.SetSummary(p.Summary)
	for _, step := range p.Steps {
		display.AddPlanStep(ui.PlanStep{
			ID:           step.ID,
			Description:  step.Description,
			Purpose:      step.Purpose,
			Tools:        step.Tools,
			Files:        planReviewStepFiles(step),
			Verification: planReviewStepVerification(step),
		})
	}
	return display
}

func planReviewStepFiles(step plan.PlanStep) []string {
	seen := make(map[string]bool)
	files := make([]string, 0, len(step.TargetFiles)+len(step.WriteFiles)+len(step.ReadFiles)+len(step.Files))
	appendFiles := func(group []string) {
		for _, file := range group {
			file = strings.TrimSpace(file)
			if file == "" || seen[file] {
				continue
			}
			seen[file] = true
			files = append(files, file)
		}
	}

	appendFiles(step.TargetFiles)
	appendFiles(step.WriteFiles)
	appendFiles(step.ReadFiles)
	appendFiles(step.Files)
	return files
}

func planReviewStepVerification(step plan.PlanStep) []string {
	seen := make(map[string]bool)
	verification := make([]string, 0, len(step.Verification))
	for _, item := range step.Verification {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		verification = append(verification, item)
	}
	return verification
}
