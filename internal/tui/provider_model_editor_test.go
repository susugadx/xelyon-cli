package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDefaultModelSyncProvider_UsesCurrentSessionOwnerWhenDefaultProviderOnlyChangesAliasSpelling(t *testing.T) {
	agent := &stubAgent{providerName: "deepseek"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "anthropic"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "deepseek" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "deepseek")
	}
}

func TestDefaultModelSyncProvider_UsesNewDefaultProviderWhenRuntimeChanges(t *testing.T) {
	agent := &stubAgent{providerName: "deepseek"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "deepseek"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "openai" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "openai")
	}
}

func TestSyncEditedProviderDefaultModel_UsesCurrentSessionOwnerWhenDefaultProviderOnlyChangesAliasSpelling(t *testing.T) {
	agent := &stubAgent{providerName: "deepseek"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.DefaultModel = "claude-custom"
	cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{DefaultModel: "deepseek-chat"})

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	m.syncEditedProviderDefaultModel()

	saved := cs.cfg.ProviderModelsForSave()
	if pm, ok := saved["deepseek"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want deepseek override", saved)
	} else if pm.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()[deepseek].DefaultModel = %q, want %q", pm.DefaultModel, "claude-custom")
	}
	if _, ok := saved["anthropic"]; ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic alias to remain untouched", saved)
	}
}

func TestDefaultModelSyncProvider_UsesSessionProviderConfigKeyWhenDefaultProviderDiffers(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "deepseek"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "anthropic" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "anthropic")
	}
}

func TestDefaultModelSyncProvider_UsesSessionProviderConfigKeyWhenDefaultProviderSpellingIsUnchanged(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "claude"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "anthropic" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "anthropic")
	}
}

func TestSyncEditedProviderDefaultModel_PreservesSessionAnthropicAliasWhenDefaultProviderDiffers(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.DefaultModel = "anthropic-new"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-old"},
		"claude":    {DefaultModel: "claude-old"},
	})

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "deepseek"
	m.configScreen = cs

	m.syncEditedProviderDefaultModel()

	saved := cs.cfg.ProviderModelsForSave()
	if pm, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic override", saved)
	} else if pm.DefaultModel != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-new")
	}
	if pm, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if pm.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", pm.DefaultModel, "claude-old")
	}
}

func TestSyncEditedProviderDefaultModel_CreatesAnthropicEntryWhenDefaultProviderSpellingIsUnchanged(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "claude"
	cfg.DefaultModel = "anthropic-new"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"claude": {DefaultModel: "claude-old"},
	})

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	m.syncEditedProviderDefaultModel()

	saved := cs.cfg.ProviderModelsForSave()
	if pm, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic override created", saved)
	} else if pm.DefaultModel != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-new")
	}
	if pm, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if pm.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", pm.DefaultModel, "claude-old")
	}
}

func TestAddStructMapKey_ProviderModelsAllowsAnthropicAndClaudeSiblings(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
	})

	cs := newConfigScreen(cfg)
	m.configScreen = cs

	if !m.addStructMapKey("provider_models", "claude") {
		t.Fatal("addStructMapKey(provider_models, claude) = false, want true")
	}

	saved := cs.cfg.ProviderModelsForEdit()
	if _, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForEdit() = %#v, want anthropic entry preserved", saved)
	}
	if _, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForEdit() = %#v, want claude sibling entry added", saved)
	}
}

func TestConfigScreen_StructMapOrder_ProviderModels(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "provider" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "provider_models" {
			cs.fieldIndex = i
			break
		}
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", cs.editMode)
	}

	keys := cs.editStructKeys
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Fatalf("editStructKeys not sorted: %q comes after %q", keys[i], keys[i-1])
		}
	}

	if len(keys) == 0 {
		t.Fatal("editStructKeys is empty")
	}
	firstKey := keys[0]

	cs.editStructIndex = 0
	m = sendConfigKey(m, "d")
	cs = m.configScreen

	for _, k := range cs.editStructKeys {
		if k == firstKey {
			t.Fatalf("key %q should have been deleted", firstKey)
		}
	}

	for i := 1; i < len(cs.editStructKeys); i++ {
		if cs.editStructKeys[i] < cs.editStructKeys[i-1] {
			t.Fatalf("editStructKeys not sorted after delete: %q after %q", cs.editStructKeys[i], cs.editStructKeys[i-1])
		}
	}
}

