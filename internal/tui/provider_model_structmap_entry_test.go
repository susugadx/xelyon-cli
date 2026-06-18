package tui

import "testing"

func TestConfigScreen_ProviderModels_EntryEdit(t *testing.T) {
	m := enterStructMapEdit(t, "provider_models")
	cs := m.configScreen

	if len(cs.editStructKeys) == 0 {
		t.Fatal("no provider_models keys")
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatal("editEntryActive should be true")
	}
	if len(cs.editEntryFields) == 0 {
		t.Fatal("editEntryFields should not be empty")
	}

	selectConfigEntryField(t, &m, "default_model")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if cs.editEntryFieldEdit != "" {
		t.Fatalf("editEntryFieldEdit = %q after esc, want empty", cs.editEntryFieldEdit)
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if cs.editEntryActive {
		t.Fatal("editEntryActive should be false after esc")
	}
}

func TestConfigScreen_ProviderModels_ValueChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")

	selectConfigEntryField(t, &m, "default_model")

	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	setConfigInputValue(t, &m, "gpt-99-turbo")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	pm, ok := cs.cfg.ProviderModels["openai"]
	if !ok {
		t.Fatal("openai not found in ProviderModels")
	}
	if pm.DefaultModel != "gpt-99-turbo" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"gpt-99-turbo\"", pm.DefaultModel)
	}

	if !cs.dirty {
		t.Fatal("dirty should be true after value change")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}

func TestConfigScreen_ProviderModels_CatalogModelChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")

	selectConfigEntryField(t, &m, "catalog_model")
	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	setConfigInputValue(t, &m, "gpt-5.4")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if got := cs.cfg.ProviderModels["openai"].CatalogModel; got != "gpt-5.4" {
		t.Fatalf("ProviderModels[openai].CatalogModel = %q, want gpt-5.4", got)
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after catalog_model change")
	}
}
