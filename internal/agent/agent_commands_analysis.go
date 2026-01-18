package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

// handleRepoMapCommand はリポジトリマップを表示
func handleRepoMapCommand() bool {
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "." // フォールバック
	}
	rm := repomap.NewRepoMap(cwd, 0) // 制限なし
	if err := rm.Build(); err != nil {
		red.Printf("Failed to build repo map: %v\n", err)
		return true
	}

	if rm.GetSymbolCount() == 0 {
		yellow.Println("No symbols found in current directory")
		return true
	}

	cyan.Printf("🗺️  Repository Map (%d symbols from %d files)\n\n",
		rm.GetSymbolCount(), len(rm.Files))
	fmt.Println(rm.Generate())
	return true
}

// handleDryRunCommand handles the /dryrun command to toggle dry-run mode
func handleDryRunCommand(agent *Agent, args []string) bool {
	if len(args) > 0 {
		switch args[0] {
		case "on":
			agent.DryRunMode = true
			green.Println("✅ Dry-Run Mode enabled. Tool executions will be simulated.")
			return true
		case "off":
			agent.DryRunMode = false
			green.Println("✅ Dry-Run Mode disabled. Tools will be executed normally.")
			return true
		case "status":
			status := "disabled"
			if agent.DryRunMode {
				status = "enabled"
			}
			cyan.Printf("Dry-Run Mode is currently %s\n", status)
			return true
		}
	}

	// Toggle if no args
	agent.DryRunMode = !agent.DryRunMode
	status := "disabled"
	if agent.DryRunMode {
		status = "enabled"
	}
	green.Printf("✅ Dry-Run Mode %s\n", status)
	return true
}
