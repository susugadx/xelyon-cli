package tuiagent

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

// InitialImageStartupSubmission は TUI 起動後に実行する画像付き初回 turn を作成する。
func InitialImageStartupSubmission(adapter *TUIAdapter, query string, image *api.ImageData) *tui.StartupSubmission {
	if image == nil {
		return nil
	}
	return &tui.StartupSubmission{
		UserMessage: query,
		Cmd: func() tea.Msg {
			err := adapter.ChatWithImage(query, image)
			return tui.AgentDoneMsg{
				Error:     err,
				ErrorKind: tui.AgentErrorKindFromError(err, tui.AgentErrorProvider),
			}
		},
	}
}
