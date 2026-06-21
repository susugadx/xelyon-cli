package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectScreen_EscBackTargets(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
			Rules:   []string{"rule"},
		},
	}

	t.Run("item returns to sections", func(t *testing.T) {
		m := newProjectTestModel(agent)
		m = moveProjectToSection(t, m, "rules")
		m = moveProjectToItemPane(t, m)
		m = sendProjectKey(m, "esc")
		if m.screen != screenProject {
			t.Fatalf("screen = %d, want screenProject", m.screen)
		}
		if got := projectSnapshot(t, m).ActivePane; got != "section" {
			t.Fatalf("activePane = %s, want section", got)
		}
	})

	t.Run("dirty opens confirm", func(t *testing.T) {
		m := newProjectTestModel(agent)
		m = editProjectContext(t, m, "dirty")
		m = sendProjectKey(m, "esc")
		if !projectSnapshot(t, m).ConfirmQuit {
			t.Fatal("dirty project screen should show quit confirmation")
		}
	})

	t.Run("clean closes", func(t *testing.T) {
		m := newProjectTestModel(agent)
		m = sendProjectKey(m, "esc")
		if m.screen != screenChat {
			t.Fatalf("screen = %d, want screenChat", m.screen)
		}
		if m.projectScreen != nil {
			t.Fatal("projectScreen should be nil after closing")
		}
	})
}
