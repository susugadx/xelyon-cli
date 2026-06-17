//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/susugadx/xelyon-cli/scripts/internal/helpgen"
	"github.com/susugadx/xelyon-cli/scripts/internal/scriptio"
)

func main() {
	rootDir, err := resolveProjectRoot()
	if err != nil {
		scriptio.ExitWithError("Error resolving project root: %v", err)
	}

	// ファイル出力
	outputPath := filepath.Join(rootDir, "internal", "agent", "help_generated.go")
	if err := os.WriteFile(outputPath, helpgen.GenerateHelpSource(), 0644); err != nil {
		scriptio.ExitWithError("Error writing file: %v", err)
	}

	fmt.Printf("Generated %s\n", outputPath)
}

func resolveProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
	}
}
