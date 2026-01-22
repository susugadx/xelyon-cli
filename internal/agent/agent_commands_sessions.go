package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// handleSaveCommand はセッション保存を処理
func handleSaveCommand(agent *Agent) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	if err := agent.storage.Save(agent.session); err != nil {
		red.Printf("Failed to save session: %v\n", err)
		return true
	}

	green.Printf("💾 Session saved: %s\n", agent.session.ID)
	return true
}

// handleLoadCommand はセッション読み込みを処理
func handleLoadCommand(agent *Agent, args []string) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	sessionID := ""
	if len(args) > 0 {
		sessionID = args[0]
	} else {
		lastID, err := agent.storage.GetLastSession()
		if err != nil {
			red.Printf("No sessions found: %v\n", err)
			return true
		}
		sessionID = lastID
	}

	session, err := agent.storage.Load(sessionID)
	if err != nil {
		red.Printf("Failed to load session: %v\n", err)
		return true
	}

	// セッション置き換え
	agent.session = session
	agent.History = session.ToAPIMessages()

	green.Printf("📂 Loaded session %s (%d messages)\n", sessionID, len(session.Messages))
	return true
}

// handleSessionsCommand はセッション一覧を表示
func handleSessionsCommand(agent *Agent) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		red.Printf("Failed to list sessions: %v\n", err)
		return true
	}

	if len(sessions) == 0 {
		yellow.Println("No sessions found")
		return true
	}

	cyan.Println("\n📚 Recent Sessions:")
	for i, s := range sessions {
		if i >= config.SessionListMaxDisplay {
			break
		}

		timeStr := s.LastModified.Format("2006-01-02 15:04")
		preview := s.Preview
		if len(preview) > config.SessionPreviewLen {
			preview = preview[:config.SessionPreviewLen] + "..."
		}

		fmt.Printf("  [%s] %s - %s (%d msgs)\n",
			s.ID, timeStr, preview, s.MessageCount)
	}
	fmt.Println()
	return true
}