func TestConfigScreen_ProviderModels_EntryEdit(t *testing.T) {
	m := enterStructMapEdit(t, "provider_models")
	cs := m.configScreen

	if len(cs.editStructKeys) == 0 {
		t.Fatal("no provider_models keys")
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatal("editEntryActive should be true")
	}
	if len(cs.editEntryFields) == 0 {
		t.Fatal("editEntryFields should not be empty")
	}

	found := false
	for i, ef := range cs.editEntryFields {
		if ef.Name == "default_model" {
			cs.editEntryIndex = i
			found = true
			break
		}
	}
	if !found {
		t.Fatal("default_model field not found in entry")
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "" {
		t.Fatalf("editEntryFieldEdit = %q after esc, want empty", cs.editEntryFieldEdit)
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editEntryActive {
		t.Fatal("editEntryActive should be false after esc")
	}
}

func TestConfigScreen_ProviderModels_NilMap_AddEntry_DoesNotPanic(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen
	cs.cfg.ProviderModels = nil
	cs.refreshCategories()

	setConfigFieldSelection(t, cs, "provider", "provider_models")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", cs.editMode)
	}

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	if !cs.editStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	cs.editStructInput.SetValue("nil_provider")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.cfg.ProviderModels == nil {
		t.Fatal("ProviderModels should be initialized after add")
	}
	if _, ok := cs.cfg.ProviderModels["nil_provider"]; !ok {
		t.Fatal("ProviderModels should contain the added key")
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after add")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", cs.saveStatus)
	}
}

func TestConfigScreen_ProviderModels_ValueChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")
	cs := m.configScreen

	setEntryFieldIndex(t, cs, "default_model")

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	cs.editInput.SetValue("gpt-99-turbo")

	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	pm, ok := cs.cfg.ProviderModels["openai"]
	if !ok {
		t.Fatal("openai not found in ProviderModels")
	}
	if pm.DefaultModel != "gpt-99-turbo" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"gpt-99-turbo\"", pm.DefaultModel)
	}

	if !cs.dirty {
		t.Fatal("dirty should be true after value change")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}

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

func TestConfigScreen_DefaultProviderThenDefaultModelSyncsEditedProvider(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(config.DefaultConfig())

	cs := m.configScreen
	cs.cfg.DefaultProvider = "deepseek"
	cs.cfg.DefaultModel = "deepseek-chat"
	cs.refreshCategories()

	m = selectConfigOption(t, m, "provider", "default_provider", "openai")
	cs = m.configScreen
	if cs.cfg.DefaultProvider != "openai" {
		t.Fatalf("DefaultProvider = %q, want \"openai\"", cs.cfg.DefaultProvider)
	}

	setConfigFieldSelection(t, cs, "provider", "default_model")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput", cs.editMode)
	}

	cs.editInput.SetValue("gpt-5.4")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.cfg.DefaultModel != "gpt-5.4" {
		t.Fatalf("DefaultModel = %q, want \"gpt-5.4\"", cs.cfg.DefaultModel)
	}

	openaiPM := cs.cfg.ProviderModels["openai"]
	if openaiPM.DefaultModel != "gpt-5.4" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"gpt-5.4\"", openaiPM.DefaultModel)
	}

	deepseekPM := cs.cfg.ProviderModels["deepseek"]
	if deepseekPM.DefaultModel == "gpt-5.4" {
		t.Fatalf("ProviderModels[deepseek].DefaultModel = %q, should not be updated from edited provider path", deepseekPM.DefaultModel)
	}

	m = saveConfigAndWait(t, m)

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.ProviderModels["openai"].DefaultModel != "gpt-5.4" {
		t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want \"gpt-5.4\"", saved.ProviderModels["openai"].DefaultModel)
	}
	if saved.ProviderModels["deepseek"].DefaultModel == "gpt-5.4" {
		t.Fatalf("saved ProviderModels[deepseek].DefaultModel = %q, should not match edited openai model", saved.ProviderModels["deepseek"].DefaultModel)
	}
}

