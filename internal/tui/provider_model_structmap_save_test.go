package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func TestConfigScreen_Save_AfterEntryEdit_UsesUpdatedConfig(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)

	enterConfigStructMapEdit(t, &m, "provider_models")
	selectConfigStructMapKey(t, &m, "openai")

	m = sendConfigKey(m, "enter")

	selectConfigEntryField(t, &m, "default_model")
	m = sendConfigKey(m, "enter")
	setConfigInputValue(t, &m, "save-test-model")
	m = sendConfigKey(m, "enter")

	m = sendConfigKey(m, "esc")
	m = sendConfigKey(m, "esc")

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated2, saveCmd := m.Update(sMsg)
	m = updated2.(Model)
	cs := configTestScreen(t, m)
	if got := cs.Snapshot().SaveStatus; got != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving", got)
	}

	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	resultMsg := saveCmd()

	updated3, _ := m.Update(resultMsg)
	m = updated3.(Model)
	cs = configTestScreen(t, m)

	snapshot := cs.Snapshot()
	if snapshot.Dirty {
		t.Fatal("dirty should be false after save")
	}
	if snapshot.SaveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved", snapshot.SaveStatus)
	}

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()

	if saved == nil {
		t.Fatal("lastSavedConfig is nil — SaveAndSyncConfig was not called")
	}
	pm, ok := saved.ProviderModels["openai"]
	if !ok {
		t.Fatal("openai not found in saved ProviderModels")
	}
	if pm.DefaultModel != "save-test-model" {
		t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want \"save-test-model\"", pm.DefaultModel)
	}
}

func TestConfigScreen_ProviderOverride_Save_NotOverwritten(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	cfg.DefaultModel = "global-model"
	if pm, ok := cfg.ProviderModels["openai"]; ok {
		pm.DefaultModel = "provider-override"
		cfg.ProviderModels["openai"] = pm
	}
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)
	setConfigDirtyForTest(t, &m, true)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, saveCmd := m.Update(sMsg)
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd is nil")
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

	pm := saved.ProviderModels["openai"]
	if pm.DefaultModel != "provider-override" {
		t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want \"provider-override\"", pm.DefaultModel)
	}

	if saved.DefaultModel != "global-model" {
		t.Fatalf("saved DefaultModel = %q, want \"global-model\"", saved.DefaultModel)
	}
}
