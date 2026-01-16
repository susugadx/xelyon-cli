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

// handlePlanCommand はPlan Modeの切り替えを処理
func handlePlanCommand(agent *Agent, args []string) bool {
	// 引数なし → 現在の状態を表示
	if len(args) == 0 {
		if agent.PlanMode {
			green.Println("📋 Plan Mode: ON")
			yellow.Println("   AI will generate execution plans for your requests.")
			yellow.Println("   Use '/plan off' to disable.")
		} else {
			yellow.Println("📋 Plan Mode: OFF")
			yellow.Println("   AI will execute tasks directly without planning.")
			yellow.Println("   Use '/plan on' to enable.")
		}
		return true
	}

	// /plan on → 有効化
	if args[0] == "on" {
		if agent.PlanMode {
			yellow.Println("Plan Mode is already enabled")
			return true
		}
		agent.PlanMode = true
		green.Println("✅ Plan Mode enabled")
		cyan.Println("   AI will now generate execution plans for your requests.")
		cyan.Println("   You can review and approve each plan before execution.")
		return true
	}

	// /plan off → 無効化
	if args[0] == "off" {
		if !agent.PlanMode {
			yellow.Println("Plan Mode is already disabled")
			return true
		}
		agent.PlanMode = false
		green.Println("✅ Plan Mode disabled")
		cyan.Println("   AI will now execute tasks directly without planning.")
		return true
	}

	// 不正な引数
	yellow.Println("Usage: /plan [on|off]")
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
