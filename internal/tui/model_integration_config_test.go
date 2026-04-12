package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIIntegration_ConfigAliasToggleSaveAndCloseFlow(t *testing.T) {
	agent := &stubAgent{
		statusLine:     "provider: deepseek model: deepseek-chat",
		saveStatusLine: "provider: openai model: gpt-5.4",
	}
	m := newModelWithViewport(agent)
	m.statusLine = agent.GetStatusLine()
	m.textInput.SetValue("/c")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("/c should open config without async cmd, got %v", cmd)
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Content != "/c" {
		t.Fatalf("last message = %#v, want original alias command /c", m.messages)
	}

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "compression", "compression.enabled")
	selected := cs.selectedField()
	before, ok := selected.Current.(bool)
	if !ok {
		t.Fatalf("selected field current = %T, want bool", selected.Current)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updated.(Model)
	if !m.configScreen.dirty {
		t.Fatal("configScreen should become dirty after toggle")
	}
	after, ok := m.configScreen.selectedField().Current.(bool)
	if !ok {
		t.Fatalf("selected field current after toggle = %T, want bool", m.configScreen.selectedField().Current)
	}
	if after == before {
		t.Fatalf("compression.enabled = %v after toggle, want %v", after, !before)
	}

	m = sendConfigKey(m, "q")
	if !m.configScreen.confirmQuit {
		t.Fatal("confirmQuit should be shown for dirty config")
	}

	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	if m.configScreen.saveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving", m.configScreen.saveStatus)
	}

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)

	if m.screen != screenChat {
		t.Fatalf("screen = %d after save-and-close, want screenChat", m.screen)
	}
	if m.statusLine != "provider: openai model: gpt-5.4" {
		t.Fatalf("statusLine = %q, want updated runtime status", m.statusLine)
	}

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.Compression.Enabled != !before {
		t.Fatalf("saved.Compression.Enabled = %v, want %v", saved.Compression.Enabled, !before)
	}
}
