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

type planHandoffFileGroup struct {
	label string
	files []string
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
	dst.Findings = append([]string(nil), src.Findings...)
	dst.Evidence = append([]string(nil), src.Evidence...)
	dst.Constraints = append([]string(nil), src.Constraints...)
	dst.Steps = make([]plan.PlanStep, len(src.Steps))
	for i := range src.Steps {
		dst.Steps[i] = src.Steps[i]
		dst.Steps[i].Tools = append([]string(nil), src.Steps[i].Tools...)
		dst.Steps[i].DependsOn = append([]int(nil), src.Steps[i].DependsOn...)
		dst.Steps[i].TargetFiles = append([]string(nil), src.Steps[i].TargetFiles...)
		dst.Steps[i].ReadFiles = append([]string(nil), src.Steps[i].ReadFiles...)
		dst.Steps[i].WriteFiles = append([]string(nil), src.Steps[i].WriteFiles...)
		dst.Steps[i].Files = append([]string(nil), src.Steps[i].Files...)
		dst.Steps[i].Verification = append([]string(nil), src.Steps[i].Verification...)
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
	writeHandoffListSection(&b, "Findings", h.approvedPlan.Findings)
	writeHandoffListSection(&b, "Evidence", h.approvedPlan.Evidence)
	writeHandoffListSection(&b, "Constraints", h.approvedPlan.Constraints)
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
		if purpose := strings.TrimSpace(step.Purpose); purpose != "" {
			b.WriteString("   Purpose: ")
			b.WriteString(purpose)
			b.WriteString("\n")
		}
		if len(step.Tools) > 0 {
			b.WriteString("   Tools: ")
			b.WriteString(strings.Join(step.Tools, ", "))
			b.WriteString("\n")
		}
		if verification := compactHandoffValues(step.Verification, nil); len(verification) > 0 {
			b.WriteString("   Verification: ")
			b.WriteString(strings.Join(verification, ", "))
			b.WriteString("\n")
		}
		for _, group := range handoffStepFileGroups(step) {
			b.WriteString("   ")
			b.WriteString(group.label)
			b.WriteString(": ")
			b.WriteString(strings.Join(group.files, ", "))
			b.WriteString("\n")
		}
	}

	b.WriteString("\nUse the plan as guidance, adapt if the code requires it, and briefly report any deviation.")
	return b.String()
}

func handoffStepFileGroups(step plan.PlanStep) []planHandoffFileGroup {
	targetFiles := compactHandoffFiles(step.TargetFiles, nil)
	readFiles := compactHandoffFiles(step.ReadFiles, nil)
	writeFiles := compactHandoffFiles(step.WriteFiles, nil)
	structuredFiles := handoffFileSet(targetFiles, readFiles, writeFiles)
	relatedFiles := compactHandoffFiles(step.Files, structuredFiles)

	groups := make([]planHandoffFileGroup, 0, 4)
	if len(targetFiles) > 0 {
		groups = append(groups, planHandoffFileGroup{label: "Target files", files: targetFiles})
	}
	if len(readFiles) > 0 {
		groups = append(groups, planHandoffFileGroup{label: "Read files", files: readFiles})
	}
	if len(writeFiles) > 0 {
		groups = append(groups, planHandoffFileGroup{label: "Write files", files: writeFiles})
	}
	if len(relatedFiles) > 0 {
		groups = append(groups, planHandoffFileGroup{label: "Related files", files: relatedFiles})
	}
	return groups
}

func writeHandoffListSection(b *strings.Builder, label string, values []string) bool {
	values = compactHandoffValues(values, nil)
	if len(values) == 0 {
		return false
	}

	b.WriteString(label)
	b.WriteString(":\n")
	for _, value := range values {
		b.WriteString(" - ")
		b.WriteString(value)
		b.WriteString("\n")
	}
	return true
}

func compactHandoffFiles(files []string, exclude map[string]bool) []string {
	return compactHandoffValues(files, exclude)
}

func compactHandoffValues(values []string, exclude map[string]bool) []string {
	seen := make(map[string]bool)
	compact := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		if exclude != nil && exclude[value] {
			continue
		}
		seen[value] = true
		compact = append(compact, value)
	}
	return compact
}

func handoffFileSet(groups ...[]string) map[string]bool {
	files := make(map[string]bool)
	for _, group := range groups {
		for _, file := range group {
			files[file] = true
		}
	}
	return files
}
