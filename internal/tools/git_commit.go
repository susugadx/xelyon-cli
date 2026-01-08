package tools

import (
	"fmt"
	"os/exec"
)

// executeGitCommit は git commit を実行
func executeGitCommit(message string) string {
	if message == "" {
		return "Error: commit message is required"
	}

	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("💾 Git Commit / Gitコミット\n")
	cyan.Printf("📝 Message / メッセージ:\n%s\n", message)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !confirm("Commit with this message? / この内容でコミットしますか？") {
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	fmt.Println(result)
	return result
}
