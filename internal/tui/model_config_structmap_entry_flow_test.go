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
	snapshot := cs.Snapshot()
	if !snapshot.EditEntryActive {
		t.Fatal("editEntryActive should be true")
	}
	if snapshot.EditEntryKey != "go" {
		t.Fatalf("editEntryKey = %q, want \"go\"", snapshot.EditEntryKey)
	}

	argsIdx := selectConfigEntryField(t, &m, "args")
	snapshot = configTestScreen(t, m).Snapshot()
	if got := snapshot.EditEntryFields[argsIdx].Type; got != "[]string" {
		t.Fatalf("args type = %q, want []string", got)
	}

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "slice" {
		t.Fatalf("editEntryFieldEdit = %q, want \"slice\"", got)
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if cs.Snapshot().EditEntryFieldEdit != "" {
		t.Fatal("editEntryFieldEdit should be empty after esc from slice")
	}
}

func TestConfigScreen_StructMapAdd_ThenEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := configTestScreen(t, m)

	initialLen := len(cs.Snapshot().EditStructKeys)

	m = sendConfigKey(m, "a")
	cs = configTestScreen(t, m)
	if !cs.Snapshot().EditStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	setConfigStructInputValue(t, &m, "testlang")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	snapshot := cs.Snapshot()
	if len(snapshot.EditStructKeys) != initialLen+1 {
		t.Fatalf("keys count = %d, want %d", len(snapshot.EditStructKeys), initialLen+1)
	}
	if snapshot.EditStructKeys[snapshot.EditStructIndex] != "testlang" {
		t.Fatalf("cursor on %q, want \"testlang\"", snapshot.EditStructKeys[snapshot.EditStructIndex])
	}

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	snapshot = cs.Snapshot()
	if !snapshot.EditEntryActive {
		t.Fatal("editEntryActive should be true for new key")
	}
	if snapshot.EditEntryKey != "testlang" {
		t.Fatalf("editEntryKey = %q, want \"testlang\"", snapshot.EditEntryKey)
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
	if m.configScreen.Snapshot().EditEntryActive {
		t.Fatal("should be back to key list")
	}
}
