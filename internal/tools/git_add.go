package tools

import (
	"fmt"
	"os/exec"
)

// executeGitAdd は git add を実行
func executeGitAdd(path string) string {
	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("➕ Git Stage / Gitステージング\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !confirm("Stage this file? / このファイルをステージングしますか？") {
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "add", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	return fmt.Sprintf("✅ Staged: %s", path)
}
