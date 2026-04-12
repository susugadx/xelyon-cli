package tui

import "testing"

func TestConfigScreen_StructMapDelete_AfterEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	if len(cs.editStructKeys) < 2 {
		t.Skip("not enough keys to test delete after edit")
	}

	firstKey := cs.editStructKeys[0]
	m = sendConfigKey(m, "enter")
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editEntryActive {
		t.Fatal("should be back to key list")
	}

	for i := 1; i < len(cs.editStructKeys); i++ {
		if cs.editStructKeys[i] < cs.editStructKeys[i-1] {
			t.Fatalf("keys not sorted after edit+back: %q after %q", cs.editStructKeys[i], cs.editStructKeys[i-1])
		}
	}

	cs.editStructIndex = 0
	m = sendConfigKey(m, "d")
	cs = m.configScreen

	for _, k := range cs.editStructKeys {
		if k == firstKey {
			t.Fatalf("key %q should have been deleted", firstKey)
		}
	}
}

func TestConfigScreen_LSPServers_NilMap_AddEntry_DoesNotPanic(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen
	cs.cfg.LSP.Servers = nil
	cs.refreshCategories()

	setConfigFieldSelection(t, cs, "lsp", "lsp.servers")
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

	cs.editStructInput.SetValue("nil_server")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.cfg.LSP.Servers == nil {
		t.Fatal("LSP.Servers should be initialized after add")
	}
	if _, ok := cs.cfg.LSP.Servers["nil_server"]; !ok {
		t.Fatal("LSP.Servers should contain the added key")
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after add")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", cs.saveStatus)
	}
}

func TestConfigScreen_StructMap_NonNil_AddEntry_BehaviorUnchanged(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	initialLen := len(cs.editStructKeys)
	initialDirty := cs.dirty

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editStructInput.SetValue("behavior_test_lsp")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if len(cs.editStructKeys) != initialLen+1 {
		t.Fatalf("keys count = %d, want %d", len(cs.editStructKeys), initialLen+1)
	}
	if _, ok := cs.cfg.LSP.Servers["behavior_test_lsp"]; !ok {
		t.Fatal("LSP.Servers should contain the added key")
	}
	if cs.dirty == initialDirty {
		t.Fatalf("dirty = %v, want changed from initial state", cs.dirty)
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", cs.saveStatus)
	}

	addedLen := len(cs.editStructKeys)
	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editStructInput.SetValue("behavior_test_lsp")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if len(cs.editStructKeys) != addedLen {
		t.Fatalf("duplicate add changed key count: got %d, want %d", len(cs.editStructKeys), addedLen)
	}
}

func TestConfigScreen_StructMap_DuplicateKey_NoUIAppend(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	if len(cs.editStructKeys) == 0 {
		t.Fatal("no keys")
	}
	existingKey := cs.editStructKeys[0]
	initialLen := len(cs.editStructKeys)
	initialDirty := cs.dirty

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editStructInput.SetValue(existingKey)
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if len(cs.editStructKeys) != initialLen {
		t.Fatalf("editStructKeys length = %d after duplicate add, want %d", len(cs.editStructKeys), initialLen)
	}
	if cs.dirty != initialDirty {
		t.Fatalf("dirty = %v after duplicate add, want %v", cs.dirty, initialDirty)
	}

	seen := make(map[string]bool)
	for _, k := range cs.editStructKeys {
		if seen[k] {
			t.Fatalf("duplicate key %q in editStructKeys", k)
		}
		seen[k] = true
	}
}
