package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/lifecycle"
)

func (m *Model) switchToComposerInput() {
	if !m.navigationMode {
		return
	}
	m.clearVisualSelection()
	if m.focusedBlock >= 0 {
		m.clearBlockFocus()
	}
	m.navigationMode = false
	m.resetNavPending()
	m.textInput.Focus()
	m.chromeDirty = true
}

func (m *Model) tryEnterNavigationMode() bool {
	if m.hasComposerDraft() {
		return false
	}
	m.navigationMode = true
	m.syncCursorToViewportTop()
	m.textInput.Blur()
	m.chromeDirty = true
	return true
}

func (m Model) handleClipboardPaste() (tea.Model, tea.Cmd) {
	content, err := readClipboardText()
	if err != nil {
		m.setTransientStatus("Paste failed: " + err.Error())
		m.chromeDirty = true
		return m, nil
	}
	if content == "" {
		return m, nil
	}
	m.switchToComposerInput()
	m.handleComposerPaste(content)
	return m, nil
}

func (m Model) handleComposerInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if updated, cmd, handled := m.handleSlashSuggestionKey(msg); handled {
		return updated, cmd
	}

	switch {
	case msg.Type == tea.KeyEsc:
		if m.hasActiveMouseSelection() {
			m.clearMouseSelection()
			m.chromeDirty = true
			return m, nil
		}
		if m.tryEnterNavigationMode() {
			return m, nil
		}

	case isBackspaceKey(msg):
		if m.textInput.Position() == 0 && m.hasFoldedPasteBlocks() {
			if m.removeLastPasteBlock() {
				m.chromeDirty = true
				return m, nil
			}
		}

	case isEnterKey(msg):
		lifecycle.DebugLog("Enter detected, textInput value=%q", m.textInput.Value())
		return m.handleComposerSubmit()

	case msg.Type == tea.KeyUp:
		m.vp.scrollUp(1)
		m.afterViewportScroll()
		return m, nil

	case msg.Type == tea.KeyDown:
		m.vp.scrollDown(1)
		m.afterViewportScroll()
		return m, nil

	case msg.Type == tea.KeyPgUp:
		m.vp.scrollUp(m.vp.height)
		m.afterViewportScroll()
		return m, nil

	case msg.Type == tea.KeyPgDown:
		m.vp.scrollDown(m.vp.height)
		m.afterViewportScroll()
		return m, nil
	}

	// その他のキーは textInput に渡す
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.refreshSlashSuggestions()
	m.chromeDirty = true
	return m, cmd
}

func (m Model) handleComposerSubmit() (tea.Model, tea.Cmd) {
	submission, ok := m.buildComposerSubmission()
	if !ok {
		return m, nil
	}
	return m.submitComposerSubmission(submission)
}
