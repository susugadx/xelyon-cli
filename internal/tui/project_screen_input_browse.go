package tui

import tea "github.com/charmbracelet/bubbletea"

func (ps *projectScreen) handleBrowseKey(msg tea.KeyMsg) projectCommand {
	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc || s == "q":
		if ps.activePane == projectPaneItem {
			ps.activePane = projectPaneSection
			return projectCommandNone
		}
		return ps.tryClose()
	case s == "s":
		if ps.dirty {
			return projectCommandSave
		}
		ps.message = "no changes"
	case msg.Type == tea.KeyTab || msg.Type == tea.KeyRight || s == "l":
		if ps.canUseItemPane() {
			ps.activePane = projectPaneItem
		}
	case msg.Type == tea.KeyLeft || s == "h":
		ps.activePane = projectPaneSection
	case msg.Type == tea.KeyUp || s == "k":
		ps.moveSelection(-1)
	case msg.Type == tea.KeyDown || s == "j":
		ps.moveSelection(1)
	case s == "a":
		if projectSectionIsList(ps.selectedSection()) {
			ps.startListEdit(ps.selectedSection(), true)
		}
	case s == "d":
		if ps.activePane == projectPaneItem && projectSectionIsList(ps.selectedSection()) {
			ps.deleteSelectedItem()
		}
	case isEnterKey(msg):
		ps.handleBrowseEnter()
	}
	return projectCommandNone
}

func (ps *projectScreen) moveSelection(delta int) {
	if ps.activePane == projectPaneItem && ps.canUseItemPane() {
		section := ps.selectedSection()
		total := len(ps.itemsForSection(section))
		if total == 0 {
			ps.itemIndex[section] = 0
			return
		}
		idx := ps.selectedItemIndex() + delta
		if idx < 0 {
			idx = 0
		}
		if idx >= total {
			idx = total - 1
		}
		ps.itemIndex[section] = idx
		return
	}

	ps.sectionIndex += delta
	if ps.sectionIndex < 0 {
		ps.sectionIndex = 0
	}
	if ps.sectionIndex >= len(projectSections) {
		ps.sectionIndex = len(projectSections) - 1
	}
	if !ps.canUseItemPane() {
		ps.activePane = projectPaneSection
	}
}

func (ps *projectScreen) handleBrowseEnter() {
	section := ps.selectedSection()
	switch section {
	case projectSectionContext:
		ps.startContextEdit()
	case projectSectionRules, projectSectionIgnore, projectSectionFinalCommands:
		if ps.activePane == projectPaneSection {
			ps.activePane = projectPaneItem
			return
		}
		items := ps.itemsForSection(section)
		ps.startListEdit(section, len(items) == 0)
	case projectSectionFinalTimeout:
		ps.startTimeoutEdit()
	case projectSectionConditional:
		ps.message = "conditional editing is preview-only in this screen"
	}
}
