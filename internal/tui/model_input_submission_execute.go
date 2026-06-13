package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/commandrouter"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
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
	decision := decideCommandSubmission(command, m.hasActiveMouseSelection())

	switch decision.kind {
	case commandSubmissionDecisionLocalSyntaxError:
		m.recordHandledCommand(command.Input)
		m.setTransientStatus("Invalid command syntax: " + decision.errorDetail)
		return m, nil
	case commandSubmissionDecisionLocalAction:
		if localActionRequiresIdleAgent(decision.action) && m.rejectAgentTurnWhileBusy() {
			return m, nil
		}
		handler := localCommandActionHandlers[decision.action]
		return handler(m, command, sub)
	case commandSubmissionDecisionDispatchAgent:
		return m.handleAgentCommandSubmission(sub, command)
	default:
		return m.commandFallbackToChat(sub, command.Payload)
	}
}

func (m Model) handleAgentCommandSubmission(sub composerSubmission, command slash.Command) (tea.Model, tea.Cmd) {
	if m.commands.HandleCommand(command.Input) {
		m.recordHandledCommand(command.Input)
		m.refreshStatusLine()
		if m.agentTurnBusy() {
			return m, nil
		}
		m.beginAgentActivity()
		return m, agentActivityDoneCmd()
	}
	return m.commandFallbackToChat(sub, command.Payload)
}

type localCommandActionHandler func(Model, slash.Command, composerSubmission) (tea.Model, tea.Cmd)

func localActionRequiresIdleAgent(action commandrouter.Action) bool {
	switch action {
	case commandrouter.ActionNewSession, commandrouter.ActionOpenSessionPicker:
		return true
	default:
		return false
	}
}

var localCommandActionHandlers = map[commandrouter.Action]localCommandActionHandler{
	commandrouter.ActionCopyMouseSelection: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		m.copyMouseSelection()
		return m, nil
	},
	commandrouter.ActionManageAttachments: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		return m.handleAttachmentCommandSubmission(command)
	},
	commandrouter.ActionQuit: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		return m.beginQuit()
	},
	commandrouter.ActionOpenConfig: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		updated, cmd := m.openConfigScreen()
		return updated, cmd
	},
	commandrouter.ActionOpenReview: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		instructions := command.RawArgText()
		if instructions != "" {
			return m.openReviewScreenAndRun(instructions)
		}
		m.recordHandledCommand(command.Input)
		return m.openReviewScreen()
	},
	commandrouter.ActionOpenProject: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		updated, cmd := m.openProjectScreen()
		return updated, cmd
	},
	commandrouter.ActionOpenProviderPicker: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		updated, cmd := m.openProviderPicker()
		return updated, cmd
	},
	commandrouter.ActionOpenModelPicker: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		updated, cmd := m.openCurrentProviderModelPicker()
		return updated, cmd
	},
	commandrouter.ActionNewSession: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		return m.startNewSessionFromCommand(command.ResolvedName == "/clear")
	},
	commandrouter.ActionOpenSessionPicker: func(m Model, command slash.Command, _ composerSubmission) (tea.Model, tea.Cmd) {
		m.recordHandledCommand(command.Input)
		return m.handleResumeCommandSubmission(command)
	},
}

func (m Model) handleResumeCommandSubmission(command slash.Command) (tea.Model, tea.Cmd) {
	all := false
	last := false
	sessionID := ""
	for _, arg := range command.Args {
		switch arg {
		case "--all":
			all = true
		case "--last":
			last = true
		default:
			if strings.HasPrefix(arg, "-") {
				m.setTransientStatus("Unknown resume option: " + arg)
				return m, nil
			}
			if sessionID != "" {
				m.setTransientStatus("Resume accepts one session ID")
				return m, nil
			}
			sessionID = arg
		}
	}
	if last && all {
		m.setTransientStatus("--last cannot be used with --all")
		return m, nil
	}
	if last && sessionID != "" {
		m.setTransientStatus("--last cannot be used with a session ID")
		return m, nil
	}
	if all && sessionID != "" {
		m.setTransientStatus("--all cannot be used with a session ID")
		return m, nil
	}
	if last {
		return m.resumeLastSession(all)
	}
	if sessionID != "" {
		return m.resumeSessionByID(sessionID)
	}
	return m.openSessionPicker(all)
}

func (m Model) handleChatSubmission(sub composerSubmission) (tea.Model, tea.Cmd) {
	if m.rejectAgentTurnWhileBusy() {
		return m, nil
	}
	dispatch := buildChatDispatchRequest(sub.payload, sub.attachments)
	cleanupTargets := m.detachAttachmentsWithoutCleanup()
	m.clearComposerDraft()
	dispatch.cleanupAttachments = cleanupTargets
	m = m.appendChatAgentTurn(dispatch.display)
	return m, m.sendChat(dispatch)
}

// sendChat は goroutine で agent.Chat を呼び出す tea.Cmd を返す。
func (m Model) sendChat(req chatDispatchRequest) tea.Cmd {
	return func() tea.Msg {
		var err error
		if req.imagePath != "" {
			err = m.conversation.ChatWithImagePath(req.input, req.imagePath)
		} else {
			err = m.conversation.Chat(req.input)
		}
		for _, att := range req.cleanupAttachments {
			cleanupTemporaryAttachment(att)
		}
		return AgentDoneMsg{Error: err, ErrorKind: AgentErrorKindFromError(err, AgentErrorUnknown)}
	}
}

func (m Model) commandFallbackToChat(sub composerSubmission, payload string) (tea.Model, tea.Cmd) {
	return m.handleChatSubmission(sub.fallbackToChatExcludingAttachments(payload))
}

func (sub composerSubmission) fallbackToChatExcludingAttachments(payload string) composerSubmission {
	return composerSubmission{
		kind:        composerSubmissionChat,
		payload:     payload,
		attachments: nil,
	}
}
