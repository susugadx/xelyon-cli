package tui

import (
	"reflect"
	"testing"
)

func TestConfigScreen_SliceEdit(t *testing.T) {
	m := newConfigTestModel()
	openConfigSliceEditor(t, &m, "execution", "execution.safe_shell_commands")

	m = sendConfigKey(m, "a")
	cs := configTestScreen(t, m)
	if !cs.editSliceAdding {
		t.Fatal("editSliceAdding should be true")
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if cs.editSliceAdding {
		t.Fatal("editSliceAdding should be false after esc")
	}

	m = sendConfigKey(m, "esc")
	cs = configTestScreen(t, m)
	if cs.editMode != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", cs.editMode)
	}
}

func TestConfigScreen_AgentInstructionProjectFilesChooser(t *testing.T) {
	m := newConfigTestModel()
	selectConfigField(t, &m, "agent_instructions", "agent_instructions.project.files")

	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
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
	selectConfigField(t, &m, "agent_instructions", "agent_instructions.global.files")

	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
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
