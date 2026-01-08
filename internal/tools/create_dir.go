package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// executeCreateDir はディレクトリを作成
func executeCreateDir(path string) string {
	if path == "" {
		return "Error: path is empty"
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if stat, err := os.Stat(absPath); err == nil {
		if stat.IsDir() {
			green.Printf("✅ Directory already exists: %s\n", path)
			return fmt.Sprintf("Directory already exists (idempotent): %s", path)
		}
		return fmt.Sprintf("Error: path exists but is not a directory: %s", path)
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err)
	}

	green.Printf("✅ Created directory: %s\n", path)
	return fmt.Sprintf("Successfully created directory: %s", path)
}
