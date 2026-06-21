package projectscreen

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/keyinput"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func isEnterKey(msg tea.KeyMsg) bool {
	return keyinput.IsEnterKey(msg)
}

func projectPaneColors(selected, active bool) (string, string) {
	switch {
	case selected && active:
		return theme.Config.BgSelected, theme.Config.FgBright
	case selected:
		return theme.Config.BgInactive, theme.Config.FgNormal
	default:
		return theme.Config.BgNormal, theme.Config.FgNormal
	}
}
