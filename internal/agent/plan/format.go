package plan

import (
	"fmt"
	"strings"
)

// FormatPlan は計画を見やすく整形
func FormatPlan(plan *Plan) string {
	var sb strings.Builder
	sb.WriteString("Plan:\n")

	writePlanValues(&sb, "Findings", plan.Findings)
	writePlanValues(&sb, "Evidence", plan.Evidence)
	writePlanValues(&sb, "Constraints", plan.Constraints)

	for _, step := range plan.Steps {
		fmt.Fprintf(&sb, "  %d. %s\n", step.ID, step.Description)

		if purpose := strings.TrimSpace(step.Purpose); purpose != "" {
			fmt.Fprintf(&sb, "     Purpose: %s\n", purpose)
		}

		if len(step.Tools) > 0 {
			fmt.Fprintf(&sb, "     Tools: %s\n", strings.Join(step.Tools, ", "))
		}

		if len(step.Files) > 0 {
			fmt.Fprintf(&sb, "     Files: %s\n", strings.Join(step.Files, ", "))
		}

		if len(step.Verification) > 0 {
			fmt.Fprintf(&sb, "     Verification: %s\n", strings.Join(step.Verification, ", "))
		}

		if len(step.DependsOn) > 0 {
			fmt.Fprintf(&sb, "     Depends on: %v\n", step.DependsOn)
		}
	}

	return sb.String()
}

func writePlanValues(sb *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(sb, "%s: %s\n", label, strings.Join(values, ", "))
}
