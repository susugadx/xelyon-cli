package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handleProjectCommand は /project コマンドを処理（xelyon.yaml の対話式編集）
func handleProjectCommand(agent *Agent) bool {
	pc := config.LoadProjectConfig()
	if pc == nil {
		yellow.Println("⚠️  xelyon.yaml not found")
		yellow.Println("   Run /init to create a template")
		return true
	}

	menu := ui.NewProjectMenu(pc)
	changed, err := menu.Run()
	if err != nil {
		red.Printf("Error: %v\n", err)
		return true
	}

	if !changed {
		yellow.Println("Cancelled")
		return true
	}

	if err := config.SaveProjectConfig(pc); err != nil {
		red.Printf("Failed to save: %v\n", err)
		return true
	}

	green.Println("✅ xelyon.yaml saved")
	green.Println("   Changes will take effect on next session")
	return true
}
