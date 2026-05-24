package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update は bubbletea の Update を実装する。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updated, cmd, handled := m.handleRootMessage(msg); handled {
		return updated, cmd
	}
	return m.updateChatScreen(msg)
}

func (m Model) handleRootMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case reviewTimelineRunFinishedMsg:
		updated, cmd := m.handleReviewTimelineFinishedMsg(msg)
		return updated, cmd, true
	case OpenPromptMsg:
		updated, cmd := m.handleOpenPromptMsg(msg)
		return updated, cmd, true
	case CancelPromptMsg:
		updated, cmd := m.handleCancelPromptMsg(msg)
		return updated, cmd, true
	}
	if m.prompt != nil {
		updated, cmd := m.updateWithPromptOpen(msg)
		return updated, cmd, true
	}
	if m.providerPicker != nil {
		updated, cmd := m.updateWithProviderPickerOpen(msg)
		return updated, cmd, true
	}
	if _, ok := msg.(OpenConfigScreenMsg); ok {
		updated, cmd := m.openConfigScreen()
		return updated, cmd, true
	}
	if m.screen == screenConfig {
		updated, cmd := m.updateConfigScreen(msg)
		return updated, cmd, true
	}
	if m.screen == screenReview {
		updated, cmd := m.updateReviewScreen(msg)
		return updated, cmd, true
	}
	if m.screen == screenProject {
		updated, cmd := m.updateProjectScreen(msg)
		return updated, cmd, true
	}
	return m, nil, false
}

func (m Model) updateChatScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		m, cmds = m.handleChatMouseMsg(msg)
	case tea.KeyMsg:
		updated, cmd := m.handleKeyMsg(msg)
		m = updated.(Model)
		cmds = appendCmd(cmds, cmd)
	case tea.WindowSizeMsg:
		m.handleChatWindowSizeMsg(msg)
	case mouseAutoScrollMsg:
		cmds = appendCmd(cmds, m.handleChatAutoScrollMsg())
	case spinner.TickMsg:
		var cmd tea.Cmd
		m, cmd = m.handleSpinnerTickMsg(msg)
		cmds = appendCmd(cmds, cmd)
	case startupSubmissionMsg:
		var cmd tea.Cmd
		m, cmd = m.handleStartupSubmissionMsg(msg)
		cmds = appendCmd(cmds, cmd)
	case startupSubmissionResultMsg:
		var cmd tea.Cmd
		m, cmd = m.handleStartupSubmissionResultMsg(msg)
		cmds = appendCmd(cmds, cmd)
	default:
		if updated, cmd, handled := m.handleStreamMessage(msg); handled {
			m = updated
			cmds = appendCmd(cmds, cmd)
		}
	}

	if shouldUpdateTextInput(msg) {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = appendCmd(cmds, cmd)
	}

	m.rebuildChromeIfDirty()

	return m, tea.Batch(cmds...)
}

func (m Model) handleChatMouseMsg(msg tea.MouseMsg) (Model, []tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 1)

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.vp.scrollUp(3)
		m.afterViewportScroll()
	case tea.MouseButtonWheelDown:
		m.vp.scrollDown(3)
		m.afterViewportScroll()
	default:
		cmds = appendCmd(cmds, m.handleMouseSelection(msg))
	}

	return m, cmds
}

func (m *Model) handleChatWindowSizeMsg(msg tea.WindowSizeMsg) {
	m.mouseDragging = false
	m.mouseAutoScrolling = false
	m.applyChatWindowSize(msg.Width, msg.Height)
}

func (m *Model) handleChatAutoScrollMsg() tea.Cmd {
	cmd := m.handleAutoScroll()
	m.chromeDirty = true
	return cmd
}

func (m Model) handleSpinnerTickMsg(msg spinner.TickMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	if m.conversation.IsProcessing() {
		m.statusSnapshot = m.conversation.StatusSnapshot()
		m.statusLine = m.statusSnapshot.LegacyLine
		if m.statusLine == "" {
			m.statusLine = m.conversation.GetStatusLine()
			m.statusSnapshot.LegacyLine = m.statusLine
		}
	}
	m.updateAgentActivitySnapshot(time.Now())
	m.chromeDirty = true
	return m, cmd
}

func appendCmd(cmds []tea.Cmd, cmd tea.Cmd) []tea.Cmd {
	if cmd == nil {
		return cmds
	}
	return append(cmds, cmd)
}

func shouldUpdateTextInput(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		return false
	default:
		return true
	}
}
