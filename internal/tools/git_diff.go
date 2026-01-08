package tools

import (
	"fmt"
	"os/exec"
)

// executeGitDiff は git diff を実行
func executeGitDiff(path string) string {
	args := []string{"diff"}
	if path != "" {
		args = append(args, path)
	}
	green.Printf("📝 git diff %s\n", path)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	if result == "" {
		result = "No changes"
	}
	if len(result) > 3000 {
		result = result[:3000] + "\n... (truncated)"
	}
	return result
}
