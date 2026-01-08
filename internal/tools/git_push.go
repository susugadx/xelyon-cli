package tools

import (
	"fmt"
	"os/exec"
)

// executeGitPush は git push を実行
func executeGitPush() string {
	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🚀 Git Push / リモートへプッシュ\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	yellow.Println("⚠️  Warning: Changes will be published to remote / 警告: リモートリポジトリに変更が公開されます")

	if !confirm("Push to remote? / プッシュしますか？") {
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	if result == "" {
		result = "✅ Pushed successfully"
	}
	fmt.Println(result)
	return result
}
