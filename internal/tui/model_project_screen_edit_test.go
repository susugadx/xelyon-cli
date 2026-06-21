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

	m = moveProjectToSection(t, m, "ignore")
	m = sendProjectKey(m, "a")
	if got := projectSnapshot(t, m).EditMode; got != "line" {
		t.Fatalf("editMode = %s, want line", got)
	}
	m = sendProjectText(m, "dist")
	m = sendProjectKey(m, "enter")

	if got := projectSnapshot(t, m).Config.Ignore.Patterns; len(got) != 1 || got[0] != "dist" {
		t.Fatalf("ignore patterns = %#v, want [dist]", got)
	}

	m = moveProjectToItemPane(t, m)
	m = sendProjectKey(m, "d")
	if got := projectSnapshot(t, m).Config.Ignore.Patterns; len(got) != 0 {
		t.Fatalf("ignore patterns after delete = %#v, want empty", got)
	}
}
