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
		m.projectScreen.sectionIndex = int(projectSectionRules)
		m.projectScreen.activePane = projectPaneItem
		m = sendProjectKey(m, "esc")
		if m.screen != screenProject {
			t.Fatalf("screen = %d, want screenProject", m.screen)
		}
		if m.projectScreen.activePane != projectPaneSection {
			t.Fatalf("activePane = %d, want section", m.projectScreen.activePane)
		}
	})

	t.Run("dirty opens confirm", func(t *testing.T) {
		m := newProjectTestModel(agent)
		m.projectScreen.setDirty()
		m = sendProjectKey(m, "esc")
		if !m.projectScreen.confirmQuit {
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
