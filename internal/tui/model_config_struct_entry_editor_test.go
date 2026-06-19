package tui

import "testing"

func TestConfigScreen_StructEntryInt_InvalidInput_DoesNotCloseOrDirty(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")
	cs := m.configScreen

	selectConfigEntryField(t, &m, "max_output_tokens")
	original := cs.ConfigSnapshot().ProviderModels["openai"].MaxOutputTokens

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", got)
	}

	setConfigInputValue(t, &m, "not-a-number")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	snapshot := cs.Snapshot()
	if snapshot.EditEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit after invalid input = %q, want \"input\"", snapshot.EditEntryFieldEdit)
	}
	if snapshot.Dirty {
		t.Fatal("dirty should remain false after invalid int input")
	}
	if snapshot.SaveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", snapshot.SaveStatus, statusSaved)
	}
	if got := cs.ConfigSnapshot().ProviderModels["openai"].MaxOutputTokens; got != original {
		t.Fatalf("ProviderModels[openai].MaxOutputTokens = %d, want %d", got, original)
	}
}

func TestConfigScreen_StructEntryInt_ValidInput_StillApplies(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")

	selectConfigEntryField(t, &m, "max_output_tokens")
	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", got)
	}

	setConfigInputValue(t, &m, "4321")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	snapshot := cs.Snapshot()
	if snapshot.EditEntryFieldEdit != "" {
		t.Fatalf("editEntryFieldEdit after valid input = %q, want empty", snapshot.EditEntryFieldEdit)
	}
	if got := cs.ConfigSnapshot().ProviderModels["openai"].MaxOutputTokens; got != 4321 {
		t.Fatalf("ProviderModels[openai].MaxOutputTokens = %d, want 4321", got)
	}
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after valid int input")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", snapshot.SaveStatus, statusModified)
	}
}

func TestConfigScreen_LSPServers_CommandChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")

	selectConfigEntryField(t, &m, "command")
	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", got)
	}

	setConfigInputValue(t, &m, "custom-gopls")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	srv, ok := cs.ConfigSnapshot().LSP.Servers["go"]
	if !ok {
		t.Fatal("go not found in LSP.Servers")
	}
	if srv.Command != "custom-gopls" {
		t.Fatalf("LSP.Servers[go].Command = %q, want \"custom-gopls\"", srv.Command)
	}
}

func TestConfigScreen_LSPServers_ArgsChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")
	cs := configTestScreen(t, m)

	selectConfigEntryField(t, &m, "args")
	origArgs := cs.ConfigSnapshot().LSP.Servers["go"].Args
	origLen := len(origArgs)

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "slice" {
		t.Fatalf("editEntryFieldEdit = %q, want \"slice\"", got)
	}

	m = sendConfigKey(m, "a")
	setConfigSliceInputValue(t, &m, "--extra-flag")
	m = sendConfigKey(m, "enter")

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)

	srv := cs.ConfigSnapshot().LSP.Servers["go"]
	if len(srv.Args) != origLen+1 {
		t.Fatalf("LSP.Servers[go].Args length = %d, want %d", len(srv.Args), origLen+1)
	}
	found := false
	for _, a := range srv.Args {
		if a == "--extra-flag" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("--extra-flag not found in LSP.Servers[go].Args: %v", srv.Args)
	}
}

func TestConfigScreen_LSPServers_DisabledToggle_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")
	cs := configTestScreen(t, m)

	before := cs.ConfigSnapshot().LSP.Servers["go"].Disabled
	selectConfigEntryField(t, &m, "disabled")

	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	after := cs.ConfigSnapshot().LSP.Servers["go"].Disabled
	if after == before {
		t.Fatalf("Disabled should have toggled: before=%v, after=%v", before, after)
	}

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	after2 := cs.ConfigSnapshot().LSP.Servers["go"].Disabled
	if after2 != before {
		t.Fatalf("Disabled should have toggled back: expected=%v, got=%v", before, after2)
	}
}
