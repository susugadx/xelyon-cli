package tui

import tea "github.com/charmbracelet/bubbletea"

// StartupSubmission は TUI 起動直後に transcript へ表示してから実行する初回送信を表す。
type StartupSubmission struct {
	UserMessage string
	Cmd         tea.Cmd
}

type startupSubmissionMsg struct {
	submission StartupSubmission
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
	if msg.submission.UserMessage != "" {
		m.appendUserMessage(msg.submission.UserMessage)
	}
	m.chromeDirty = true
	return m, msg.submission.Cmd
}
