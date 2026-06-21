package tui

import "testing"

func TestConfigScreen_CategoryNavigation(t *testing.T) {
	m := newConfigTestModel()

	if got := configSnapshot(t, m).categoryIndex; got != 0 {
		t.Fatalf("initial catIndex = %d, want 0", got)
	}

	m = sendConfigKey(m, "j")
	if got := configSnapshot(t, m).categoryIndex; got != 1 {
		t.Fatalf("catIndex after j = %d, want 1", got)
	}

	m = sendConfigKey(m, "k")
	if got := configSnapshot(t, m).categoryIndex; got != 0 {
		t.Fatalf("catIndex after k = %d, want 0", got)
	}

	m = sendConfigKey(m, "down")
	if got := configSnapshot(t, m).categoryIndex; got != 1 {
		t.Fatalf("catIndex after down = %d, want 1", got)
	}
}

func TestConfigScreen_PaneNavigation(t *testing.T) {
	m := newConfigTestModel()

	if got := configSnapshot(t, m).activePane; got != paneCategory {
		t.Fatalf("initial pane = %d, want paneCategory", got)
	}

	m = sendConfigKey(m, "l")
	if got := configSnapshot(t, m).activePane; got != paneField {
		t.Fatalf("pane after l = %d, want paneField", got)
	}

	m = sendConfigKey(m, "l")
	if got := configSnapshot(t, m).activePane; got != paneDetail {
		t.Fatalf("pane after l+l = %d, want paneDetail", got)
	}

	m = sendConfigKey(m, "h")
	if got := configSnapshot(t, m).activePane; got != paneField {
		t.Fatalf("pane after h = %d, want paneField", got)
	}

	m = sendConfigKey(m, "h")
	if got := configSnapshot(t, m).activePane; got != paneCategory {
		t.Fatalf("pane after h+h = %d, want paneCategory", got)
	}
}

func TestConfigScreen_EnterMovesToFieldPane(t *testing.T) {
	m := newConfigTestModel()
	m = sendConfigKey(m, "enter")
	if got := configSnapshot(t, m).activePane; got != paneField {
		t.Fatalf("pane after Enter = %d, want paneField", got)
	}
}

func TestConfigScreen_EscFromFieldPane(t *testing.T) {
	m := newConfigTestModel()
	selectConfigField(t, &m, "provider", "default_provider")

	m = sendConfigKey(m, "esc")
	snapshot := configSnapshot(t, m)
	if snapshot.activePane != paneCategory {
		t.Fatalf("pane after esc from field = %d, want paneCategory", snapshot.activePane)
	}
	if snapshot.fieldIndex != 0 {
		t.Fatalf("fieldIndex after esc = %d, want 0", snapshot.fieldIndex)
	}
}

func TestConfigScreen_FieldScroll_FollowsCursor(t *testing.T) {
	m := newConfigTestModel()
	m.height = 5
	selectConfigCategory(t, &m, "final_checks")
	m = sendConfigKey(m, "enter")

	cs := configTestScreen(t, m)
	fields := cs.Snapshot().FilteredFields
	if len(fields) < 2 {
		t.Skipf("final_checks category has %d fields, need >=2 for scroll test", len(fields))
	}

	m = sendConfigKey(m, "j")
	snapshot := configSnapshot(t, m)
	if snapshot.fieldIndex != 1 {
		t.Fatalf("fieldIndex = %d, want 1", snapshot.fieldIndex)
	}
	if snapshot.fieldIndex < snapshot.fieldScroll || snapshot.fieldIndex >= snapshot.fieldScroll+3 {
		t.Fatalf("fieldIndex=%d out of visible range [%d, %d)", snapshot.fieldIndex, snapshot.fieldScroll, snapshot.fieldScroll+3)
	}

	m = sendConfigKey(m, "k")
	snapshot = configSnapshot(t, m)
	if snapshot.fieldIndex != 0 {
		t.Fatalf("fieldIndex = %d, want 0", snapshot.fieldIndex)
	}
	if snapshot.fieldScroll != 0 {
		t.Fatalf("fieldScroll = %d, want 0", snapshot.fieldScroll)
	}
}

func TestConfigScreen_FieldScroll_ResetOnCategoryChange(t *testing.T) {
	m := newConfigTestModel()
	m.height = 5
	selectConfigCategory(t, &m, "final_checks")
	m = sendConfigKey(m, "enter")
	m = sendConfigKeys(m, "j", "j", "j")
	if configSnapshot(t, m).fieldScroll == 0 {
		t.Skip("fieldScroll did not advance, not enough fields")
	}

	m = sendConfigKey(m, "esc")
	snapshot := configSnapshot(t, m)
	if snapshot.fieldIndex != 0 {
		t.Fatalf("fieldIndex = %d after Esc, want 0", snapshot.fieldIndex)
	}
	if snapshot.fieldScroll != 0 {
		t.Fatalf("fieldScroll = %d after Esc, want 0", snapshot.fieldScroll)
	}
}
