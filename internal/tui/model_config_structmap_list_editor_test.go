package tui

import "testing"

func TestConfigScreen_StructMapEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "lsp.servers" {
			cs.fieldIndex = i
			break
		}
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap(%d)", cs.editMode, editStructMap)
	}
	if len(cs.editStructKeys) == 0 {
		t.Fatal("editStructKeys should not be empty for lsp.servers")
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", cs.editMode)
	}
}

func TestConfigScreen_LSPServersEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	found := false
	for i, f := range fields {
		if f.Path == "lsp.servers" {
			cs.fieldIndex = i
			found = true
			break
		}
	}
	if !found {
		t.Fatal("lsp.servers not found in lsp category fields")
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap(%d)", cs.editMode, editStructMap)
	}
	if len(cs.editStructKeys) == 0 {
		t.Fatal("editStructKeys should not be empty for lsp.servers")
	}
}

func TestConfigScreen_StructMapOrder_LSPServers(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "lsp.servers" {
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
	if len(keys) < 5 {
		t.Fatalf("expected many LSP server keys, got %d", len(keys))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Fatalf("lsp.servers keys not sorted: %q comes after %q", keys[i], keys[i-1])
		}
	}

	target := keys[2]
	cs.editStructIndex = 2
	m = sendConfigKey(m, "d")
	cs = m.configScreen

	for _, k := range cs.editStructKeys {
		if k == target {
			t.Fatalf("key %q at index 2 should have been deleted", target)
		}
	}
	for i := 1; i < len(cs.editStructKeys); i++ {
		if cs.editStructKeys[i] < cs.editStructKeys[i-1] {
			t.Fatalf("keys not sorted after delete")
		}
	}
}
