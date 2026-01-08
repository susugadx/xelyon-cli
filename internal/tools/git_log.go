package tools

import (
	"fmt"
	"os/exec"
)

// executeGitLog は git log を実行
func executeGitLog() string {
	green.Println("📜 git log")
	cmd := exec.Command("git", "log", "--oneline", "-10")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	return result
}
