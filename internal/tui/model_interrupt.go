package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleCtrlC() (tea.Model, tea.Cmd) {
	if m.hasActiveMouseSelection() {
		m.copyMouseSelection()
		return m, nil
	}
	if m.agent.IsProcessing() {
		m.agent.Cancel()
		m.appendSystemInfo("⚠️  Interrupted. Press Ctrl+C again to exit.")
		return m, nil
	}

	now := time.Now()
	if !m.lastInterrupt.IsZero() && now.Sub(m.lastInterrupt) < 3*time.Second {
		m.quitting = true
		m.agent.Cleanup()
		return m, tea.Quit
	}

	m.lastInterrupt = now
	m.appendSystemInfo("⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")
	return m, nil
}