func TestDefaultModelSync_UsesIntendedProvider(t *testing.T) {
	t.Run("edited default_provider wins for edit and reset", func(t *testing.T) {
		run := func(t *testing.T, reset bool) {
			t.Helper()

			agent := &stubAgent{}
			m := newModelWithViewport(agent)
			m.screen = screenConfig
			m.configScreen = newConfigScreen(config.DefaultConfig())

			cs := m.configScreen
			cs.cfg.DefaultProvider = "deepseek"
			cs.cfg.DefaultModel = "custom-global-model"

			openaiPM := cs.cfg.ProviderModels["openai"]
			openaiPM.DefaultModel = "stale-openai-model"
			cs.cfg.ProviderModels["openai"] = openaiPM

			deepseekPM := cs.cfg.ProviderModels["deepseek"]
			deepseekPM.DefaultModel = "stale-deepseek-model"
			cs.cfg.ProviderModels["deepseek"] = deepseekPM
			cs.refreshCategories()

			m = selectConfigOption(t, m, "provider", "default_provider", "openai")
			cs = m.configScreen
			setConfigFieldSelection(t, cs, "provider", "default_model")

			wantModel := "gpt-5.4"
			if reset {
				m = sendConfigKey(m, "r")
				wantModel = config.DefaultConfig().DefaultModel
			} else {
				m = sendConfigKey(m, "enter")
				cs = m.configScreen
				if cs.editMode != editInput {
					t.Fatalf("editMode = %d, want editInput", cs.editMode)
				}
				cs.editInput.SetValue(wantModel)
				m = sendConfigKey(m, "enter")
			}

			cs = m.configScreen
			if cs.cfg.DefaultModel != wantModel {
				t.Fatalf("DefaultModel = %q, want %q", cs.cfg.DefaultModel, wantModel)
			}
			if cs.cfg.ProviderModels["openai"].DefaultModel != wantModel {
				t.Fatalf("ProviderModels[openai].DefaultModel = %q, want %q", cs.cfg.ProviderModels["openai"].DefaultModel, wantModel)
			}
			if cs.cfg.ProviderModels["deepseek"].DefaultModel == wantModel {
				t.Fatalf("ProviderModels[deepseek].DefaultModel = %q, should not be updated", cs.cfg.ProviderModels["deepseek"].DefaultModel)
			}

			m = saveConfigAndWait(t, m)

			agent.mu.RLock()
			saved := agent.lastSavedConfig
			agent.mu.RUnlock()
			if saved == nil {
				t.Fatal("lastSavedConfig is nil")
			}
			if saved.ProviderModels["openai"].DefaultModel != wantModel {
				t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want %q", saved.ProviderModels["openai"].DefaultModel, wantModel)
			}
			if saved.ProviderModels["deepseek"].DefaultModel == wantModel {
				t.Fatalf("saved ProviderModels[deepseek].DefaultModel = %q, should not be updated", saved.ProviderModels["deepseek"].DefaultModel)
			}
		}

		t.Run("edit", func(t *testing.T) { run(t, false) })
		t.Run("reset", func(t *testing.T) { run(t, true) })
	})

	t.Run("runtime provider wins when default_provider unchanged", func(t *testing.T) {
		run := func(t *testing.T, reset bool) {
			t.Helper()

			agent := &stubAgent{providerName: "openai"}
			m := newModelWithViewport(agent)
			m.screen = screenConfig
			m.configScreen = newConfigScreen(config.DefaultConfig())

			cs := m.configScreen
			cs.cfg.DefaultProvider = "deepseek"
			cs.cfg.DefaultModel = "custom-global-model"

			openaiPM := cs.cfg.ProviderModels["openai"]
			openaiPM.DefaultModel = "stale-openai-model"
			cs.cfg.ProviderModels["openai"] = openaiPM

			deepseekPM := cs.cfg.ProviderModels["deepseek"]
			deepseekPM.DefaultModel = "stale-deepseek-model"
			cs.cfg.ProviderModels["deepseek"] = deepseekPM
			cs.refreshCategories()

			setConfigFieldSelection(t, cs, "provider", "default_model")

			wantModel := "gpt-5.4"
			if reset {
				m = sendConfigKey(m, "r")
				wantModel = config.DefaultConfig().DefaultModel
			} else {
				m = sendConfigKey(m, "enter")
				cs = m.configScreen
				if cs.editMode != editInput {
					t.Fatalf("editMode = %d, want editInput", cs.editMode)
				}
				cs.editInput.SetValue(wantModel)
				m = sendConfigKey(m, "enter")
			}

			cs = m.configScreen
			if cs.cfg.DefaultModel != wantModel {
				t.Fatalf("DefaultModel = %q, want %q", cs.cfg.DefaultModel, wantModel)
			}
			if cs.cfg.ProviderModels["openai"].DefaultModel != wantModel {
				t.Fatalf("ProviderModels[openai].DefaultModel = %q, want %q", cs.cfg.ProviderModels["openai"].DefaultModel, wantModel)
			}
			if cs.cfg.ProviderModels["deepseek"].DefaultModel == wantModel {
				t.Fatalf("ProviderModels[deepseek].DefaultModel = %q, should not be updated", cs.cfg.ProviderModels["deepseek"].DefaultModel)
			}

			m = saveConfigAndWait(t, m)

			agent.mu.RLock()
			saved := agent.lastSavedConfig
			agent.mu.RUnlock()
			if saved == nil {
				t.Fatal("lastSavedConfig is nil")
			}
			if saved.ProviderModels["openai"].DefaultModel != wantModel {
				t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want %q", saved.ProviderModels["openai"].DefaultModel, wantModel)
			}
			if saved.ProviderModels["deepseek"].DefaultModel == wantModel {
				t.Fatalf("saved ProviderModels[deepseek].DefaultModel = %q, should not be updated", saved.ProviderModels["deepseek"].DefaultModel)
			}
		}

		t.Run("edit", func(t *testing.T) { run(t, false) })
		t.Run("reset", func(t *testing.T) { run(t, true) })
	})
}

