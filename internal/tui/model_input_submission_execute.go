package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/commandrouter"
)

func (m *Model) resetComposerInput() {
	m.textInput.Reset()
	m.clearSlashSuggestions()
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

	switch commandrouter.Route(command, commandrouter.Context{HasMouseSelection: m.hasActiveMouseSelection()}) {
	case commandrouter.ActionCopyMouseSelection:
		m.resetComposerInput()
		m.appendUserMessage(command.Input)
		m.copyMouseSelection()
		return m, nil
	case commandrouter.ActionQuit:
		m.resetComposerInput()
		m.appendUserMessage(command.Input)
		m.quitting = true
		m.conversation.Cleanup()
		return m, tea.Quit
	case commandrouter.ActionOpenConfig:
		m.resetComposerInput()
		m.appendUserMessage(command.Input)
		updated, cmd := m.openConfigScreen()
		return updated, cmd
	case commandrouter.ActionOpenReview:
		m.resetComposerInput()
		m.appendUserMessage(command.Input)
		instructions := reviewInstructionsFromCommandInput(command.Input)
		if instructions != "" {
			updated, cmd := m.openReviewScreenAndRun(instructions)
			return updated, cmd
		}
		updated, cmd := m.openReviewScreen()
		return updated, cmd
	case commandrouter.ActionOpenProject:
		m.resetComposerInput()
		m.appendUserMessage(command.Input)
		updated, cmd := m.openProjectScreen()
		return updated, cmd
	case commandrouter.ActionDispatchAgent:
		if m.commands.HandleCommand(command.Input) {
			m.resetComposerInput()
			m.appendUserMessage(command.Input)
			m.statusLine = m.conversation.GetStatusLine()
			return m, nil
		}
	}

	return m.handleChatSubmission(composerSubmission{
		kind:    composerSubmissionChat,
		payload: command.Payload,
	})
}

func (m Model) handleChatSubmission(sub composerSubmission) (tea.Model, tea.Cmd) {
	m.clearComposer()
	m.chromeDirty = true // textInput 状態変更を chrome に反映
	m.appendUserMessage(sub.payload)
	return m, m.sendChat(sub.payload)
}

func reviewInstructionsFromCommandInput(input string) string {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(input[len(parts[0]):])
}

// sendChat は goroutine で agent.Chat を呼び出す tea.Cmd を返す。
func (m Model) sendChat(input string) tea.Cmd {
	return func() tea.Msg {
		m.conversation.Chat(input)
		return AgentDoneMsg{}
	}
}
