package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

type planModeImplementationHandoff struct {
	originalUserRequest string
	approvedPlan        plan.Plan
}

func newPlanModeImplementationHandoff(originalUserRequest string, approvedPlan *plan.Plan) *planModeImplementationHandoff {
	if approvedPlan == nil || len(approvedPlan.Steps) == 0 {
		return nil
	}
	return &planModeImplementationHandoff{
		originalUserRequest: strings.TrimSpace(originalUserRequest),
		approvedPlan:        clonePlanForHandoff(approvedPlan),
	}
}

func clonePlanForHandoff(src *plan.Plan) plan.Plan {
	if src == nil {
		return plan.Plan{}
	}
	dst := *src
	dst.Steps = make([]plan.PlanStep, len(src.Steps))
	for i := range src.Steps {
		dst.Steps[i] = src.Steps[i]
		dst.Steps[i].Tools = append([]string(nil), src.Steps[i].Tools...)
		dst.Steps[i].DependsOn = append([]int(nil), src.Steps[i].DependsOn...)
		dst.Steps[i].TargetFiles = append([]string(nil), src.Steps[i].TargetFiles...)
		dst.Steps[i].ReadFiles = append([]string(nil), src.Steps[i].ReadFiles...)
		dst.Steps[i].WriteFiles = append([]string(nil), src.Steps[i].WriteFiles...)
		dst.Steps[i].Files = append([]string(nil), src.Steps[i].Files...)
	}
	return dst
}

func (h *planModeImplementationHandoff) normalModeInput() string {
	if h == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("Implement the approved plan now.\n\n")
	if h.originalUserRequest != "" {
		b.WriteString("Original request:\n")
		b.WriteString(h.originalUserRequest)
		b.WriteString("\n\n")
	}

	b.WriteString("Approved plan:\n")
	if summary := strings.TrimSpace(h.approvedPlan.Summary); summary != "" {
		b.WriteString("Summary: ")
		b.WriteString(summary)
		b.WriteString("\n")
	}
	for idx, step := range h.approvedPlan.Steps {
		description := strings.TrimSpace(step.Description)
		if description == "" {
			continue
		}
		id := step.ID
		if id <= 0 {
			id = idx + 1
		}
		fmt.Fprintf(&b, "%d. %s\n", id, description)
		if len(step.Tools) > 0 {
			b.WriteString("   Tools: ")
			b.WriteString(strings.Join(step.Tools, ", "))
			b.WriteString("\n")
		}
		if files := handoffStepFiles(step); len(files) > 0 {
			b.WriteString("   Files: ")
			b.WriteString(strings.Join(files, ", "))
			b.WriteString("\n")
		}
	}

	b.WriteString("\nUse the plan as guidance, adapt if the code requires it, and briefly report any deviation.")
	return b.String()
}

func handoffStepFiles(step plan.PlanStep) []string {
	seen := make(map[string]bool)
	files := make([]string, 0, len(step.TargetFiles)+len(step.Files)+len(step.ReadFiles)+len(step.WriteFiles))
	for _, group := range [][]string{step.TargetFiles, step.Files, step.ReadFiles, step.WriteFiles} {
		for _, file := range group {
			file = strings.TrimSpace(file)
			if file == "" || seen[file] {
				continue
			}
			seen[file] = true
			files = append(files, file)
		}
	}
	return files
}
