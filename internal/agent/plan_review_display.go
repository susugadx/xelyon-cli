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
	display.AddDetailSection("検証予定", planReviewPlanVerification(p))
	display.AddDetailSection("調査結果", p.Findings)
	display.AddDetailSection("根拠", p.Evidence)
	display.AddDetailSection("制約", p.Constraints)
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

func buildPlanNoImplementationDisplay(p *plan.Plan) *ui.PlanDisplay {
	if p == nil || !planHasNoImplementationDetails(p) {
		return nil
	}

	display := ui.NewPlanDisplay("Investigation Result")
	display.SetSummary(p.Summary)
	display.AddDetailSection("調査結果", p.Findings)
	display.AddDetailSection("根拠", p.Evidence)
	display.AddDetailSection("制約", p.Constraints)
	return display
}

func planHasNoImplementationDetails(p *plan.Plan) bool {
	return strings.TrimSpace(p.Summary) != "" ||
		hasNonEmptyStrings(p.Findings) ||
		hasNonEmptyStrings(p.Evidence) ||
		hasNonEmptyStrings(p.Constraints)
}

func hasNonEmptyStrings(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
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

func planReviewPlanVerification(p *plan.Plan) []string {
	if p == nil {
		return nil
	}

	verification := make([]string, 0)
	for _, step := range p.Steps {
		verification = append(verification, step.Verification...)
	}
	return plan.CompactVerificationHints(verification)
}

func planReviewStepVerification(step plan.PlanStep) []string {
	return plan.CompactVerificationHints(step.Verification)
}
