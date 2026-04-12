package tui

import "testing"

func TestConfigScreen_SliceEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "execution.safe_shell_commands" {
			cs.fieldIndex = i
			break
		}
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSlice {
		t.Fatalf("editMode = %d, want editSlice(%d)", cs.editMode, editSlice)
	}

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	if !cs.editSliceAdding {
		t.Fatal("editSliceAdding should be true")
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editSliceAdding {
		t.Fatal("editSliceAdding should be false after esc")
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", cs.editMode)
	}
}
