package uiplanview

import (
	"fmt"
	"strings"
)

// PlanDetailSection は計画レビュー上部に表示する補足情報を表す。
type PlanDetailSection struct {
	Title  string
	Values []string
}

// AddDetailSection は計画レビュー上部の補足セクションを追加する。
func (p *PlanDisplay) AddDetailSection(title string, values []string) *PlanDisplay {
	p.DetailSections = append(p.DetailSections, PlanDetailSection{
		Title:  title,
		Values: append([]string(nil), values...),
	})
	return p
}

func writePlanDetailSections(sb *strings.Builder, sections []PlanDetailSection) bool {
	wrote := false
	for _, section := range sections {
		if writePlanDetailSection(sb, section) {
			wrote = true
		}
	}
	return wrote
}

func writePlanDetailSection(sb *strings.Builder, section PlanDetailSection) bool {
	title := strings.TrimSpace(section.Title)
	values := compactPlanStepDetailValues(section.Values)
	if title == "" || len(values) == 0 {
		return false
	}

	sb.WriteString(title)
	sb.WriteString("\n")
	for _, value := range values {
		fmt.Fprintf(sb, "  - %s\n", value)
	}
	return true
}
