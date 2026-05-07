package agent

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

func initialImageStartupSubmission(adapter *TUIAdapter, query string, image *api.ImageData) *tui.StartupSubmission {
	if image == nil {
		return nil
	}
	return &tui.StartupSubmission{
		UserMessage: query,
		Cmd: func() tea.Msg {
			return tui.AgentDoneMsg{Error: adapter.ChatWithImage(query, image)}
		},
	}
}

func loadLastSessionForTUI(agent *Agent) {
	out := agent.output()
	if agent.storage == nil {
		red.Fprintln(out, "History storage not available")
		return
	}

	sessionID, err := agent.storage.GetLastSession()
	if err != nil {
		yellow.Fprintln(out, "No previous session found, starting new session")
		return
	}

	session, err := agent.storage.Load(sessionID)
	if err != nil {
		red.Fprintf(out, "Failed to load session: %v\n", err)
		return
	}

	agent.applyLoadedSession(session)
	green.Fprintf(out, "📂 Resumed session %s (%d messages)\n", sessionID, len(session.ToAPIMessages()))
}
