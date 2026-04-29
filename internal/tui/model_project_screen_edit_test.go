package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectScreen_ListEditAddsAndDeletes(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m.projectScreen.sectionIndex = int(projectSectionIgnore)
	m = sendProjectKey(m, "a")
	if m.projectScreen.editMode != projectEditLine {
		t.Fatalf("editMode = %d, want line edit", m.projectScreen.editMode)
	}
	m.projectScreen.editInput.SetValue("dist")
	m = sendProjectKey(m, "enter")

	if got := m.projectScreen.pc.Ignore.Patterns; len(got) != 1 || got[0] != "dist" {
		t.Fatalf("ignore patterns = %#v, want [dist]", got)
	}

	m.projectScreen.activePane = projectPaneItem
	m = sendProjectKey(m, "d")
	if got := m.projectScreen.pc.Ignore.Patterns; len(got) != 0 {
		t.Fatalf("ignore patterns after delete = %#v, want empty", got)
	}
}
