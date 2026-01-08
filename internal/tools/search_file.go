package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// executeSearchFile はファイル名で検索（find）
func executeSearchFile(pattern string, path string) string {
	if pattern == "" {
		return "Error: pattern is required"
	}

	green.Printf("📁 Searching for files matching '%s' in %s\n", pattern, path)

	// findで検索（.gitは除外）
	cmd := exec.Command("find", path, "-type", "f", "-name", pattern, "-not", "-path", "*/.git/*")
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, result)
	}

	if strings.TrimSpace(result) == "" {
		return fmt.Sprintf("No files found matching '%s'", pattern)
	}

	// 結果が長すぎる場合は切り詰め
	lines := strings.Split(result, "\n")
	if len(lines) > 30 {
		result = strings.Join(lines[:30], "\n") + fmt.Sprintf("\n... (%d more files)", len(lines)-30)
	}

	return result
}
