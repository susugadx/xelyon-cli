package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

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
	enterConfigStructMapEdit(t, &m, "provider_models")

	cs := configTestScreen(t, m)
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

	selectConfigStructMapKeyIndex(t, &m, 0)
	m = sendConfigKey(m, "d")
	cs = configTestScreen(t, m)

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

func TestConfigScreen_ProviderModels_NilMap_AddEntry_DoesNotPanic(t *testing.T) {
	m := newConfigTestModel()
	cs := configTestScreen(t, m)
	cs.cfg.ProviderModels = nil
	cs.refreshCategories()

	enterConfigStructMapEdit(t, &m, "provider_models")

	m = sendConfigKey(m, "a")
	cs = configTestScreen(t, m)
	if !cs.editStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	setConfigStructInputValue(t, &m, "nil_provider")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

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
