//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/scripts/internal/commanddocs"
	"github.com/susugadx/xelyon-cli/scripts/internal/scriptio"
)

func main() {
	// docs/commands.md を読み込み
	inputPath := "docs/commands.md"
	content, err := os.ReadFile(inputPath)
	if err != nil {
		scriptio.ExitWithError("Error reading %s: %v", inputPath, err)
	}

	newContent, missingCommands := commanddocs.AppendMissingCommandSkeleton(string(content), commandcatalog.Commands)
	if len(missingCommands) == 0 {
		fmt.Println("All commands are documented in docs/commands.md")
		return
	}

	if err := os.WriteFile(inputPath, []byte(newContent), 0644); err != nil {
		scriptio.ExitWithError("Error writing file: %v", err)
	}

	fmt.Printf("Added %d missing command(s) to %s:\n", len(missingCommands), inputPath)
	for _, cmd := range missingCommands {
		fmt.Printf("  - %s\n", cmd.Name)
	}
}
