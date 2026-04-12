package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_Save_AfterEntryEdit_UsesUpdatedConfig(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)

	cs := m.configScreen
	for i, cat := range cs.categories {
		if cat.Name == "provider" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	for i, f := range cs.filteredFields() {
		if f.Path == "provider_models" {
			cs.fieldIndex = i
			break
		}
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	for i, k := range cs.editStructKeys {
		if k == "openai" {
			cs.editStructIndex = i
			break
		}
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	setEntryFieldIndex(t, cs, "default_model")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	cs.editInput.SetValue("save-test-model")
	m = sendConfigKey(m, "enter")

	m = sendConfigKey(m, "esc")
	m = sendConfigKey(m, "esc")

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated2, saveCmd := m.Update(sMsg)
	m = updated2.(Model)
	cs = m.configScreen
	if cs.saveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving", cs.saveStatus)
	}

	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	resultMsg := saveCmd()

	updated3, _ := m.Update(resultMsg)
	m = updated3.(Model)
	cs = m.configScreen

	if cs.dirty {
		t.Fatal("dirty should be false after save")
	}
	if cs.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved", cs.saveStatus)
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
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	cs := m.configScreen

	cs.cfg.DefaultModel = "global-model"

	if pm, ok := cs.cfg.ProviderModels["openai"]; ok {
		pm.DefaultModel = "provider-override"
		cs.cfg.ProviderModels["openai"] = pm
	}
	cs.dirty = true

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
