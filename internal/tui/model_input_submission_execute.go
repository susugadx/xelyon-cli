package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) resetComposerInput() {
	m.textInput.Reset()
	m.chromeDirty = true
}

func (m *Model) appendUserMessage(content string) {
	m.appendMessage(ChatMessage{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (m Model) submitComposerSubmission(sub composerSubmission) (tea.Model, tea.Cmd) {
	switch sub.kind {
	case composerSubmissionCommand:
		return m.handleCommandSubmission(sub)
	case composerSubmissionChat:
		return m.handleChatSubmission(sub)
	default:
		return m, nil
	}
}

func (m Model) handleCommandSubmission(sub composerSubmission) (tea.Model, tea.Cmd) {
	command := m.resolveComposerCommand(sub)

	if command.isBare("/copy") && m.hasActiveMouseSelection() {
		m.resetComposerInput()
		m.appendUserMessage(command.input)
		m.copyMouseSelection()
		return m, nil
	}
	if command.matches("/exit") || command.matches("/quit") {
		m.resetComposerInput()
		m.appendUserMessage(command.input)
		m.quitting = true
		m.agent.Cleanup()
		return m, tea.Quit
	}
	// /config（引数なし）は TUI config screen に遷移
	// alias 解決後に判定（/c 等にも対応）
	if command.isBare("/config") {
		m.resetComposerInput()
		m.appendUserMessage(command.input)
		updated, cmd := m.openConfigScreen()
		return updated, cmd
	}
	if m.agent.HandleCommand(command.input) {
		m.resetComposerInput()
		m.appendUserMessage(command.input)
		m.statusLine = m.agent.GetStatusLine()
		return m, nil
	}

	return m.handleChatSubmission(composerSubmission{
		kind:    composerSubmissionChat,
		payload: command.payload,
	})
}

func (m Model) handleChatSubmission(sub composerSubmission) (tea.Model, tea.Cmd) {
	m.clearComposer()
	m.chromeDirty = true // textInput 状態変更を chrome に反映
	m.appendUserMessage(sub.payload)
	return m, m.sendChat(sub.payload)
}

// sendChat は goroutine で agent.Chat を呼び出す tea.Cmd を返す。
func (m Model) sendChat(input string) tea.Cmd {
	return func() tea.Msg {
		m.agent.Chat(input)
		return AgentDoneMsg{}
	}
}
