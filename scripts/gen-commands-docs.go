//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/scripts/internal/commanddocs"
)

func main() {
	// docs/commands.md を読み込み
	inputPath := "docs/commands.md"
	content, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	// 既存のコマンドセクションを検出
	existingCommands := commanddocs.FindExistingCommands(string(content))

	// 不足しているコマンドを検出
	missingCommands := commanddocs.MissingCommands(commandcatalog.Commands, existingCommands)

	if len(missingCommands) == 0 {
		fmt.Println("All commands are documented in docs/commands.md")
		return
	}

	// 骨格を生成
	skeleton := commanddocs.RenderMissingCommandSkeleton(missingCommands)

	// ファイルに追記
	newContent := string(content) + skeleton
	if err := os.WriteFile(inputPath, []byte(newContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added %d missing command(s) to %s:\n", len(missingCommands), inputPath)
	for _, cmd := range missingCommands {
		fmt.Printf("  - %s\n", cmd.Name)
	}
}
