package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

const (
	agentInstructionProjectFilesPath = "agent_instructions.project.files"
	agentInstructionGlobalFilesPath  = "agent_instructions.global.files"
)

func guidanceFileChoicesForField(path string, selected []string) []guidanceFileChoice {
	presets := guidanceFilePresets(path)
	if len(presets) == 0 {
		return nil
	}
	choices := make([]guidanceFileChoice, 0, len(presets)+len(selected))
	seen := make(map[string]struct{}, len(presets)+len(selected))
	for _, preset := range presets {
		normalized := strings.TrimSpace(preset)
		if normalized == "" {
			continue
		}
		seen[normalized] = struct{}{}
		choices = append(choices, guidanceFileChoice{Path: normalized, Preset: true})
	}
	for _, item := range selected {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		choices = append(choices, guidanceFileChoice{Path: normalized})
	}
	return choices
}

func guidanceFilePresets(path string) []string {
	switch path {
	case agentInstructionProjectFilesPath:
		return []string{"AGENTS.md", "CLAUDE.md", ".claude/CLAUDE.md"}
	case agentInstructionGlobalFilesPath:
		return []string{"~/.xelyon/AGENTS.md", "~/.codex/AGENTS.md", "~/.claude/CLAUDE.md", "~/.xelyon/CLAUDE.md"}
	default:
		return nil
	}
}

func (cs *configScreen) editingGuidanceFileChoices(path string) bool {
	return len(cs.editGuidanceChoices) > 0 && (path == agentInstructionProjectFilesPath || path == agentInstructionGlobalFilesPath)
}

func (cs *configScreen) handleGuidanceFileChoiceKey(msg tea.KeyMsg, field *config.ConfigField) (configCommand, tea.Cmd) {
	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.closeFieldSliceEdit(field)
		return configCommandNone, nil
	case msg.Type == tea.KeyUp || s == "k":
		cs.moveGuidanceChoiceIndex(-1)
		return configCommandNone, nil
	case msg.Type == tea.KeyDown || s == "j":
		cs.moveGuidanceChoiceIndex(1)
		return configCommandNone, nil
	case s == " ":
		cs.toggleSelectedGuidanceChoice()
		return configCommandNone, nil
	case s == "a":
		cs.beginSliceAdd()
		return configCommandNone, nil
	case s == "d":
		cs.deleteSelectedCustomGuidanceChoice()
		return configCommandNone, nil
	default:
		return configCommandNone, nil
	}
}

func (cs *configScreen) moveGuidanceChoiceIndex(delta int) {
	next := cs.editGuidanceIndex + delta
	if next < 0 {
		next = 0
	}
	if next >= len(cs.editGuidanceChoices) {
		next = len(cs.editGuidanceChoices) - 1
	}
	if next < 0 {
		next = 0
	}
	cs.editGuidanceIndex = next
}

func (cs *configScreen) toggleSelectedGuidanceChoice() {
	if cs.editGuidanceIndex < 0 || cs.editGuidanceIndex >= len(cs.editGuidanceChoices) {
		return
	}
	path := cs.editGuidanceChoices[cs.editGuidanceIndex].Path
	if guidanceItemSelected(cs.editSliceItems, path) {
		cs.editSliceItems = removeGuidanceItem(cs.editSliceItems, path)
		return
	}
	cs.editSliceItems = append(cs.editSliceItems, path)
}

func (cs *configScreen) deleteSelectedCustomGuidanceChoice() {
	if cs.editGuidanceIndex < 0 || cs.editGuidanceIndex >= len(cs.editGuidanceChoices) {
		return
	}
	choice := cs.editGuidanceChoices[cs.editGuidanceIndex]
	if choice.Preset {
		return
	}
	cs.editSliceItems = removeGuidanceItem(cs.editSliceItems, choice.Path)
	cs.editGuidanceChoices = append(cs.editGuidanceChoices[:cs.editGuidanceIndex], cs.editGuidanceChoices[cs.editGuidanceIndex+1:]...)
	if cs.editGuidanceIndex >= len(cs.editGuidanceChoices) && cs.editGuidanceIndex > 0 {
		cs.editGuidanceIndex--
	}
}

func (cs *configScreen) syncGuidanceChoicesForCurrentField() {
	field := cs.selectedField()
	if field == nil || !cs.editingGuidanceFileChoices(field.Path) {
		return
	}
	cs.editGuidanceChoices = guidanceFileChoicesForField(field.Path, cs.editSliceItems)
	if cs.editGuidanceIndex >= len(cs.editGuidanceChoices) {
		cs.editGuidanceIndex = len(cs.editGuidanceChoices) - 1
	}
	if cs.editGuidanceIndex < 0 {
		cs.editGuidanceIndex = 0
	}
}

func guidanceItemSelected(items []string, path string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == path {
			return true
		}
	}
	return false
}

func removeGuidanceItem(items []string, path string) []string {
	next := items[:0]
	for _, item := range items {
		if strings.TrimSpace(item) == path {
			continue
		}
		next = append(next, item)
	}
	return next
}

func (m Model) renderGuidanceFileChoiceDetail(addLine func(string), field *config.ConfigField) {
	cs := m.configScreen
	addLine(theme.Config.FgCyan + "  Guidance files: (" + fmt.Sprintf("%d selected", len(cs.editSliceItems)) + ")")
	for i, choice := range cs.editGuidanceChoices {
		marker := "[ ]"
		if guidanceItemSelected(cs.editSliceItems, choice.Path) {
			marker = "[x]"
		}
		label := choice.Path
		if !choice.Preset {
			label += " (custom)"
		}
		prefix := "    "
		bg := theme.Config.BgNormal
		if i == cs.editGuidanceIndex {
			prefix = "  > "
			bg = theme.Config.BgInactive
		}
		addLine(bg + theme.Config.FgNormal + prefix + marker + " " + label)
	}
	if cs.editSliceAdding {
		addLine(theme.Config.FgCyan + "  + " + cs.editSliceInput.View())
	}
	addLine("")
	switch field.Path {
	case agentInstructionProjectFilesPath:
		addLine(theme.Config.FgDim + "  AGENTS.md is the default. CLAUDE paths are compatibility choices.")
	case agentInstructionGlobalFilesPath:
		addLine(theme.Config.FgDim + "  ~/.xelyon/AGENTS.md is created empty by default. Add existing assets when needed.")
	}
}
