package tui

import "testing"

func TestConfigScreen_StructEntryInt_InvalidInput_DoesNotCloseOrDirty(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")
	cs := m.configScreen

	setEntryFieldIndex(t, cs, "max_output_tokens")
	original := cs.cfg.ProviderModels["openai"].MaxOutputTokens

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	cs.editInput.SetValue("not-a-number")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit after invalid input = %q, want \"input\"", cs.editEntryFieldEdit)
	}
	if cs.dirty {
		t.Fatal("dirty should remain false after invalid int input")
	}
	if cs.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", cs.saveStatus, statusSaved)
	}
	if got := cs.cfg.ProviderModels["openai"].MaxOutputTokens; got != original {
		t.Fatalf("ProviderModels[openai].MaxOutputTokens = %d, want %d", got, original)
	}
}

func TestConfigScreen_StructEntryInt_ValidInput_StillApplies(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")
	cs := m.configScreen

	setEntryFieldIndex(t, cs, "max_output_tokens")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	cs.editInput.SetValue("4321")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.editEntryFieldEdit != "" {
		t.Fatalf("editEntryFieldEdit after valid input = %q, want empty", cs.editEntryFieldEdit)
	}
	if got := cs.cfg.ProviderModels["openai"].MaxOutputTokens; got != 4321 {
		t.Fatalf("ProviderModels[openai].MaxOutputTokens = %d, want 4321", got)
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after valid int input")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}

func TestConfigScreen_LSPServers_CommandChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")
	cs := m.configScreen

	setEntryFieldIndex(t, cs, "command")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	cs.editInput.SetValue("custom-gopls")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	srv, ok := cs.cfg.LSP.Servers["go"]
	if !ok {
		t.Fatal("go not found in LSP.Servers")
	}
	if srv.Command != "custom-gopls" {
		t.Fatalf("LSP.Servers[go].Command = %q, want \"custom-gopls\"", srv.Command)
	}
}

func TestConfigScreen_LSPServers_ArgsChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")
	cs := m.configScreen

	setEntryFieldIndex(t, cs, "args")
	origArgs := cs.cfg.LSP.Servers["go"].Args
	origLen := len(origArgs)

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "slice" {
		t.Fatalf("editEntryFieldEdit = %q, want \"slice\"", cs.editEntryFieldEdit)
	}

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editSliceInput.SetValue("--extra-flag")
	m = sendConfigKey(m, "enter")

	m = sendConfigKey(m, "esc")
	cs = m.configScreen

	srv := cs.cfg.LSP.Servers["go"]
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
	cs := m.configScreen

	before := cs.cfg.LSP.Servers["go"].Disabled
	setEntryFieldIndex(t, cs, "disabled")

	m = sendConfigKey(m, " ")
	cs = m.configScreen
	after := cs.cfg.LSP.Servers["go"].Disabled
	if after == before {
		t.Fatalf("Disabled should have toggled: before=%v, after=%v", before, after)
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	after2 := cs.cfg.LSP.Servers["go"].Disabled
	if after2 != before {
		t.Fatalf("Disabled should have toggled back: expected=%v, got=%v", before, after2)
	}
}
