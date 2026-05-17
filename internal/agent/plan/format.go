package plan

import (
	"fmt"
	"strings"
)

// FormatPlan は計画を見やすく整形
func FormatPlan(plan *Plan) string {
	var sb strings.Builder
	sb.WriteString("Plan:\n")

	for _, step := range plan.Steps {
		fmt.Fprintf(&sb, "  %d. %s\n", step.ID, step.Description)

		if len(step.Tools) > 0 {
			fmt.Fprintf(&sb, "     Tools: %s\n", strings.Join(step.Tools, ", "))
		}

		if len(step.DependsOn) > 0 {
			fmt.Fprintf(&sb, "     Depends on: %v\n", step.DependsOn)
		}
	}

	return sb.String()
}
