package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// 自動実行可能なコマンド（安全なもの）
var defaultSafeCommands = map[string]bool{
	"ls": true, "cat": true, "pwd": true, "echo": true, "which": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"git status": true, "git log": true, "git diff": true, "git branch": true,
	"git ls-files": true, "git show": true, "git remote": true,
	"go version": true, "go mod tidy": true,
	"node -v": true, "npm -v": true, "npm list": true,
	"python --version": true, "pip list": true,
}

// 絶対にブロックするコマンド（どのレベルでも禁止）
var alwaysBlockedCommands = []string{
	"rm -rf /", "rm -rf ~", "rm -rf *",
	"sudo rm", "sudo chmod", "sudo chown",
	"chmod 777", "chmod -R 777",
	"mkfs", "dd if=", ":(){:|:&};:",
	"> /dev/sda", "mv / ",
}

// インライン編集コマンド（permissive以外でブロック）
var inlineEditCommands = []string{
	"sed -i", "sed -e", "sed '",
	"awk -i", "perl -i", "perl -p",
}

// 危険なパイプパターン
var dangerousPipePatterns = []string{
	"| sh", "| bash", "| sudo", "| rm ",
	"| xargs rm", "| xargs sudo",
}

// コマンド連結文字（strict モードでブロック）
var strictSeparators = []string{
	";", "&&", "||", "|", "`", "$(", ">", ">>", "<",
}

// moderate モードでもブロックする連結文字
var moderateSeparators = []string{
	";", "&&", "||", "`", "$(", // パイプ | とリダイレクト >, >>, < は除外
}

// executeBash はシェルコマンドを実行
func executeBash(command string) string {
	if command == "" {
		return "Error: command is empty"
	}

	cfg := config.GetGlobalConfig().Bash

	// 絶対にブロックするコマンド（どのレベルでも）
	for _, blocked := range alwaysBlockedCommands {
		if strings.Contains(command, blocked) {
			red.Printf("🚫 Blocked dangerous command: %s\n", command)
			return "Error: This command is blocked for safety"
		}
	}

	// 安全性レベルに応じたチェック
	if err := checkBashSafety(command, cfg); err != "" {
		return err
	}

	// 安全なコマンドかどうかを判定
	isSafe := isSafeCommand(command, cfg)

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

// checkBashSafety は安全性レベルに応じたチェックを行う
// エラーがあればエラーメッセージを返す、なければ空文字列
func checkBashSafety(command string, cfg config.BashConfig) string {
	level := cfg.SafetyLevel
	if level == "" {
		level = "moderate" // デフォルト
	}

	switch level {
	case "permissive":
		// 最小限のチェックのみ（alwaysBlockedCommandsは既にチェック済み）
		// インライン編集の許可を確認
		if !cfg.AllowInlineEdit {
			for _, inline := range inlineEditCommands {
				if strings.Contains(command, inline) {
					red.Printf("🚫 Inline edit not allowed: %s\n", command)
					yellow.Println("💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable")
					return "Error: Inline edit commands are not allowed"
				}
			}
		}
		// 危険なパイプパターンのみチェック
		for _, pattern := range dangerousPipePatterns {
			if strings.Contains(command, pattern) {
				red.Printf("🚫 Dangerous pipe pattern: %s\n", command)
				return "Error: Dangerous pipe pattern detected"
			}
		}
		return ""

	case "moderate":
		// インライン編集をブロック
		if !cfg.AllowInlineEdit {
			for _, inline := range inlineEditCommands {
				if strings.Contains(command, inline) {
					red.Printf("🚫 Inline edit not allowed: %s\n", command)
					yellow.Println("💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable")
					return "Error: Inline edit commands are not allowed"
				}
			}
		}
		// 危険なパイプパターンをチェック
		for _, pattern := range dangerousPipePatterns {
			if strings.Contains(command, pattern) {
				red.Printf("🚫 Dangerous pipe pattern: %s\n", command)
				return "Error: Dangerous pipe pattern detected"
			}
		}
		// moderate でブロックする連結文字をチェック（安全コマンド以外）
		if !isSafeCommand(command, cfg) {
			for _, sep := range moderateSeparators {
				if strings.Contains(command, sep) {
					red.Printf("🚫 Blocked command injection attempt: %s\n", command)
					yellow.Println("⚠️  Command contains potentially dangerous separator characters.")
					yellow.Println("   Allowed separators in moderate mode: | > >> <")
					return "Error: Command injection attempt detected (separator found)"
				}
			}
			// リダイレクトのチェック
			if !cfg.AllowRedirect {
				for _, redir := range []string{">", ">>", "<"} {
					if strings.Contains(command, redir) {
						red.Printf("🚫 Redirect not allowed: %s\n", command)
						yellow.Println("💡 Tip: Set bash.allow_redirect: true in config.yaml to enable")
						return "Error: Redirect is not allowed"
					}
				}
			}
		}
		return ""

	default: // strict
		// インライン編集をブロック
		for _, inline := range inlineEditCommands {
			if strings.Contains(command, inline) {
				red.Printf("🚫 Inline edit not allowed: %s\n", command)
				yellow.Println("💡 Tip: Set bash.safety_level: moderate in config.yaml")
				return "Error: Inline edit commands are not allowed"
			}
		}
		// すべての連結文字をブロック（安全コマンド以外）
		if !isSafeCommand(command, cfg) {
			for _, sep := range strictSeparators {
				if strings.Contains(command, sep) {
					red.Printf("🚫 Blocked command injection attempt: %s\n", command)
					yellow.Println("⚠️  Command contains potentially dangerous separator characters.")
					yellow.Println("💡 Tip: Set bash.safety_level: moderate in config.yaml to allow pipes")
					return "Error: Command injection attempt detected (separator found)"
				}
			}
		}
		return ""
	}
}

// isSafeCommand は安全なコマンドかどうかを判定
func isSafeCommand(command string, cfg config.BashConfig) bool {
	// デフォルトの安全コマンドをチェック
	for safe := range defaultSafeCommands {
		if strings.HasPrefix(command, safe) {
			return true
		}
	}
	// 設定で追加された安全コマンドをチェック
	for _, safe := range cfg.SafeCommands {
		if strings.HasPrefix(command, safe) {
			return true
		}
	}
	return false
}
