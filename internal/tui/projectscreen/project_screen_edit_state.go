package projectscreen

import (
	"strconv"
	"strings"
)

func (ps *Screen) setDirty() {
	ps.dirty = true
	ps.saveStatus = projectStatusModified
	ps.saveError = ""
	ps.message = ""
}

func (ps *Screen) startContextEdit() {
	if ps.pc == nil {
		return
	}
	ps.pendingClose = false
	ps.contextDraft = ps.pc.Context
	ps.contextArea.SetValue(ps.contextDraft)
	_ = ps.contextArea.Focus()
	ps.editMode = projectEditContext
	ps.message = ""
}

func (ps *Screen) startListEdit(section projectSection, add bool) {
	items := ps.itemsForSection(section)
	value := ""
	if !add && len(items) > 0 {
		value = items[ps.selectedItemIndex()]
	}
	ps.pendingClose = false
	ps.lineEditSection = section
	ps.lineEditKind = projectLineEditList
	ps.lineEditAdd = add
	ps.editInput.SetValue(value)
	ps.editInput.Focus()
	ps.editMode = projectEditLine
	ps.message = ""
}

func (ps *Screen) startTimeoutEdit() {
	if !ps.hasProjectFinalCheckCommands() {
		ps.message = "add a final check command before setting timeout"
		return
	}
	ps.pendingClose = false
	ps.lineEditSection = projectSectionFinalTimeout
	ps.lineEditKind = projectLineEditTimeout
	ps.lineEditAdd = false
	ps.editInput.SetValue(strconv.Itoa(ps.finalChecksTimeoutForDisplay()))
	ps.editInput.Focus()
	ps.editMode = projectEditLine
	ps.message = ""
}

func (ps *Screen) applyContextEdit() {
	if ps.pc == nil {
		return
	}
	ps.pc.Context = ps.contextArea.Value()
	ps.contextArea.Blur()
	ps.editMode = projectEditNone
	ps.setDirty()
}

func (ps *Screen) applyLineEdit() {
	switch ps.lineEditKind {
	case projectLineEditList:
		ps.applyListEdit()
	case projectLineEditTimeout:
		ps.applyTimeoutEdit()
	}
}

func (ps *Screen) applyListEdit() {
	value := strings.TrimSpace(ps.editInput.Value())
	section := ps.lineEditSection
	items := append([]string(nil), ps.itemsForSection(section)...)
	if value == "" {
		ps.cancelEdit()
		return
	}
	if ps.lineEditAdd || len(items) == 0 {
		items = append(items, value)
		ps.itemIndex[section] = len(items) - 1
	} else {
		items[ps.selectedItemIndex()] = value
	}
	ps.setItemsForSection(section, items)
	ps.finishLineEdit()
	ps.setDirty()
}

func (ps *Screen) applyTimeoutEdit() {
	if !ps.hasProjectFinalCheckCommands() {
		ps.cancelEdit()
		ps.message = "add a final check command before setting timeout"
		return
	}
	value := strings.TrimSpace(ps.editInput.Value())
	timeout, err := strconv.Atoi(value)
	if err != nil || timeout <= 0 {
		ps.message = "timeout must be a positive integer"
		return
	}
	ps.pc.FinalChecks.Timeout = timeout
	ps.finishLineEdit()
	ps.setDirty()
}

func (ps *Screen) finishLineEdit() {
	ps.editInput.Blur()
	ps.editMode = projectEditNone
	ps.lineEditKind = projectLineEditNone
	ps.lineEditAdd = false
	ps.message = ""
}

func (ps *Screen) cancelEdit() {
	ps.editInput.Blur()
	ps.contextArea.Blur()
	ps.editMode = projectEditNone
	ps.lineEditKind = projectLineEditNone
	ps.lineEditAdd = false
	ps.message = ""
}

func (ps *Screen) tryClose() Command {
	if ps.dirty {
		ps.confirmQuit = true
		ps.confirmIdx = 0
		return CommandNone
	}
	return CommandClose
}
