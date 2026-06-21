package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func runtimeProviderSyncConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.DefaultModel = "custom-global-model"
	openaiPM := cfg.ProviderModels["openai"]
	openaiPM.DefaultModel = "stale-openai-model"
	cfg.ProviderModels["openai"] = openaiPM
	deepseekPM := cfg.ProviderModels["deepseek"]
	deepseekPM.DefaultModel = "stale-deepseek-model"
	cfg.ProviderModels["deepseek"] = deepseekPM
	return cfg
}

func TestConfigScreen_ActiveSessionProviderThenDefaultModelSyncsRuntimeProvider(t *testing.T) {
	agent := &stubAgent{providerName: "openai"}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	cfg := runtimeProviderSyncConfig()
	cfg.DefaultModel = "deepseek-chat"
	m.configScreen = configscreen.New(cfg)

	selectConfigField(t, &m, "provider", "default_model")
	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if got := cs.Snapshot().EditMode; got != editInput {
		t.Fatalf("editMode = %d, want editInput", got)
	}

	setConfigInputValue(t, &m, "gpt-5.4")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	cfgSnapshot := cs.ConfigSnapshot()

	if cfgSnapshot.DefaultModel != "gpt-5.4" {
		t.Fatalf("DefaultModel = %q, want \"gpt-5.4\"", cfgSnapshot.DefaultModel)
	}
	if cfgSnapshot.ProviderModels["openai"].DefaultModel != "gpt-5.4" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"gpt-5.4\"", cfgSnapshot.ProviderModels["openai"].DefaultModel)
	}
	if cfgSnapshot.ProviderModels["deepseek"].DefaultModel == "gpt-5.4" {
		t.Fatalf("ProviderModels[deepseek].DefaultModel = %q, should not be updated from runtime provider path", cfgSnapshot.ProviderModels["deepseek"].DefaultModel)
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

func TestDefaultModelSync_UsesRuntimeProviderWhenDefaultProviderUnchanged(t *testing.T) {
	run := func(t *testing.T, reset bool) {
		t.Helper()

		agent := &stubAgent{providerName: "openai"}
		m := newModelWithViewport(agent)
		m.screen = screenConfig
		m.configScreen = configscreen.New(runtimeProviderSyncConfig())

		selectConfigField(t, &m, "provider", "default_model")

		wantModel := "gpt-5.4"
		if reset {
			m = sendConfigKey(m, "r")
			wantModel = config.DefaultConfig().DefaultModel
		} else {
			m = sendConfigKey(m, "enter")
			cs := configTestScreen(t, m)
			if got := cs.Snapshot().EditMode; got != editInput {
				t.Fatalf("editMode = %d, want editInput", got)
			}
			setConfigInputValue(t, &m, wantModel)
			m = sendConfigKey(m, "enter")
		}

		cs := configTestScreen(t, m)
		cfgSnapshot := cs.ConfigSnapshot()
		if cfgSnapshot.DefaultModel != wantModel {
			t.Fatalf("DefaultModel = %q, want %q", cfgSnapshot.DefaultModel, wantModel)
		}
		if cfgSnapshot.ProviderModels["openai"].DefaultModel != wantModel {
			t.Fatalf("ProviderModels[openai].DefaultModel = %q, want %q", cfgSnapshot.ProviderModels["openai"].DefaultModel, wantModel)
		}
		if cfgSnapshot.ProviderModels["deepseek"].DefaultModel == wantModel {
			t.Fatalf("ProviderModels[deepseek].DefaultModel = %q, should not be updated", cfgSnapshot.ProviderModels["deepseek"].DefaultModel)
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

func TestConfigScreen_ActiveSessionProviderThenResetDefaultModelSyncsRuntimeProvider(t *testing.T) {
	agent := &stubAgent{providerName: "openai"}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(runtimeProviderSyncConfig())

	selectConfigField(t, &m, "provider", "default_model")
	m = sendConfigKey(m, "r")
	cs := configTestScreen(t, m)
	cfgSnapshot := cs.ConfigSnapshot()

	defaultCfg := config.DefaultConfig()
	if cfgSnapshot.DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("DefaultModel after reset = %q, want %q", cfgSnapshot.DefaultModel, defaultCfg.DefaultModel)
	}
	if cfgSnapshot.ProviderModels["openai"].DefaultModel != defaultCfg.DefaultModel {
		t.Fatalf("ProviderModels[openai].DefaultModel after reset = %q, want %q", cfgSnapshot.ProviderModels["openai"].DefaultModel, defaultCfg.DefaultModel)
	}
	if cfgSnapshot.ProviderModels["deepseek"].DefaultModel == defaultCfg.DefaultModel {
		t.Fatalf("ProviderModels[deepseek].DefaultModel after reset = %q, should not be updated from runtime provider path", cfgSnapshot.ProviderModels["deepseek"].DefaultModel)
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
