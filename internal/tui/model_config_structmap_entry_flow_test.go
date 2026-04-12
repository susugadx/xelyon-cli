package tui

import (
	"strings"
	"testing"
)

func TestConfigScreen_LSPServers_EntryEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	goIdx := -1
	for i, k := range cs.editStructKeys {
		if k == "go" {
			goIdx = i
			break
		}
	}
	if goIdx < 0 {
		t.Fatal("go key not found in lsp.servers")
	}
	cs.editStructIndex = goIdx

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatal("editEntryActive should be true")
	}
	if cs.editEntryKey != "go" {
		t.Fatalf("editEntryKey = %q, want \"go\"", cs.editEntryKey)
	}

	argsIdx := -1
	for i, ef := range cs.editEntryFields {
		if ef.Name == "args" {
			argsIdx = i
			if ef.Type != "[]string" {
				t.Fatalf("args type = %q, want []string", ef.Type)
			}
			break
		}
	}
	if argsIdx < 0 {
		t.Fatal("args field not found")
	}

	cs.editEntryIndex = argsIdx
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "slice" {
		t.Fatalf("editEntryFieldEdit = %q, want \"slice\"", cs.editEntryFieldEdit)
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "" {
		t.Fatal("editEntryFieldEdit should be empty after esc from slice")
	}
}

func TestConfigScreen_StructMapAdd_ThenEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	initialLen := len(cs.editStructKeys)

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	if !cs.editStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	cs.editStructInput.SetValue("testlang")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if len(cs.editStructKeys) != initialLen+1 {
		t.Fatalf("keys count = %d, want %d", len(cs.editStructKeys), initialLen+1)
	}
	if cs.editStructKeys[cs.editStructIndex] != "testlang" {
		t.Fatalf("cursor on %q, want \"testlang\"", cs.editStructKeys[cs.editStructIndex])
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatal("editEntryActive should be true for new key")
	}
	if cs.editEntryKey != "testlang" {
		t.Fatalf("editEntryKey = %q, want \"testlang\"", cs.editEntryKey)
	}
}

func TestConfigScreen_StructMapEntryEdit_HintTransition(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")

	m.width = 120
	m.height = 30
	view := m.View()
	if !strings.Contains(view, "Enter:edit entry") {
		t.Fatal("key list hint should contain 'Enter:edit entry'")
	}

	m = sendConfigKey(m, "enter")
	m.width = 120
	m.height = 30
	view = m.View()
	if !strings.Contains(view, "Esc:back") {
		t.Fatal("entry edit hint should contain 'Esc:back'")
	}

	m = sendConfigKey(m, "esc")
	if m.configScreen.editEntryActive {
		t.Fatal("should be back to key list")
	}
}
