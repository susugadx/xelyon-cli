package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

// handleSaveCommand はセッション保存を処理
func handleSaveCommand(agent *Agent) bool {
	out := agent.output()

	if agent.storage == nil {
		red.Fprintln(out, "History storage not available")
		return true
	}

	agent.syncSessionPersistenceState()
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

	var session *history.Session
	var err error
	if len(args) > 0 {
		session, err = agent.ResumeSession(args[0])
	} else {
		session, err = agent.ResumeLastSession(history.ResumeListOptions{})
		if errors.Is(err, history.ErrNoResumeSessions) {
			red.Fprintf(out, "No sessions found: %v\n", err)
			return true
		}
	}
	if err != nil {
		red.Fprintf(out, "Failed to load session: %v\n", err)
		return true
	}

	green.Fprintf(out, "📂 Loaded session %s (%d messages)\n", session.ID, len(session.ToAPIMessages()))
	return true
}

func handleResumeCommand(agent *Agent, args []string) bool {
	out := agent.output()
	opts, sessionID, err := parseResumeCommandArgs(args)
	if err != nil {
		yellow.Fprintf(out, "%v\n", err)
		return true
	}

	var session *history.Session
	if sessionID != "" {
		session, err = agent.ResumeSession(sessionID)
	} else {
		session, err = agent.ResumeLastSession(opts)
	}
	if err != nil {
		red.Fprintf(out, "Failed to resume session: %v\n", err)
		return true
	}
	green.Fprintf(out, "📂 Resumed session %s (%d messages)\n", session.ID, len(session.ToAPIMessages()))
	return true
}

// handleSessionsCommand はセッション一覧を表示
func handleSessionsCommand(agent *Agent) bool {
	out := agent.output()

	if agent.storage == nil {
		red.Fprintln(out, "History storage not available")
		return true
	}

	sessions, err := agent.storage.ListResumeSessions(history.ResumeListOptions{All: true})
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
		preview := history.TruncateWithEllipsis(s.Preview, config.SessionPreviewLen)

		_, _ = fmt.Fprintf(out, "  [%s] %s - %s (%d msgs)\n",
			s.ID, timeStr, preview, s.MessageCount)
	}
	_, _ = fmt.Fprintln(out)
	return true
}

func parseResumeCommandArgs(args []string) (history.ResumeListOptions, string, error) {
	var opts history.ResumeListOptions
	var sessionID string
	last := false
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		switch arg {
		case "":
			continue
		case "--all":
			opts.All = true
		case "--last":
			last = true
			continue
		default:
			if strings.HasPrefix(arg, "-") || sessionID != "" {
				return opts, "", fmt.Errorf("usage: /resume [--last|--all|session-id]")
			}
			sessionID = arg
		}
	}
	if last && opts.All {
		return opts, "", fmt.Errorf("--last cannot be used with --all")
	}
	if last && sessionID != "" {
		return opts, "", fmt.Errorf("--last cannot be used with a session ID")
	}
	if opts.All && sessionID != "" {
		return opts, "", fmt.Errorf("--all cannot be used with a session ID")
	}
	return opts, sessionID, nil
}
