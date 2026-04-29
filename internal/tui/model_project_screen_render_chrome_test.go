package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectScreen_CloseAfterResize_RebuildsChatFooter(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
		},
	}
	m := newProjectTestModel(agent)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = updated.(Model)
	m = sendProjectKey(m, "esc")

	if m.screen != screenChat {
		t.Fatalf("screen after close = %d, want screenChat", m.screen)
	}
	if m.chromeDirty {
		t.Fatal("chromeDirty should be false because closeProjectScreen rebuilds chrome immediately")
	}
	verifyViewLines(t, m, "project close after resize")
}
