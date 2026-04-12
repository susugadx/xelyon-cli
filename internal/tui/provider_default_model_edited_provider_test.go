package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

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

func TestDefaultModelSync_UsesEditedProviderForEditAndReset(t *testing.T) {
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
