package projectscreen

func (ps *Screen) canUseItemPane() bool {
	return projectSectionIsList(ps.selectedSection())
}

func projectSectionIsList(section projectSection) bool {
	switch section {
	case projectSectionRules, projectSectionIgnore, projectSectionFinalCommands:
		return true
	default:
		return false
	}
}

func (ps *Screen) selectedItems() []string {
	return ps.itemsForSection(ps.selectedSection())
}

func (ps *Screen) itemsForSection(section projectSection) []string {
	if ps.pc == nil {
		return nil
	}
	switch section {
	case projectSectionRules:
		return ps.pc.Rules
	case projectSectionIgnore:
		return ps.pc.Ignore.Patterns
	case projectSectionFinalCommands:
		if ps.pc.FinalChecks == nil {
			return nil
		}
		return ps.pc.FinalChecks.Commands
	default:
		return nil
	}
}

func (ps *Screen) setItemsForSection(section projectSection, items []string) {
	if ps.pc == nil {
		return
	}
	switch section {
	case projectSectionRules:
		ps.pc.Rules = append([]string(nil), items...)
	case projectSectionIgnore:
		ps.pc.Ignore.Patterns = append([]string(nil), items...)
	case projectSectionFinalCommands:
		ps.ensureFinalChecks()
		ps.pc.FinalChecks.Commands = append([]string(nil), items...)
		ps.clearTUIOnlyFinalChecksWithoutCommands()
	}
}

func (ps *Screen) selectedItemIndex() int {
	section := ps.selectedSection()
	items := ps.itemsForSection(section)
	idx := ps.itemIndex[section]
	if idx < 0 {
		idx = 0
	}
	if len(items) == 0 {
		idx = 0
	} else if idx >= len(items) {
		idx = len(items) - 1
	}
	ps.itemIndex[section] = idx
	return idx
}

func (ps *Screen) deleteSelectedItem() {
	section := ps.selectedSection()
	items := append([]string(nil), ps.itemsForSection(section)...)
	if len(items) == 0 {
		return
	}
	idx := ps.selectedItemIndex()
	items = append(items[:idx], items[idx+1:]...)
	ps.setItemsForSection(section, items)
	if idx >= len(items) && idx > 0 {
		idx--
	}
	ps.itemIndex[section] = idx
	ps.setDirty()
}
