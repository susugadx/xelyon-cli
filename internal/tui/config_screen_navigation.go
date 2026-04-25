package tui

func (cs *configScreen) handleBrowseEscape() configCommand {
	switch cs.activePane {
	case paneField:
		cs.activePane = paneCategory
		cs.fieldIndex = 0
		cs.fieldScroll = 0
	case paneDetail:
		cs.activePane = paneField
	default:
		return cs.tryClose()
	}
	return configCommandNone
}

func (cs *configScreen) navUp(layout configLayout) {
	switch cs.activePane {
	case paneCategory:
		cs.selectPreviousCategory()
	case paneField:
		cs.selectPreviousField(layout)
	}
}

func (cs *configScreen) navDown(layout configLayout) {
	switch cs.activePane {
	case paneCategory:
		cs.selectNextCategory()
	case paneField:
		cs.selectNextField(layout)
	}
}

func (cs *configScreen) navLeft() {
	switch cs.activePane {
	case paneField:
		cs.activePane = paneCategory
	case paneDetail:
		cs.activePane = paneField
	}
}

func (cs *configScreen) navRight(layout configLayout) {
	switch cs.activePane {
	case paneCategory:
		cs.activePane = paneField
	case paneField:
		if layout.DetailVisible() {
			cs.activePane = paneDetail
		}
	}
}

func (cs *configScreen) selectPreviousCategory() {
	if cs.catIndex <= 0 {
		return
	}
	cs.catIndex--
	cs.resetFieldSelection()
	cs.filterText = ""
}

func (cs *configScreen) selectNextCategory() {
	if cs.catIndex >= len(cs.categories)-1 {
		return
	}
	cs.catIndex++
	cs.resetFieldSelection()
	cs.filterText = ""
}

func (cs *configScreen) selectPreviousField(layout configLayout) {
	fields := cs.filteredFields()
	if cs.fieldIndex > 0 {
		cs.fieldIndex--
	} else if len(fields) > 0 {
		cs.fieldIndex = len(fields) - 1
	}
	cs.ensureFieldVisible(layout.FieldPaneVisibleRows(cs.filterMode || cs.filterText != ""))
}

func (cs *configScreen) selectNextField(layout configLayout) {
	fields := cs.filteredFields()
	if cs.fieldIndex < len(fields)-1 {
		cs.fieldIndex++
	} else if len(fields) > 0 {
		cs.fieldIndex = 0
	}
	cs.ensureFieldVisible(layout.FieldPaneVisibleRows(cs.filterMode || cs.filterText != ""))
}

func (cs *configScreen) resetFieldSelection() {
	cs.fieldIndex = 0
	cs.fieldScroll = 0
}

// ensureFieldVisible は fieldIndex が可視範囲内に入るよう fieldScroll を調整する。
func (cs *configScreen) ensureFieldVisible(visibleRows int) {
	if visibleRows <= 0 {
		return
	}
	if cs.fieldIndex < cs.fieldScroll {
		cs.fieldScroll = cs.fieldIndex
	}
	if cs.fieldIndex >= cs.fieldScroll+visibleRows {
		cs.fieldScroll = cs.fieldIndex - visibleRows + 1
	}
	if cs.fieldScroll < 0 {
		cs.fieldScroll = 0
	}
}
