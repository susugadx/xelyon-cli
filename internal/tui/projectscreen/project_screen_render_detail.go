package projectscreen

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (ps *Screen) renderProjectSectionPane(width, height int) []string {
	lines := make([]string, 0, height)
	for i, info := range projectSections {
		selected := i == ps.sectionIndex
		bg, fg := projectPaneColors(selected, ps.activePane == projectPaneSection)
		prefix := "  "
		if selected {
			prefix = "> "
		}
		line := bg + fg + prefix + termtext.TruncateWithANSI(info.title, width-3) + theme.Config.Reset
		lines = append(lines, termtext.FillANSITextWidth(line, width, bg))
	}
	return appendProjectPanePadding(lines, width, height, theme.Config.BgNormal)
}

func (ps *Screen) renderProjectDetailPane(width, height int) []string {
	if ps.missing {
		return appendProjectPanePadding(projectMissingLines(width), width, height, theme.Config.BgNormal)
	}
	if ps.editMode != projectEditNone {
		return appendProjectPanePadding(ps.renderProjectEditLines(width), width, height, theme.Config.BgNormal)
	}

	info := ps.selectedSectionInfo()
	lines := []string{
		theme.Config.BgNormal + theme.Config.FgBright + " " + info.title + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgDim + " " + info.description + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgDim + "" + theme.Config.Reset,
	}
	lines = append(lines, ps.renderProjectSectionDetail(width, max(0, height-len(lines)))...)
	return appendProjectPanePadding(lines, width, height, theme.Config.BgNormal)
}

func projectMissingLines(width int) []string {
	return []string{
		theme.Config.BgNormal + theme.Config.FgYellow + " xelyon.yaml not found" + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgDim + "" + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgNormal + " Press Enter to create a template." + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgDim + " Press Esc to return to chat." + theme.Config.Reset,
	}
}

func (ps *Screen) renderProjectSectionDetail(width, height int) []string {
	switch ps.selectedSection() {
	case projectSectionContext:
		return projectContextLines(ps.pc.Context, width)
	case projectSectionRules, projectSectionIgnore, projectSectionFinalCommands:
		return ps.renderProjectListLines(width, height)
	case projectSectionConditional:
		return projectConditionalLines(ps)
	case projectSectionFinalTimeout:
		if !ps.hasProjectFinalCheckCommands() {
			return []string{
				theme.Config.BgNormal + theme.Config.FgNormal + " Timeout: inherited/default" + theme.Config.Reset,
				theme.Config.BgNormal + theme.Config.FgDim + " Add a final check command before editing timeout." + theme.Config.Reset,
			}
		}
		return []string{
			theme.Config.BgNormal + theme.Config.FgNormal + fmt.Sprintf(" Timeout: %d sec", ps.finalChecksTimeoutForDisplay()) + theme.Config.Reset,
			theme.Config.BgNormal + theme.Config.FgDim + " Enter to edit." + theme.Config.Reset,
		}
	default:
		return nil
	}
}

func projectContextLines(context string, width int) []string {
	if strings.TrimSpace(context) == "" {
		return []string{
			theme.Config.BgNormal + theme.Config.FgDim + " (empty)" + theme.Config.Reset,
			theme.Config.BgNormal + theme.Config.FgDim + " Enter to edit. Ctrl+S confirms multiline edits." + theme.Config.Reset,
		}
	}
	lines := make([]string, 0, 8)
	for _, line := range strings.Split(context, "\n") {
		if len(lines) >= 8 {
			lines = append(lines, theme.Config.BgNormal+theme.Config.FgDim+" ..."+theme.Config.Reset)
			break
		}
		lines = append(lines, theme.Config.BgNormal+theme.Config.FgNormal+" "+termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(line), max(0, width-2))+theme.Config.Reset)
	}
	lines = append(lines, theme.Config.BgNormal+theme.Config.FgDim+" Enter to edit. Ctrl+S confirms multiline edits."+theme.Config.Reset)
	return lines
}

func projectConditionalLines(ps *Screen) []string {
	if ps.pc == nil || len(ps.pc.Conditional) == 0 {
		return []string{
			theme.Config.BgNormal + theme.Config.FgDim + " (empty)" + theme.Config.Reset,
			theme.Config.BgNormal + theme.Config.FgDim + " Preview-only in this first TUI pass." + theme.Config.Reset,
		}
	}
	lines := make([]string, 0, len(ps.pc.Conditional)*2)
	for _, block := range ps.pc.Conditional {
		name := block.Name
		if name == "" {
			name = "(unnamed)"
		}
		lines = append(lines, theme.Config.BgNormal+theme.Config.FgNormal+" "+termtext.SanitizeSingleLineANSI(name)+theme.Config.Reset)
		lines = append(lines, theme.Config.BgNormal+theme.Config.FgDim+fmt.Sprintf("   paths:%d rules:%d", len(block.Paths), len(block.Rules))+theme.Config.Reset)
	}
	lines = append(lines, theme.Config.BgNormal+theme.Config.FgDim+" Preview-only in this first TUI pass."+theme.Config.Reset)
	return lines
}

func (ps *Screen) renderProjectEditLines(width int) []string {
	switch ps.editMode {
	case projectEditContext:
		view := strings.ReplaceAll(ps.contextArea.View(), theme.Config.Reset, theme.Config.Reset+theme.Config.BgNormal)
		lines := []string{
			theme.Config.BgNormal + theme.Config.FgBright + " Editing context" + theme.Config.Reset,
			theme.Config.BgNormal + theme.Config.FgDim + " Ctrl+S:confirm  Esc:cancel" + theme.Config.Reset,
		}
		for _, line := range strings.Split(view, "\n") {
			lines = append(lines, theme.Config.BgNormal+line+theme.Config.Reset)
		}
		return lines
	case projectEditLine:
		label := "Value"
		if ps.lineEditKind == projectLineEditTimeout {
			label = "Timeout seconds"
		}
		inputView := strings.ReplaceAll(ps.editInput.View(), theme.Config.Reset, theme.Config.Reset+theme.Config.BgNormal)
		return []string{
			theme.Config.BgNormal + theme.Config.FgBright + " " + label + theme.Config.Reset,
			theme.Config.BgNormal + theme.Config.FgDim + " Enter:confirm  Esc:cancel" + theme.Config.Reset,
			theme.Config.BgNormal + theme.Config.FgNormal + " " + inputView + theme.Config.Reset,
		}
	default:
		return nil
	}
}

func appendProjectPanePadding(lines []string, width, height int, bg string) []string {
	for len(lines) < height {
		lines = append(lines, termtext.FillANSITextWidth("", width, bg))
	}
	if len(lines) > height {
		return lines[:height]
	}
	return lines
}
