package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// handleSaveCommand はセッション保存を処理
func handleSaveCommand(agent *Agent) bool {
	out := agent.output()

	if agent.storage == nil {
		red.Fprintln(out, "History storage not available")
		return true
	}

	agent.syncApprovedPlanStateToSession()
	agent.syncResponseIDToSession()
	if err := agent.storage.Save(agent.session); err != nil {
		red.Fprintf(out, "Failed to save session: %v\n", err)
		return true
	}

	green.Fprintf(out, "💾 Session saved: %s\n", agent.session.ID)
	return true
}

// handleLoadCommand はセッション読み込みを処理
func handleLoadCommand(agent *Agent, args []string) bool {
	out := agent.output()

	if agent.storage == nil {
		red.Fprintln(out, "History storage not available")
		return true
	}

	sessionID := ""
	if len(args) > 0 {
		sessionID = args[0]
	} else {
		lastID, err := agent.storage.GetLastSession()
		if err != nil {
			red.Fprintf(out, "No sessions found: %v\n", err)
			return true
		}
		sessionID = lastID
	}

	session, err := agent.storage.Load(sessionID)
	if err != nil {
		red.Fprintf(out, "Failed to load session: %v\n", err)
		return true
	}

	agent.restoreSessionConversation(session)

	green.Fprintf(out, "📂 Loaded session %s (%d messages)\n", sessionID, len(session.ToAPIMessages()))
	return true
}

// handleSessionsCommand はセッション一覧を表示
func handleSessionsCommand(agent *Agent) bool {
	out := agent.output()

	if agent.storage == nil {
		red.Fprintln(out, "History storage not available")
		return true
	}

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		red.Fprintf(out, "Failed to list sessions: %v\n", err)
		return true
	}

	if len(sessions) == 0 {
		yellow.Fprintln(out, "No sessions found")
		return true
	}

	cyan.Fprintln(out, "\n📚 Recent Sessions:")
	for i, s := range sessions {
		if i >= config.SessionListMaxDisplay {
			break
		}

		timeStr := s.LastModified.Format("2006-01-02 15:04")
		preview := s.Preview
		if len(preview) > config.SessionPreviewLen {
			preview = preview[:config.SessionPreviewLen] + "..."
		}

		_, _ = fmt.Fprintf(out, "  [%s] %s - %s (%d msgs)\n",
			s.ID, timeStr, preview, s.MessageCount)
	}
	_, _ = fmt.Fprintln(out)
	return true
}
