package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
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

// コマンド連結文字（インジェクション攻撃防止）
var commandSeparators = []string{
	";", "&&", "||", "|", "`", "$(", ">", ">>", "<",
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

	// コマンド連結攻撃の検知（safeCommands以外）
	isSafe := false
	for safe := range safeCommands {
		if strings.HasPrefix(command, safe) {
			isSafe = true
			break
		}
	}
	if !isSafe {
		for _, sep := range commandSeparators {
			if strings.Contains(command, sep) {
				red.Printf("🚫 Blocked command injection attempt: %s\n", command)
				yellow.Println("⚠️  Command contains potentially dangerous separator characters.")
				yellow.Println("   If you need to run multiple commands, execute them separately.")
				return "Error: Command injection attempt detected (separator found)"
			}
		}
	}

	// 確認が必要な場合（安全なコマンド以外）
	if !isSafe {
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("⚙️  Shell Command / シェルコマンド実行\n")
		cyan.Printf("📜 Command / コマンド: %s\n", command)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This command may modify your system / 警告: システムに変更が加わる可能性があります")

		dec := Confirm("Run this command? / 実行しますか？")
		switch dec.Action {
		case ConfirmYes:
			// continue
		case ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for bash.

Comment:
%s

Next actions:
- Revise the command to be safer/smaller and propose again.
- Or split into multiple safe commands.

IMPORTANT: Do NOT execute the previous command as-is.`, strings.TrimSpace(dec.Comment))
		default: // ConfirmNo
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
	if len(result) > config.OutputTruncateLen {
		result = result[:config.OutputTruncateLen] + "\n... (truncated)"
	}

	return result
}
