package dev

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// alwaysBlockedCommands は安全性のため常時拒否するコマンド群。
var alwaysBlockedCommands = []string{
	"rm -rf /", "rm -rf ~", "rm -rf *",
	"sudo rm", "sudo chmod", "sudo chown",
	"chmod 777", "chmod -R 777",
	"mkfs", "dd if=", ":(){:|:&};:",
	"> /dev/sda", "mv / ",
	"base64 -d", "base64 --decode",
	"python -c", "python3 -c",
	"node -e", "node --eval",
	"ruby -e",
	"perl -e",
	"LD_PRELOAD=", "LD_LIBRARY_PATH=",
}

// inlineEditCommands は permissive 以外で拒否対象となる inline 編集コマンド。
var inlineEditCommands = []string{
	"sed -i", "sed -e", "sed '",
	"awk -i", "perl -i", "perl -p",
}

// dangerousPipePatterns は safety level に関係なく拒否する危険なパイプパターン。
var dangerousPipePatterns = []string{
	"| sh", "| bash", "| sudo", "| rm ",
	"| xargs rm", "| xargs sudo",
}

// strictSeparators は strict mode で拒否する区切り文字。
var strictSeparators = []string{
	";", "&&", "||", "|", "`", "$(", ">", ">>", "<",
}

// moderateSeparators は moderate mode で拒否する区切り文字。
var moderateSeparators = []string{
	";", "&&", "||", "`", "$(", // pipe | and redirects >, >>, < are excluded
}

// CheckBashSafety は bash の safety_level に基づく安全性チェックを行う。
// 失敗時はエラーメッセージを返し、成功時は空文字を返す。
func CheckBashSafety(command string, cfg config.BashConfig) string {
	return CheckBashSafetyWithOutput(common.DefaultOutput(), command, cfg)
}

// CheckBashSafetyWithOutput は出力先を指定して bash の安全性チェックを行う。
func CheckBashSafetyWithOutput(out common.Output, command string, cfg config.BashConfig) string {
	level := resolveBashSafetyLevel(cfg.SafetyLevel)

	switch level {
	case "permissive":
		if err := checkInlineEditPermission(out, command, cfg.AllowInlineEdit, "💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable"); err != "" {
			return err
		}
		if err := checkDangerousPipePatterns(out, command); err != "" {
			return err
		}
		return ""

	case "moderate":
		if err := checkInlineEditPermission(out, command, cfg.AllowInlineEdit, "💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable"); err != "" {
			return err
		}
		if err := checkDangerousPipePatterns(out, command); err != "" {
			return err
		}
		if err := checkModerateSeparatorsAndRedirect(out, command, cfg); err != "" {
			return err
		}
		return ""

	default: // strict
		if err := checkInlineEditPermission(out, command, false, "💡 Tip: Set bash.safety_level: moderate in config.yaml"); err != "" {
			return err
		}
		if err := checkStrictSeparators(out, command); err != "" {
			return err
		}
		return ""
	}
}

func resolveBashSafetyLevel(level string) string {
	if level == "" {
		return "moderate"
	}
	return level
}

func checkInlineEditPermission(out common.Output, command string, allowInlineEdit bool, tip string) string {
	if allowInlineEdit {
		return ""
	}
	for _, inline := range inlineEditCommands {
		if strings.Contains(command, inline) {
			out.Red.Printf("🚫 Inline edit not allowed: %s\n", command)
			out.Yellow.Println(tip)
			return "Error: Inline edit commands are not allowed"
		}
	}
	return ""
}

func checkDangerousPipePatterns(out common.Output, command string) string {
	for _, pattern := range dangerousPipePatterns {
		if strings.Contains(command, pattern) {
			out.Red.Printf("🚫 Dangerous pipe pattern: %s\n", command)
			return "Error: Dangerous pipe pattern detected"
		}
	}
	return ""
}

func checkModerateSeparatorsAndRedirect(out common.Output, command string, cfg config.BashConfig) string {
	if IsSafeCommand(command, cfg) {
		return ""
	}
	for _, sep := range moderateSeparators {
		if strings.Contains(command, sep) {
			out.Red.Printf("🚫 Blocked command injection attempt: %s\n", command)
			out.Yellow.Println("⚠️  Command contains potentially dangerous separator characters.")
			out.Yellow.Println("   Allowed separators in moderate mode: | > >> <")
			return "Error: Command injection attempt detected (separator found)"
		}
	}
	if cfg.AllowRedirect {
		return ""
	}
	for _, redir := range []string{">", ">>", "<"} {
		if strings.Contains(command, redir) {
			out.Red.Printf("🚫 Redirect not allowed: %s\n", command)
			out.Yellow.Println("💡 Tip: Set bash.allow_redirect: true in config.yaml to enable")
			return "Error: Redirect is not allowed"
		}
	}
	return ""
}

func checkStrictSeparators(out common.Output, command string) string {
	for _, sep := range strictSeparators {
		if strings.Contains(command, sep) {
			out.Red.Printf("🚫 Blocked command injection attempt: %s\n", command)
			out.Yellow.Println("⚠️  Command contains potentially dangerous separator characters.")
			out.Yellow.Println("💡 Tip: Set bash.safety_level: moderate in config.yaml to allow pipes")
			return "Error: Command injection attempt detected (separator found)"
		}
	}
	return ""
}
