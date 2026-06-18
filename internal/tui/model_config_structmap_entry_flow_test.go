package tui

import (
	"strings"
	"testing"
)

func TestConfigScreen_LSPServers_EntryEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	selectConfigStructMapKey(t, &m, "go")

	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if !cs.editEntryActive {
		t.Fatal("editEntryActive should be true")
	}
	if cs.editEntryKey != "go" {
		t.Fatalf("editEntryKey = %q, want \"go\"", cs.editEntryKey)
	}

	argsIdx := selectConfigEntryField(t, &m, "args")
	if got := cs.editEntryFields[argsIdx].Type; got != "[]string" {
		t.Fatalf("args type = %q, want []string", got)
	}

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if cs.editEntryFieldEdit != "slice" {
		t.Fatalf("editEntryFieldEdit = %q, want \"slice\"", cs.editEntryFieldEdit)
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if cs.editEntryFieldEdit != "" {
		t.Fatal("editEntryFieldEdit should be empty after esc from slice")
	}
}

func TestConfigScreen_StructMapAdd_ThenEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := configTestScreen(t, m)

	initialLen := len(cs.editStructKeys)

	m = sendConfigKey(m, "a")
	cs = configTestScreen(t, m)
	if !cs.editStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	setConfigStructInputValue(t, &m, "testlang")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if len(cs.editStructKeys) != initialLen+1 {
		t.Fatalf("keys count = %d, want %d", len(cs.editStructKeys), initialLen+1)
	}
	if cs.editStructKeys[cs.editStructIndex] != "testlang" {
		t.Fatalf("cursor on %q, want \"testlang\"", cs.editStructKeys[cs.editStructIndex])
	}

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
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
