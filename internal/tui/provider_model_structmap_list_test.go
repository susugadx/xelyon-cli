package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func TestAddStructMapKey_ProviderModelsAllowsAnthropicAndClaudeSiblings(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
	})

	cs := configscreen.New(cfg)
	m.configScreen = cs

	if !m.addStructMapKey("provider_models", "claude") {
		t.Fatal("addStructMapKey(provider_models, claude) = false, want true")
	}

	saved := cs.ConfigSnapshot().ProviderModelsForEdit()
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
	keys := cs.Snapshot().EditStructKeys
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
	keys = cs.Snapshot().EditStructKeys

	for _, k := range keys {
		if k == firstKey {
			t.Fatalf("key %q should have been deleted", firstKey)
		}
	}

	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Fatalf("editStructKeys not sorted after delete: %q after %q", keys[i], keys[i-1])
		}
	}
}

func TestConfigScreen_ProviderModels_NilMap_AddEntry_DoesNotPanic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProviderModels = nil
	m := newConfigTestModelWithConfig(cfg)

	enterConfigStructMapEdit(t, &m, "provider_models")

	m = sendConfigKey(m, "a")
	cs := configTestScreen(t, m)
	if !cs.Snapshot().EditStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	setConfigStructInputValue(t, &m, "nil_provider")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	cfgSnapshot := cs.ConfigSnapshot()
	if cfgSnapshot.ProviderModels == nil {
		t.Fatal("ProviderModels should be initialized after add")
	}
	if _, ok := cfgSnapshot.ProviderModels["nil_provider"]; !ok {
		t.Fatal("ProviderModels should contain the added key")
	}
	snapshot := cs.Snapshot()
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after add")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", snapshot.SaveStatus)
	}
}
