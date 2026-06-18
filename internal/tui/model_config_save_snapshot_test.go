package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_SaveSnapshot_Isolation(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	cs := m.configScreen

	cs.cfg.DefaultModel = "gpt-5.4"
	setConfigDirtyForTest(t, &m, true)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, saveCmd := m.Update(sMsg)
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.DefaultModel != "gpt-5.4" {
		t.Fatalf("saved.DefaultModel = %q, want \"gpt-5.4\"", saved.DefaultModel)
	}

	m.configScreen.cfg.DefaultModel = "gpt-5.4-mini"

	agent.mu.RLock()
	savedAfter := agent.lastSavedConfig
	agent.mu.RUnlock()
	if savedAfter.DefaultModel != "gpt-5.4" {
		t.Fatalf("savedAfter.DefaultModel = %q, want \"gpt-5.4\" (snapshot should be isolated)", savedAfter.DefaultModel)
	}

	if pm, ok := m.configScreen.cfg.ProviderModels["openai"]; ok {
		pm.DefaultModel = "mutated"
		m.configScreen.cfg.ProviderModels["openai"] = pm
	}
	agent.mu.RLock()
	savedPM := agent.lastSavedConfig.ProviderModels["openai"]
	agent.mu.RUnlock()
	if savedPM.DefaultModel == "mutated" {
		t.Fatal("saved ProviderModels[openai].DefaultModel was mutated — snapshot is not isolated")
	}
}

func TestConfigScreen_SaveCmd_SnapshotIsolation(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	m.configScreen.cfg.DefaultModel = "at-save-time"
	setConfigDirtyForTest(t, &m, true)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, saveCmd := m.Update(sMsg)
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd is nil")
	}

	m.configScreen.cfg.DefaultModel = "mutated-after-save"

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.DefaultModel != "at-save-time" {
		t.Fatalf("saved.DefaultModel = %q, want \"at-save-time\" (snapshot should be isolated from post-save mutation)", saved.DefaultModel)
	}
}