func TestConfigScreen_DefaultProviderThenResetDefaultModelSyncsEditedProvider(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(config.DefaultConfig())

	cs := m.configScreen
	cs.cfg.DefaultProvider = "deepseek"
	cs.cfg.DefaultModel = "custom-global-model"
	deepseekPM := cs.cfg.ProviderModels["deepseek"]
	deepseekPM.DefaultModel = "stale-deepseek-model"
	cs.cfg.ProviderModels["deepseek"] = deepseekPM
	openaiPM := cs.cfg.ProviderModels["openai"]
	openaiPM.DefaultModel = "stale-openai-model"
	cs.cfg.ProviderModels["openai"] = openaiPM
	cs.refreshCategories()

	m = selectConfigOption(t, m, "provider", "default_provider", "openai")
	cs = m.configScreen
	setConfigFieldSelection(t, cs, "provider", "default_model")

	m = sendConfigKey(m, "r")
	cs = m.configScreen

	defaultCfg := config.DefaultConfig()
	if cs.cfg.DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("DefaultModel after reset = %q, want %q", cs.cfg.DefaultModel, defaultCfg.DefaultModel)
	}
	openaiPM = cs.cfg.ProviderModels["openai"]
	if openaiPM.DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("ProviderModels[openai].DefaultModel after reset = %q, want %q", openaiPM.DefaultModel, defaultCfg.DefaultModel)
	}
	deepseekPM = cs.cfg.ProviderModels["deepseek"]
	if deepseekPM.DefaultModel == defaultCfg.DefaultModel {
		t.Fatalf("ProviderModels[deepseek].DefaultModel after reset = %q, should not be updated from edited provider path", deepseekPM.DefaultModel)
	}

	m = saveConfigAndWait(t, m)

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("saved.DefaultModel = %q, want %q", saved.DefaultModel, defaultCfg.DefaultModel)
	}
	if saved.ProviderModels["openai"].DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want %q", saved.ProviderModels["openai"].DefaultModel, defaultCfg.DefaultModel)
	}
	if saved.ProviderModels["deepseek"].DefaultModel == defaultCfg.DefaultModel {
		t.Fatalf("saved ProviderModels[deepseek].DefaultModel = %q, should not match edited openai reset target", saved.ProviderModels["deepseek"].DefaultModel)
	}
}

