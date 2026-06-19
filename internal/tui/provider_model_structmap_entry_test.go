package tui

import "testing"

func TestConfigScreen_ProviderModels_EntryEdit(t *testing.T) {
	m := enterStructMapEdit(t, "provider_models")
	cs := m.configScreen

	if len(cs.Snapshot().EditStructKeys) == 0 {
		t.Fatal("no provider_models keys")
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	snapshot := cs.Snapshot()
	if !snapshot.EditEntryActive {
		t.Fatal("editEntryActive should be true")
	}
	if len(snapshot.EditEntryFields) == 0 {
		t.Fatal("editEntryFields should not be empty")
	}

	selectConfigEntryField(t, &m, "default_model")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", got)
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "" {
		t.Fatalf("editEntryFieldEdit = %q after esc, want empty", got)
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if cs.Snapshot().EditEntryActive {
		t.Fatal("editEntryActive should be false after esc")
	}
}

func TestConfigScreen_ProviderModels_ValueChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")

	selectConfigEntryField(t, &m, "default_model")

	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", got)
	}

	setConfigInputValue(t, &m, "gpt-99-turbo")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	pm, ok := cs.ConfigSnapshot().ProviderModels["openai"]
	if !ok {
		t.Fatal("openai not found in ProviderModels")
	}
	if pm.DefaultModel != "gpt-99-turbo" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"gpt-99-turbo\"", pm.DefaultModel)
	}

	snapshot := cs.Snapshot()
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after value change")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", snapshot.SaveStatus, statusModified)
	}
}

func TestConfigScreen_ProviderModels_CatalogModelChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")

	selectConfigEntryField(t, &m, "catalog_model")
	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", got)
	}

	setConfigInputValue(t, &m, "gpt-5.4")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if got := cs.ConfigSnapshot().ProviderModels["openai"].CatalogModel; got != "gpt-5.4" {
		t.Fatalf("ProviderModels[openai].CatalogModel = %q, want gpt-5.4", got)
	}
	if !cs.Snapshot().Dirty {
		t.Fatal("dirty should be true after catalog_model change")
	}
}
