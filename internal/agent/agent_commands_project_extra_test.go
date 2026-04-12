package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type scriptedProjectMenu struct {
	changed bool
	err     error
}

func (m *scriptedProjectMenu) Run() (bool, error) {
	return m.changed, m.err
}

func withProjectCommandHooks(t *testing.T) {
	t.Helper()

	oldLoad := loadProjectConfigForCommand
	oldSave := saveProjectConfigForCommand
	oldMenu := newProjectMenuForCommand
	t.Cleanup(func() {
		loadProjectConfigForCommand = oldLoad
		saveProjectConfigForCommand = oldSave
		newProjectMenuForCommand = oldMenu
	})
}

func TestHandleProjectCommand_ProjectMenuFlows(t *testing.T) {
	newAgent := func() (*Agent, *bytes.Buffer) {
		var out bytes.Buffer
		return &Agent{
			Runtime: &AgentRuntime{
				UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
			},
		}, &out
	}

	t.Run("menu error is reported", func(t *testing.T) {
		withProjectCommandHooks(t)
		loadProjectConfigForCommand = func() *config.ProjectConfig {
			return &config.ProjectConfig{Context: "ctx"}
		}
		newProjectMenuForCommand = func(pc *config.ProjectConfig, runtime *ui.Runtime) projectCommandMenu {
			return &scriptedProjectMenu{err: errors.New("menu failed")}
		}

		agent, out := newAgent()
		if !handleProjectCommand(agent) {
			t.Fatal("handleProjectCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Error: menu failed") {
			t.Fatalf("output = %q, want menu error", out.String())
		}
	})

	t.Run("cancel reports cancelled", func(t *testing.T) {
		withProjectCommandHooks(t)
		loadProjectConfigForCommand = func() *config.ProjectConfig {
			return &config.ProjectConfig{Context: "ctx"}
		}
		newProjectMenuForCommand = func(pc *config.ProjectConfig, runtime *ui.Runtime) projectCommandMenu {
			return &scriptedProjectMenu{changed: false}
		}

		agent, out := newAgent()
		if !handleProjectCommand(agent) {
			t.Fatal("handleProjectCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Cancelled") {
			t.Fatalf("output = %q, want cancelled message", out.String())
		}
	})

	t.Run("save error is reported", func(t *testing.T) {
		withProjectCommandHooks(t)
		loadProjectConfigForCommand = func() *config.ProjectConfig {
			return &config.ProjectConfig{Context: "ctx"}
		}
		newProjectMenuForCommand = func(pc *config.ProjectConfig, runtime *ui.Runtime) projectCommandMenu {
			return &scriptedProjectMenu{changed: true}
		}
		saveProjectConfigForCommand = func(pc *config.ProjectConfig) error {
			return errors.New("save failed")
		}

		agent, out := newAgent()
		if !handleProjectCommand(agent) {
			t.Fatal("handleProjectCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Failed to save: save failed") {
			t.Fatalf("output = %q, want save error", out.String())
		}
	})

	t.Run("success reports saved", func(t *testing.T) {
		withProjectCommandHooks(t)
		loadProjectConfigForCommand = func() *config.ProjectConfig {
			return &config.ProjectConfig{Context: "ctx"}
		}
		newProjectMenuForCommand = func(pc *config.ProjectConfig, runtime *ui.Runtime) projectCommandMenu {
			return &scriptedProjectMenu{changed: true}
		}
		saveProjectConfigForCommand = func(pc *config.ProjectConfig) error {
			return nil
		}

		agent, out := newAgent()
		if !handleProjectCommand(agent) {
			t.Fatal("handleProjectCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "xelyon.yaml saved") {
			t.Fatalf("output = %q, want saved message", out.String())
		}
	})
}
