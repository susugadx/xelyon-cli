package tools

import (
	"fmt"
	"os/exec"
)

// executeGitStatus は git status を実行
func executeGitStatus() string {
	green.Println("📊 git status")
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	if result == "" {
		result = "✨ Working tree clean"
	}
	return result
}
