package dev

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// checkAndConfirmBash は共通のセキュリティチェック + 確認 UI を実行する。
// execution policy に基づき discovery bash の抑止・verification bash の自動承認を行う。
// 返り値: (reason, "", true) = 自動承認（reason は理由）, ("", errorMsg, false) = ブロック/キャンセル
func checkAndConfirmBash(promptIO ui.PromptIO, cfg *config.Config, command string) (AutoApproveReason, string, bool) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	bashCfg := cfg.Bash
	policy := config.ResolveExecutionPolicy(cfg.Execution)

	if err := blockAlwaysDangerousCommand(out, command); err != "" {
		return approveNone, err, false
	}
	if err := blockEvalInvocation(out, command); err != "" {
		return approveNone, err, false
	}

	if err := CheckBashSafetyWithOutput(out, command, bashCfg); err != "" {
		return approveNone, err, false
	}

	shellCat := config.ClassifyShellCommand(command)

	if shellCat == config.ShellDiscovery && !policy.AllowDiscoveryBash {
		printDiscoveryShellPrompt(out, command)
		return confirmBashPromptWithReason(promptIO, command)
	}

	if shellCat == config.ShellDestructive && policy.ShouldConfirm(config.ConfirmBashDestructive) {
		printDestructiveShellPrompt(out, command)
		return confirmBashPromptWithReason(promptIO, command)
	}
	if shellCat == config.ShellRedirectWrite && policy.ShouldConfirm(config.ConfirmBashRedirectWrite) {
		printRedirectWriteShellPrompt(out, command)
		return confirmBashPromptWithReason(promptIO, command)
	}

	if shellCat == config.ShellVerification {
		if IsSafeCommand(command, bashCfg) && policy.AutoApproveVerificationBash {
			return approveVerificationBash, "", true
		}
	}

	if shellCat == config.ShellVerification && isSafeShellCommand(command, policy.SafeShellCommands) {
		return approveSafeShellCmd, "", true
	}

	if IsSafeCommand(command, bashCfg) {
		return approveSafeBuiltin, "", true
	}

	printGenericShellPrompt(out, command)
	return confirmBashPromptWithReason(promptIO, command)
}

func blockAlwaysDangerousCommand(out common.Output, command string) string {
	for _, blocked := range alwaysBlockedCommands {
		if strings.Contains(command, blocked) {
			out.Red.Printf("🚫 Blocked dangerous command: %s\n", command)
			return "Error: This command is blocked for safety"
		}
	}
	return ""
}

func blockEvalInvocation(out common.Output, command string) string {
	for _, part := range splitChainCommand(command) {
		if isEvalInvocation(strings.TrimSpace(part)) {
			out.Red.Printf("🚫 Blocked dangerous command: eval\n")
			return "Error: eval is blocked for safety"
		}
	}
	return ""
}

func printDiscoveryShellPrompt(out common.Output, command string) {
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Cyan.Printf("🔍 Discovery Shell / 探索系シェルコマンド\n")
	out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Yellow.Println("💡 Tip: Use gather_context first for code exploration; fall back only to the low-level investigation tools that are actually visible in the current surface when exact control is required")
}

func printDestructiveShellPrompt(out common.Output, command string) {
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Cyan.Printf("⚠️  Destructive Shell / 破壊的シェルコマンド\n")
	out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func printRedirectWriteShellPrompt(out common.Output, command string) {
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Cyan.Printf("📝 Redirect Write Shell / リダイレクト書き込み\n")
	out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func printGenericShellPrompt(out common.Output, command string) {
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Cyan.Printf("⚙️  Shell Command / シェルコマンド実行\n")
	out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Yellow.Println("⚠️  Warning: This command may modify your system / 警告: システムに変更が加わる可能性があります")
}

// confirmBashPromptWithReason は bash の確認プロンプトを表示する。
// ユーザーが承認した場合は approveNone（ユーザー確認済み）を返す。
func confirmBashPromptWithReason(promptIO ui.PromptIO, command string) (AutoApproveReason, string, bool) {
	dec := common.ConfirmWithIO(promptIO, "Run this command? / 実行しますか？")
	switch dec.Action {
	case common.ConfirmYes:
		return approveNone, "", true
	case common.ConfirmComment:
		return approveNone, fmt.Sprintf(`[COMMENT] User provided feedback for bash.

Comment:
%s

Next actions:
- Revise the command to be safer/smaller and propose again.
- Or split into multiple safe commands.

IMPORTANT: Do NOT execute the previous command as-is.`, strings.TrimSpace(dec.Comment)), false
	default:
		return approveNone, "Cancelled by user", false
	}
}

// isSafeShellCommand は safe_shell_commands に含まれるか判定する。
func isSafeShellCommand(command string, safeCommands []string) bool {
	trimmed := strings.TrimSpace(command)
	for _, safe := range safeCommands {
		if matchCommandPrefix(trimmed, safe) {
			return true
		}
	}
	return false
}
