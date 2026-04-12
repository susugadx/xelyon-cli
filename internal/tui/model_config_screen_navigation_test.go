package tui

import "testing"

func TestConfigScreen_CategoryNavigation(t *testing.T) {
	m := newConfigTestModel()

	if m.configScreen.catIndex != 0 {
		t.Fatalf("initial catIndex = %d, want 0", m.configScreen.catIndex)
	}

	m = sendConfigKey(m, "j")
	if m.configScreen.catIndex != 1 {
		t.Fatalf("catIndex after j = %d, want 1", m.configScreen.catIndex)
	}

	m = sendConfigKey(m, "k")
	if m.configScreen.catIndex != 0 {
		t.Fatalf("catIndex after k = %d, want 0", m.configScreen.catIndex)
	}

	m = sendConfigKey(m, "down")
	if m.configScreen.catIndex != 1 {
		t.Fatalf("catIndex after down = %d, want 1", m.configScreen.catIndex)
	}
}

func TestConfigScreen_PaneNavigation(t *testing.T) {
	m := newConfigTestModel()

	if m.configScreen.activePane != paneCategory {
		t.Fatalf("initial pane = %d, want paneCategory", m.configScreen.activePane)
	}

	m = sendConfigKey(m, "l")
	if m.configScreen.activePane != paneField {
		t.Fatalf("pane after l = %d, want paneField", m.configScreen.activePane)
	}

	m = sendConfigKey(m, "l")
	if m.configScreen.activePane != paneDetail {
		t.Fatalf("pane after l+l = %d, want paneDetail", m.configScreen.activePane)
	}

	m = sendConfigKey(m, "h")
	if m.configScreen.activePane != paneField {
		t.Fatalf("pane after h = %d, want paneField", m.configScreen.activePane)
	}

	m = sendConfigKey(m, "h")
	if m.configScreen.activePane != paneCategory {
		t.Fatalf("pane after h+h = %d, want paneCategory", m.configScreen.activePane)
	}
}

func TestConfigScreen_EnterMovesToFieldPane(t *testing.T) {
	m := newConfigTestModel()
	m = sendConfigKey(m, "enter")
	if m.configScreen.activePane != paneField {
		t.Fatalf("pane after Enter = %d, want paneField", m.configScreen.activePane)
	}
}

func TestConfigScreen_EscFromFieldPane(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	cs.activePane = paneField
	cs.fieldIndex = 1

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.activePane != paneCategory {
		t.Fatalf("pane after esc from field = %d, want paneCategory", cs.activePane)
	}
	if cs.fieldIndex != 0 {
		t.Fatalf("fieldIndex after esc = %d, want 0", cs.fieldIndex)
	}
}

func TestConfigScreen_FieldScroll_FollowsCursor(t *testing.T) {
	m := newConfigTestModel()
	m.height = 5
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "hooks" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	cs.fieldIndex = 0
	cs.fieldScroll = 0

	fields := cs.filteredFields()
	if len(fields) < 4 {
		t.Skipf("hooks category has %d fields, need >=4 for scroll test", len(fields))
	}

	m = sendConfigKeys(m, "j", "j", "j")
	cs = m.configScreen
	if cs.fieldIndex != 3 {
		t.Fatalf("fieldIndex = %d, want 3", cs.fieldIndex)
	}
	if cs.fieldScroll < 1 {
		t.Fatalf("fieldScroll = %d, want >= 1 (fieldIndex=3 with 3 visible rows)", cs.fieldScroll)
	}
	if cs.fieldIndex < cs.fieldScroll || cs.fieldIndex >= cs.fieldScroll+3 {
		t.Fatalf("fieldIndex=%d out of visible range [%d, %d)", cs.fieldIndex, cs.fieldScroll, cs.fieldScroll+3)
	}

	m = sendConfigKeys(m, "k", "k", "k")
	cs = m.configScreen
	if cs.fieldIndex != 0 {
		t.Fatalf("fieldIndex = %d, want 0", cs.fieldIndex)
	}
	if cs.fieldScroll != 0 {
		t.Fatalf("fieldScroll = %d, want 0", cs.fieldScroll)
	}
}

func TestConfigScreen_FieldScroll_ResetOnCategoryChange(t *testing.T) {
	m := newConfigTestModel()
	m.height = 5
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "hooks" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	m = sendConfigKeys(m, "j", "j", "j")
	cs = m.configScreen
	if cs.fieldScroll == 0 {
		t.Skip("fieldScroll did not advance, not enough fields")
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.fieldIndex != 0 {
		t.Fatalf("fieldIndex = %d after Esc, want 0", cs.fieldIndex)
	}
	if cs.fieldScroll != 0 {
		t.Fatalf("fieldScroll = %d after Esc, want 0", cs.fieldScroll)
	}
}
