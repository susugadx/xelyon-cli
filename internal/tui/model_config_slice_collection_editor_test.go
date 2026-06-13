package tui

import (
	"reflect"
	"testing"
)

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

func TestConfigScreen_AgentInstructionProjectFilesChooser(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen
	setConfigFieldSelection(t, cs, "agent_instructions", "agent_instructions.project.files")

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSlice || len(cs.editGuidanceChoices) == 0 {
		t.Fatalf("guidance chooser not opened: editMode=%d choices=%#v", cs.editMode, cs.editGuidanceChoices)
	}

	m = sendConfigKey(m, " ")
	m = sendConfigKey(m, "down")
	m = sendConfigKey(m, " ")
	m = sendConfigKey(m, "esc")

	got := m.configScreen.cfg.AgentInstructions.Project.Files
	want := []string{"CLAUDE.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("project files = %#v, want %#v", got, want)
	}
}

func TestConfigScreen_AgentInstructionGlobalFilesChooserAllowsCodexAsset(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen
	setConfigFieldSelection(t, cs, "agent_instructions", "agent_instructions.global.files")

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSlice || len(cs.editGuidanceChoices) == 0 {
		t.Fatalf("guidance chooser not opened: editMode=%d choices=%#v", cs.editMode, cs.editGuidanceChoices)
	}

	m = sendConfigKey(m, "down")
	m = sendConfigKey(m, " ")
	m = sendConfigKey(m, "esc")

	got := m.configScreen.cfg.AgentInstructions.Global.Files
	want := []string{"~/.xelyon/AGENTS.md", "~/.codex/AGENTS.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("global files = %#v, want %#v", got, want)
	}
}
