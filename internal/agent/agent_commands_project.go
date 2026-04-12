package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type projectCommandMenu interface {
	Run() (bool, error)
}

var (
	loadProjectConfigForCommand = config.LoadProjectConfig
	saveProjectConfigForCommand = config.SaveProjectConfig
	newProjectMenuForCommand = func(pc *config.ProjectConfig, runtime *ui.Runtime) projectCommandMenu {
		return ui.NewProjectMenuWithRuntime(pc, runtime)
	}
)

// handleProjectCommand は /project コマンドを処理（xelyon.yaml の対話式編集）
func handleProjectCommand(agent *Agent) bool {
	out := agent.output()
	pc := loadProjectConfigForCommand()
	if pc == nil {
		yellow.Fprintln(out, "⚠️  xelyon.yaml not found")
		yellow.Fprintln(out, "   Run /init to create a template")
		return true
	}

	menu := newProjectMenuForCommand(pc, agent.ui())
	changed, err := menu.Run()
	if err != nil {
		red.Fprintf(out, "Error: %v\n", err)
		return true
	}

	if !changed {
		yellow.Fprintln(out, "Cancelled")
		return true
	}

	if err := saveProjectConfigForCommand(pc); err != nil {
		red.Fprintf(out, "Failed to save: %v\n", err)
		return true
	}

	green.Fprintln(out, "✅ xelyon.yaml saved")
	green.Fprintln(out, "   Changes will take effect on next session")
	return true
}
