package search

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ExecuteSearchFile searches for files by name pattern
func ExecuteSearchFile(pattern string, path string) string {
	if pattern == "" {
		return "Error: pattern is required"
	}

	// Cache check
	if tools.GlobalToolCache != nil {
		if cached, ok := tools.GlobalToolCache.GetSearch(pattern, path); ok {
			return cached
		}
	}

	green.Printf("📁 Searching for files matching '%s' in %s\n", pattern, path)

	// find search (exclude .git)
	cmd := exec.Command("find", path, "-type", "f", "-name", pattern, "-not", "-path", "*/.git/*")
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, result)
	}

	if strings.TrimSpace(result) == "" {
		noMatchResult := fmt.Sprintf("No files found matching '%s'", pattern)
		if tools.GlobalToolCache != nil {
			tools.GlobalToolCache.SetSearch(pattern, path, noMatchResult)
		}
		return noMatchResult
	}

	// Truncate long results
	lines := strings.Split(result, "\n")
	if len(lines) > 30 {
		result = strings.Join(lines[:30], "\n") + fmt.Sprintf("\n... (%d more files)", len(lines)-30)
	}

	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetSearch(pattern, path, result)
	}
	return result
}
