package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

type agentActivityOptions struct {
	title       string
	workingText string
	doneText    string
	hideStatus  bool
}

func agentActivityDoneCmd() tea.Cmd {
	return func() tea.Msg {
		return AgentDoneMsg{}
	}
}

func newAgentActivityState(startedAt time.Time, opts agentActivityOptions) agentActivityState {
	title := strings.TrimSpace(opts.title)
	if title == "" {
		title = "agent"
	}
	return agentActivityState{
		active:      true,
		title:       termtext.SanitizeSingleLineANSI(title),
		workingText: termtext.SanitizeSingleLineANSI(strings.TrimSpace(opts.workingText)),
		doneText:    termtext.SanitizeSingleLineANSI(strings.TrimSpace(opts.doneText)),
		hideStatus:  opts.hideStatus,
		startedAt:   startedAt,
		status:      agentActivityStatusWorking,
		tools:       nil,
		errorText:   "",
		errorKind:   AgentErrorUnknown,
	}
}
