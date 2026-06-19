package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_StructMapDelete_AfterEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen
	snapshot := cs.Snapshot()

	if len(snapshot.EditStructKeys) < 2 {
		t.Skip("not enough keys to test delete after edit")
	}

	firstKey := snapshot.EditStructKeys[0]
	m = sendConfigKey(m, "enter")
	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	snapshot = cs.Snapshot()
	if snapshot.EditEntryActive {
		t.Fatal("should be back to key list")
	}

	for i := 1; i < len(snapshot.EditStructKeys); i++ {
		if snapshot.EditStructKeys[i] < snapshot.EditStructKeys[i-1] {
			t.Fatalf("keys not sorted after edit+back: %q after %q", snapshot.EditStructKeys[i], snapshot.EditStructKeys[i-1])
		}
	}

	selectConfigStructMapKeyIndex(t, &m, 0)
	m = sendConfigKey(m, "d")
	cs = configTestScreen(t, m)

	for _, k := range cs.Snapshot().EditStructKeys {
		if k == firstKey {
			t.Fatalf("key %q should have been deleted", firstKey)
		}
	}
}

func TestConfigScreen_LSPServers_NilMap_AddEntry_DoesNotPanic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LSP.Servers = nil
	m := newConfigTestModelWithConfig(cfg)

	enterConfigStructMapEdit(t, &m, "lsp.servers")

	m = sendConfigKey(m, "a")
	cs := configTestScreen(t, m)
	if !cs.Snapshot().EditStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	setConfigStructInputValue(t, &m, "nil_server")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	cfgSnapshot := cs.ConfigSnapshot()
	if cfgSnapshot.LSP.Servers == nil {
		t.Fatal("LSP.Servers should be initialized after add")
	}
	if _, ok := cfgSnapshot.LSP.Servers["nil_server"]; !ok {
		t.Fatal("LSP.Servers should contain the added key")
	}
	snapshot := cs.Snapshot()
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after add")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", snapshot.SaveStatus)
	}
}

func TestConfigScreen_StructMap_NonNil_AddEntry_BehaviorUnchanged(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	snapshot := cs.Snapshot()
	initialLen := len(snapshot.EditStructKeys)
	initialDirty := snapshot.Dirty

	m = sendConfigKey(m, "a")
	setConfigStructInputValue(t, &m, "behavior_test_lsp")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	snapshot = cs.Snapshot()

	if len(snapshot.EditStructKeys) != initialLen+1 {
		t.Fatalf("keys count = %d, want %d", len(snapshot.EditStructKeys), initialLen+1)
	}
	if _, ok := cs.ConfigSnapshot().LSP.Servers["behavior_test_lsp"]; !ok {
		t.Fatal("LSP.Servers should contain the added key")
	}
	if snapshot.Dirty == initialDirty {
		t.Fatalf("dirty = %v, want changed from initial state", snapshot.Dirty)
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", snapshot.SaveStatus)
	}

	addedLen := len(snapshot.EditStructKeys)
	m = sendConfigKey(m, "a")
	setConfigStructInputValue(t, &m, "behavior_test_lsp")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if got := len(cs.Snapshot().EditStructKeys); got != addedLen {
		t.Fatalf("duplicate add changed key count: got %d, want %d", got, addedLen)
	}
}

func TestConfigScreen_StructMap_DuplicateKey_NoUIAppend(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen
	snapshot := cs.Snapshot()

	if len(snapshot.EditStructKeys) == 0 {
		t.Fatal("no keys")
	}
	existingKey := snapshot.EditStructKeys[0]
	initialLen := len(snapshot.EditStructKeys)
	initialDirty := snapshot.Dirty

	m = sendConfigKey(m, "a")
	setConfigStructInputValue(t, &m, existingKey)
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	snapshot = cs.Snapshot()

	if len(snapshot.EditStructKeys) != initialLen {
		t.Fatalf("editStructKeys length = %d after duplicate add, want %d", len(snapshot.EditStructKeys), initialLen)
	}
	if snapshot.Dirty != initialDirty {
		t.Fatalf("dirty = %v after duplicate add, want %v", snapshot.Dirty, initialDirty)
	}

	seen := make(map[string]bool)
	for _, k := range snapshot.EditStructKeys {
		if seen[k] {
			t.Fatalf("duplicate key %q in editStructKeys", k)
		}
		seen[k] = true
	}
}
