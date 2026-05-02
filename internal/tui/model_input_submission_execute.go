package tui

import (
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
		m.recordHandledCommand(command.Input)
		m.copyMouseSelection()
		return m, nil
	case commandrouter.ActionQuit:
		m.recordHandledCommand(command.Input)
		return m.beginQuit()
	case commandrouter.ActionOpenConfig:
		m.recordHandledCommand(command.Input)
		updated, cmd := m.openConfigScreen()
		return updated, cmd
	case commandrouter.ActionOpenReview:
		m.recordHandledCommand(command.Input)
		updated, cmd := m.openReviewScreen()
		return updated, cmd
	case commandrouter.ActionOpenProject:
		m.recordHandledCommand(command.Input)
		updated, cmd := m.openProjectScreen()
		return updated, cmd
	case commandrouter.ActionDispatchAgent:
		if m.commands.HandleCommand(command.Input) {
			m.recordHandledCommand(command.Input)
			m.statusLine = m.conversation.GetStatusLine()
			return m, nil
		}
	}

	return m.handleChatSubmission(sub.fallbackToChat(command.Payload))
}

func (m Model) handleChatSubmission(sub composerSubmission) (tea.Model, tea.Cmd) {
	dispatch := buildChatDispatchRequest(sub.payload, sub.attachments)
	m.clearComposer()
	m.chromeDirty = true // textInput 状態変更を chrome に反映
	m.appendUserMessage(dispatch.display)
	return m, m.sendChat(dispatch)
}

// sendChat は goroutine で agent.Chat を呼び出す tea.Cmd を返す。
func (m Model) sendChat(req chatDispatchRequest) tea.Cmd {
	return func() tea.Msg {
		if req.imagePath != "" {
			m.conversation.ChatWithImagePath(req.input, req.imagePath)
		} else {
			m.conversation.Chat(req.input)
		}
		return AgentDoneMsg{}
	}
}

func (sub composerSubmission) fallbackToChat(payload string) composerSubmission {
	return composerSubmission{
		kind:        composerSubmissionChat,
		payload:     payload,
		attachments: sub.attachments,
	}
}
