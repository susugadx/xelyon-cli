package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// 自動実行可能なコマンド（安全なもの）
var safeCommands = map[string]bool{
	"ls": true, "cat": true, "pwd": true, "echo": true, "which": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"git status": true, "git log": true, "git diff": true, "git branch": true,
	"git ls-files": true, "git show": true, "git remote": true,
	"go version": true, "go mod tidy": true,
	"node -v": true, "npm -v": true, "npm list": true,
	"python --version": true, "pip list": true,
}

// ブロックするコマンド（危険すぎ）
var blockedCommands = []string{
	"rm -rf /", "rm -rf ~", "rm -rf *",
	"sudo rm", "sudo chmod", "sudo chown",
	"chmod 777", "chmod -R 777",
	"mkfs", "dd if=", ":(){:|:&};:",
	"> /dev/sda", "mv / ",
	"sed -i", "sed -e", "sed '", // ファイル編集系sed
	"awk -i", "perl -i", "perl -p", // その他のインライン編集コマンド
}

// executeBash はシェルコマンドを実行
func executeBash(command string) string {
	if command == "" {
		return "Error: command is empty"
	}

	// ブロックチェック
	for _, blocked := range blockedCommands {
		if strings.Contains(command, blocked) {
			red.Printf("🚫 Blocked dangerous command: %s\n", command)
			return "Error: This command is blocked for safety"
		}
	}

	// 安全なコマンドか確認
	needConfirm := true
	for safe := range safeCommands {
		if strings.HasPrefix(command, safe) {
			needConfirm = false
			break
		}
	}

	// 確認が必要な場合
	if needConfirm {
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("⚙️  Shell Command / シェルコマンド実行\n")
		cyan.Printf("📜 Command / コマンド: %s\n", command)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This command may modify your system / 警告: システムに変更が加わる可能性があります")

		if !confirm("Run this command? / 実行しますか？") {
			return "Cancelled by user"
		}
	}

	// 実行
	green.Printf("▶ Running: %s\n", command)
	cmd := exec.Command("bash", "-c", command)
	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	} else {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cmd.Dir = "." // フォールバック
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))
	}

	result := string(output)
	if len(result) > 5000 {
		result = result[:5000] + "\n... (truncated)"
	}

	return result
}
