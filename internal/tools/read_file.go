package tools

import (
	"fmt"
	"os"
)

// executeReadFile はファイルを読み込む
func executeReadFile(path string) string {
	if path == "" {
		return "Error: path is empty"
	}

	// パストラバーサル防止
	absPath, err := ValidatePath(path)
	if err != nil {
		red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	result := string(content)
	green.Printf("📄 Read: %s (%d bytes)\n", path, len(result))

	// 長すぎる場合は切り詰め
	if len(result) > 10000 {
		result = result[:10000] + "\n... (truncated, showing first 10000 chars)"
	}

	return result
}