func TestConfigScreen_ActiveSessionProviderThenDefaultModelSyncsRuntimeProvider(t *testing.T) {
	agent := &stubAgent{providerName: "openai"}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(config.DefaultConfig())

	cs := m.configScreen
	cs.cfg.DefaultProvider = "deepseek"
	cs.cfg.DefaultModel = "deepseek-chat"
	openaiPM := cs.cfg.ProviderModels["openai"]
	openaiPM.DefaultModel = "stale-openai-model"
	cs.cfg.ProviderModels["openai"] = openaiPM
	deepseekPM := cs.cfg.ProviderModels["deepseek"]
	deepseekPM.DefaultModel = "stale-deepseek-model"
	cs.cfg.ProviderModels["deepseek"] = deepseekPM
	cs.refreshCategories()

	setConfigFieldSelection(t, cs, "provider", "default_model")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput", cs.editMode)
	}

	cs.editInput.SetValue("gpt-5.4")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.cfg.DefaultModel != "gpt-5.4" {
		t.Fatalf("DefaultModel = %q, want \"gpt-5.4\"", cs.cfg.DefaultModel)
	}
	if cs.cfg.ProviderModels["openai"].DefaultModel != "gpt-5.4" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"gpt-5.4\"", cs.cfg.ProviderModels["openai"].DefaultModel)
	}
	if cs.cfg.ProviderModels["deepseek"].DefaultModel == "gpt-5.4" {
		t.Fatalf("ProviderModels[deepseek].DefaultModel = %q, should not be updated from runtime provider path", cs.cfg.ProviderModels["deepseek"].DefaultModel)
	}

	m = saveConfigAndWait(t, m)

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.ProviderModels["openai"].DefaultModel != "gpt-5.4" {
		t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want \"gpt-5.4\"", saved.ProviderModels["openai"].DefaultModel)
	}
	if saved.ProviderModels["deepseek"].DefaultModel == "gpt-5.4" {
		t.Fatalf("saved ProviderModels[deepseek].DefaultModel = %q, should not match runtime provider sync target", saved.ProviderModels["deepseek"].DefaultModel)
	}
}

func TestConfigScreen_ActiveSessionProviderThenResetDefaultModelSyncsRuntimeProvider(t *testing.T) {
	agent := &stubAgent{providerName: "openai"}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(config.DefaultConfig())

	cs := m.configScreen
	cs.cfg.DefaultProvider = "deepseek"
	cs.cfg.DefaultModel = "custom-global-model"
	openaiPM := cs.cfg.ProviderModels["openai"]
	openaiPM.DefaultModel = "stale-openai-model"
	cs.cfg.ProviderModels["openai"] = openaiPM
	deepseekPM := cs.cfg.ProviderModels["deepseek"]
	deepseekPM.DefaultModel = "stale-deepseek-model"
	cs.cfg.ProviderModels["deepseek"] = deepseekPM
	cs.refreshCategories()

	setConfigFieldSelection(t, cs, "provider", "default_model")
	m = sendConfigKey(m, "r")
	cs = m.configScreen

	defaultCfg := config.DefaultConfig()
	if cs.cfg.DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("DefaultModel after reset = %q, want %q", cs.cfg.DefaultModel, defaultCfg.DefaultModel)
	}
	if cs.cfg.ProviderModels["openai"].DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("ProviderModels[openai].DefaultModel after reset = %q, want %q", cs.cfg.ProviderModels["openai"].DefaultModel, defaultCfg.DefaultModel)
	}
	if cs.cfg.ProviderModels["deepseek"].DefaultModel == defaultCfg.DefaultModel {
		t.Fatalf("ProviderModels[deepseek].DefaultModel after reset = %q, should not be updated from runtime provider path", cs.cfg.ProviderModels["deepseek"].DefaultModel)
	}

	m = saveConfigAndWait(t, m)

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.ProviderModels["openai"].DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("saved ProviderModels[openai].DefaultModel = %q, want %q", saved.ProviderModels["openai"].DefaultModel, defaultCfg.DefaultModel)
	}
	if saved.ProviderModels["deepseek"].DefaultModel == defaultCfg.DefaultModel {
		t.Fatalf("saved ProviderModels[deepseek].DefaultModel = %q, should not match runtime provider reset target", saved.ProviderModels["deepseek"].DefaultModel)
	}
}

func TestConfigScreen_ProviderModelDirectEditStillDoesNotGetOverwritten(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	cs.cfg.DefaultProvider = "openai"
	cs.cfg.DefaultModel = "global-model"
	if pm, ok := cs.cfg.ProviderModels["openai"]; ok {
		pm.DefaultModel = "openai-specific"
		cs.cfg.ProviderModels["openai"] = pm
	}
	cs.dirty = true

	m = saveConfigAndWait(t, m)
	cs = m.configScreen

	pm := cs.cfg.ProviderModels["openai"]
	if pm.DefaultModel != "openai-specific" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"openai-specific\"", pm.DefaultModel)
	}
}
