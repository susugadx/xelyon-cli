package search

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// Colors from common package
var green = common.Green

// ExecuteSearchCode searches for a pattern in code files
func ExecuteSearchCode(pattern string, path string) string {
	if pattern == "" {
		return "Error: pattern is required"
	}

	// Cache check
	if tools.GlobalToolCache != nil {
		if cached, ok := tools.GlobalToolCache.GetSearch(pattern, path); ok {
			return cached
		}
	}

	green.Printf("🔍 Searching for '%s' in %s\n", pattern, path)

	// grep search (-r: recursive, -n: line numbers, -I: exclude binary)
	cmd := exec.Command("grep", "-rn", "-I", "--include=*.go", "--include=*.js", "--include=*.ts", "--include=*.py", "--include=*.md", "--include=*.json", "--include=*.yaml", "--include=*.yml", pattern, path)
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		// grep returns error when no matches
		if result == "" {
			noMatchResult := fmt.Sprintf("No matches found for '%s'", pattern)
			if tools.GlobalToolCache != nil {
				tools.GlobalToolCache.SetSearch(pattern, path, noMatchResult)
			}
			return noMatchResult
		}
	}

	// Truncate long results
	lines := strings.Split(result, "\n")
	if len(lines) > 50 {
		result = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... (%d more matches)", len(lines)-50)
	}

	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetSearch(pattern, path, result)
	}
	return result
}
