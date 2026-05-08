package tui

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// StartupSubmission は TUI 起動直後に transcript へ表示してから実行する初回送信を表す。
type StartupSubmission struct {
	UserMessage string
	Cmd         tea.Cmd
}

type startupSubmissionMsg struct {
	submission StartupSubmission
}

type startupSubmissionResultMsg struct {
	msg tea.Msg
}

func startupSubmissionCmd(submission *StartupSubmission) tea.Cmd {
	if submission == nil {
		return nil
	}
	copied := *submission
	return func() tea.Msg {
		return startupSubmissionMsg{submission: copied}
	}
}

func (m Model) handleStartupSubmissionMsg(msg startupSubmissionMsg) (Model, tea.Cmd) {
	m = m.appendStartupAgentTurn(msg.submission)
	return m, wrapStartupSubmissionActivityCmd(msg.submission.Cmd)
}

func wrapStartupSubmissionActivityCmd(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() (result tea.Msg) {
		defer func() {
			if recovered := recover(); recovered != nil {
				result = startupSubmissionResultMsg{
					msg: AgentDoneMsg{
						Error:     fmt.Errorf("startup command failed: %v", recovered),
						ErrorKind: AgentErrorStartup,
					},
				}
			}
		}()
		return startupSubmissionResultMsg{msg: cmd()}
	}
}

func (m Model) handleStartupSubmissionResultMsg(msg startupSubmissionResultMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg.msg != nil {
		if updated, next, handled := m.handleStreamMessage(msg.msg); handled {
			m = updated
			cmd = next
		}
	}
	if m.hasActiveAgentActivity() {
		m.handleAgentDoneMsg(AgentDoneMsg{
			Error:     startupSubmissionMissingDoneError(msg.msg),
			ErrorKind: AgentErrorStartup,
		})
	}
	return m, cmd
}

func startupSubmissionMissingDoneError(msg tea.Msg) error {
	if msg == nil {
		return errors.New("startup command did not report completion")
	}
	return fmt.Errorf("startup command returned %T without completion", msg)
}
