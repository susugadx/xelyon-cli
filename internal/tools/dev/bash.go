package dev

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// Colors from common package
var (
	cyan   = common.Cyan
	green  = common.Green
	yellow = common.Yellow
	red    = common.Red
)

// Auto-executable safe commands
var defaultSafeCommands = map[string]bool{
	"ls": true, "cat": true, "pwd": true, "echo": true, "which": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"git status": true, "git log": true, "git diff": true, "git branch": true,
	"git ls-files": true, "git show": true, "git remote": true,
	"go version": true, "go mod tidy": true,
	"node -v": true, "npm -v": true, "npm list": true,
	"python --version": true, "pip list": true,
}

// Always blocked commands (blocked at all levels)
var alwaysBlockedCommands = []string{
	"rm -rf /", "rm -rf ~", "rm -rf *",
	"sudo rm", "sudo chmod", "sudo chown",
	"chmod 777", "chmod -R 777",
	"mkfs", "dd if=", ":(){:|:&};:",
	"> /dev/sda", "mv / ",
}

// Inline edit commands (blocked except permissive)
var inlineEditCommands = []string{
	"sed -i", "sed -e", "sed '",
	"awk -i", "perl -i", "perl -p",
}

// Dangerous pipe patterns
var dangerousPipePatterns = []string{
	"| sh", "| bash", "| sudo", "| rm ",
	"| xargs rm", "| xargs sudo",
}

// Command separator characters (blocked in strict mode)
var strictSeparators = []string{
	";", "&&", "||", "|", "`", "$(", ">", ">>", "<",
}

// Separator characters blocked in moderate mode
var moderateSeparators = []string{
	";", "&&", "||", "`", "$(", // pipe | and redirects >, >>, < are excluded
}

// ExecuteBash executes a shell command
func ExecuteBash(command string) string {
	if command == "" {
		return "Error: command is empty"
	}

	cfg := config.GetGlobalConfig().Bash

	// Always blocked commands (at all levels)
	for _, blocked := range alwaysBlockedCommands {
		if strings.Contains(command, blocked) {
			red.Printf("🚫 Blocked dangerous command: %s\n", command)
			return "Error: This command is blocked for safety"
		}
	}

	// Safety level checks
	if err := CheckBashSafety(command, cfg); err != "" {
		return err
	}

	// Determine if command is safe
	isSafe := IsSafeCommand(command, cfg)

	// Require confirmation for non-safe commands
	if !isSafe {
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("⚙️  Shell Command / シェルコマンド実行\n")
		cyan.Printf("📜 Command / コマンド: %s\n", command)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This command may modify your system / 警告: システムに変更が加わる可能性があります")

		dec := common.Confirm("Run this command? / 実行しますか？")
		switch dec.Action {
		case common.ConfirmYes:
			// continue
		case common.ConfirmComment:
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

	// Execute
	green.Printf("▶ Running: %s\n", command)
	cmd := exec.Command("bash", "-c", command)
	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	} else {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cmd.Dir = "." // fallback
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

// CheckBashSafety performs safety level checks
// Returns error message if check fails, empty string otherwise
func CheckBashSafety(command string, cfg config.BashConfig) string {
	level := cfg.SafetyLevel
	if level == "" {
		level = "moderate" // default
	}

	switch level {
	case "permissive":
		// Minimal checks only (alwaysBlockedCommands already checked)
		// Check inline edit permission
		if !cfg.AllowInlineEdit {
			for _, inline := range inlineEditCommands {
				if strings.Contains(command, inline) {
					red.Printf("🚫 Inline edit not allowed: %s\n", command)
					yellow.Println("💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable")
					return "Error: Inline edit commands are not allowed"
				}
			}
		}
		// Check dangerous pipe patterns only
		for _, pattern := range dangerousPipePatterns {
			if strings.Contains(command, pattern) {
				red.Printf("🚫 Dangerous pipe pattern: %s\n", command)
				return "Error: Dangerous pipe pattern detected"
			}
		}
		return ""

	case "moderate":
		// Block inline edits
		if !cfg.AllowInlineEdit {
			for _, inline := range inlineEditCommands {
				if strings.Contains(command, inline) {
					red.Printf("🚫 Inline edit not allowed: %s\n", command)
					yellow.Println("💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable")
					return "Error: Inline edit commands are not allowed"
				}
			}
		}
		// Check dangerous pipe patterns
		for _, pattern := range dangerousPipePatterns {
			if strings.Contains(command, pattern) {
				red.Printf("🚫 Dangerous pipe pattern: %s\n", command)
				return "Error: Dangerous pipe pattern detected"
			}
		}
		// Check moderate separators (except safe commands)
		if !IsSafeCommand(command, cfg) {
			for _, sep := range moderateSeparators {
				if strings.Contains(command, sep) {
					red.Printf("🚫 Blocked command injection attempt: %s\n", command)
					yellow.Println("⚠️  Command contains potentially dangerous separator characters.")
					yellow.Println("   Allowed separators in moderate mode: | > >> <")
					return "Error: Command injection attempt detected (separator found)"
				}
			}
			// Check redirects
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
		// Block inline edits
		for _, inline := range inlineEditCommands {
			if strings.Contains(command, inline) {
				red.Printf("🚫 Inline edit not allowed: %s\n", command)
				yellow.Println("💡 Tip: Set bash.safety_level: moderate in config.yaml")
				return "Error: Inline edit commands are not allowed"
			}
		}
		// Block all separators (except safe commands)
		if !IsSafeCommand(command, cfg) {
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

// IsSafeCommand checks if a command is safe
func IsSafeCommand(command string, cfg config.BashConfig) bool {
	// Check default safe commands
	for safe := range defaultSafeCommands {
		if strings.HasPrefix(command, safe) {
			return true
		}
	}
	// Check custom safe commands from config
	for _, safe := range cfg.SafeCommands {
		if strings.HasPrefix(command, safe) {
			return true
		}
	}
	return false
}
