package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type openSessionPickerMsg struct {
	all     bool
	startup bool
}

func openSessionPickerCmd(all bool, startup bool) tea.Cmd {
	return func() tea.Msg {
		return openSessionPickerMsg{all: all, startup: startup}
	}
}

func (m Model) openSessionPicker(all bool, startup bool) (Model, tea.Cmd) {
	m.switchToComposerInput()
	candidates, err := m.sessions.ResumeSessionCandidates(SessionResumeOptions{All: all})
	if err != nil {
		m.setTransientStatus(err.Error())
		return m, nil
	}
	if len(candidates) == 0 {
		m.setTransientStatus("No sessions found")
		return m, nil
	}
	m.sessionPicker = newSessionPickerState(candidates, all, startup)
	m.clearSlashSuggestions()
	m.chromeDirty = true
	return m, nil
}

func (m Model) updateWithSessionPickerOpen(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleSessionPickerKeyMsg(msg)
	case tea.MouseMsg:
		if isPromptBackgroundWheelMsg(msg) {
			return m.forwardMessageBehindSessionPicker(msg)
		}
		return m, nil
	default:
		return m.forwardMessageBehindSessionPicker(msg)
	}
}

func (m Model) forwardMessageBehindSessionPicker(msg tea.Msg) (Model, tea.Cmd) {
	active := m.sessionPicker
	m.sessionPicker = nil

	updated, cmd := m.Update(msg)
	next := updated.(Model)
	next.sessionPicker = active
	return next, cmd
}

func (m Model) handleSessionPickerKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.sessionPicker == nil {
		return m, nil
	}

	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		m.closeSessionPicker("Resume cancelled")
	case msg.Type == tea.KeyUp || (!m.sessionPicker.filtering && msg.String() == "k"):
		m.sessionPicker.moveSelection(-1)
	case msg.Type == tea.KeyDown || (!m.sessionPicker.filtering && msg.String() == "j"):
		m.sessionPicker.moveSelection(1)
	case msg.String() == "/":
		m.sessionPicker.filtering = true
		m.sessionPicker.filter = ""
		m.sessionPicker.selected = 0
	case isBackspaceKey(msg):
		m.handleSessionPickerBackspace()
	case isEnterKey(msg):
		return m.submitSessionPickerSelection()
	case m.sessionPicker.filtering && msg.Type == tea.KeyRunes:
		m.sessionPicker.filter += string(msg.Runes)
		m.sessionPicker.selected = 0
	}

	m.sessionPicker.clampSelection()
	m.chromeDirty = true
	return m, nil
}

func (m *Model) handleSessionPickerBackspace() {
	if !m.sessionPicker.filtering || m.sessionPicker.filter == "" {
		return
	}
	runes := []rune(m.sessionPicker.filter)
	m.sessionPicker.filter = string(runes[:len(runes)-1])
	m.sessionPicker.selected = 0
	m.sessionPicker.clampSelection()
	m.chromeDirty = true
}

func (m Model) submitSessionPickerSelection() (Model, tea.Cmd) {
	row, ok := m.sessionPicker.selectedSession()
	if !ok {
		return m, nil
	}
	if m.rejectCrossWorkingDirSessionSelection(row) {
		return m, nil
	}
	return m.resumeSessionByID(row.ID)
}

func (m *Model) rejectCrossWorkingDirSessionSelection(row SessionCandidate) bool {
	if m.sessionPicker == nil || m.sessionPicker.all {
		return false
	}
	currentDir := normalizeSessionPickerWorkingDir(m.workingDir)
	sessionDir := normalizeSessionPickerWorkingDir(row.WorkingDir)
	if currentDir == "" || sessionDir == "" || currentDir == sessionDir {
		return false
	}
	m.setTransientStatus("Cannot resume session from a different working directory: " + row.WorkingDir)
	m.chromeDirty = true
	return true
}

func normalizeSessionPickerWorkingDir(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func (m Model) resumeSessionByID(sessionID string) (Model, tea.Cmd) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		m.setTransientStatus("Session ID is required")
		return m, nil
	}
	if m.sessionPicker != nil && m.sessionPicker.startup {
		if err := m.sessions.ResumeStartupSession(sessionID); err != nil {
			m.setTransientStatus(err.Error())
			return m, nil
		}
	} else if err := m.sessions.ResumeSession(sessionID); err != nil {
		m.setTransientStatus(err.Error())
		return m, nil
	}
	m.sessionPicker = nil
	m.clearPendingAttachmentsForSessionSwitch()
	m.resetVisibleTranscript()
	m.appendSystemNotice("Resumed session " + shortSessionID(sessionID))
	m.refreshStatusLine()
	m.setTransientStatus("Session resumed")
	return m, nil
}

func (m Model) resumeLastSession(all bool) (Model, tea.Cmd) {
	session, err := m.sessions.ResumeLastSession(SessionResumeOptions{All: all})
	if err != nil {
		m.setTransientStatus(err.Error())
		return m, nil
	}
	if strings.TrimSpace(session.ID) == "" {
		m.setTransientStatus("No sessions found")
		return m, nil
	}
	m.sessionPicker = nil
	m.clearPendingAttachmentsForSessionSwitch()
	m.resetVisibleTranscript()
	m.appendSystemNotice("Resumed session " + shortSessionID(session.ID))
	m.refreshStatusLine()
	m.setTransientStatus("Session resumed")
	return m, nil
}

func (m Model) startNewSessionFromCommand(clearVisible bool) (Model, tea.Cmd) {
	sessionID, err := m.sessions.StartNewSession()
	if err != nil {
		m.setTransientStatus(err.Error())
		return m, nil
	}
	if clearVisible {
		m.resetVisibleTranscript()
	}
	m.clearPendingAttachmentsForSessionSwitch()
	m.appendSystemNotice("Started new session " + shortSessionID(sessionID))
	m.refreshStatusLine()
	m.setTransientStatus("New session started")
	return m, nil
}

func (m *Model) clearPendingAttachmentsForSessionSwitch() {
	m.clearAttachments()
	m.syncComposerLayout()
}

func (m *Model) closeSessionPicker(status string) {
	m.sessionPicker = nil
	m.setTransientStatus(status)
	m.chromeDirty = true
}

func (m *Model) resetVisibleTranscript() {
	m.messages = nil
	m.rawLines = nil
	m.toolBlocks = nil
	m.agentActivity = agentActivityState{}
	m.focusedBlock = -1
	m.newOutput = false
	m.streamingActive = false
	m.streamCursorCol = 0
	m.streamActiveANSI = ""
	m.streamPendingANSI = ""
	m.navigationState = navigationState{
		visualStart: visualPosition{line: -1, col: -1},
	}
	m.mouseSelectionState = mouseSelectionState{
		mouseSelAnchor: visualPosition{line: -1, col: -1},
		mouseSelEnd:    visualPosition{line: -1, col: -1},
	}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoBottom()
	m.chromeDirty = true
}

func (m *Model) appendSystemNotice(text string) {
	m.appendMessage(ChatMessage{
		Role:    "system_info",
		Content: text,
	})
}

func shortSessionID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return fmt.Sprintf("%s...", id[:8])
}
