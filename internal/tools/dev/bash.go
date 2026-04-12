package dev

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// verificationSafeCommands はビルド・テスト・検証・環境確認用の安全コマンド。
// コード探索系（grep, find, cat, head, tail, sed -n 等）は含めない。
// 探索は gather_context を第一選択にし、必要時のみ低レベル investigation tool を使う。
var verificationSafeCommands = map[string]bool{
	// 環境確認
	"pwd": true, "echo": true, "which": true, "env": true, "printenv": true,
	// ファイル情報（内容読み取りではない）
	// ls は探索用途で多用されるため verification safe から除外（discovery 寄り扱い）
	"diff": true, "file": true, "du": true, "stat": true,
	"md5sum": true, "sha256sum": true,
	// Git 状態確認
	"git status": true, "git log": true, "git diff": true, "git branch": true,
	"git ls-files": true, "git show": true, "git remote": true,
	// ビルド・テスト・lint・format
	// NOTE: go run / npm install / pip install / docker run は任意コード実行が可能なため除外
	"go version": true, "go build": true, "go test": true, "go vet": true,
	"go fmt": true, "go mod tidy": true, "go mod download": true,
	"go generate": true,
	"make":        true,
	"npm test":    true, "npm run": true,
	"cargo build": true, "cargo test": true, "cargo check": true, "cargo fmt": true,
	"pytest": true, "python -m pytest": true, "python -m unittest": true,
	"pip list": true,
	"node -v":  true, "npm -v": true,
	"python --version": true, "python3 --version": true,
	"rustc --version": true, "cargo --version": true,
	"java -version": true, "javac -version": true,
	"golangci-lint": true, "eslint": true, "prettier": true,
	// Docker（状態確認のみ、docker run は任意コマンド実行可能なため除外）
	"docker ps": true, "docker images": true,
}

// Always blocked commands (blocked at all levels)
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
	return ExecuteBashWithOutput(common.DefaultOutput(), command)
}

// ExecuteBashWithOutput executes a shell command with explicit output writers.
func ExecuteBashWithOutput(out common.Output, command string) string {
	return ExecuteBashWithPromptIOAndConfig(ui.NewPromptIO(nil, out.StdoutWriter(), out.StderrWriter(), nil), config.DefaultConfig(), command)
}

// ExecuteBashWithPromptIOAndConfig executes a shell command with explicit config.
func ExecuteBashWithPromptIOAndConfig(promptIO ui.PromptIO, cfg *config.Config, command string) string {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if command == "" {
		return "Error: command is empty"
	}

	reason, msg, ok := checkAndConfirmBash(promptIO, cfg, command)
	if !ok {
		return msg
	}

	// Execute — auto-approved 理由を Running 表示に統合
	printRunning(out, command, reason)
	cmd := exec.Command("bash", "-c", command)
	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	} else {
		out.Yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cmd.Dir = "." // fallback
	}

	// ストリーミング設定を確認
	streamOutput := cfg.Streaming.StreamBashOutput

	var result string
	var cmdErr error

	if streamOutput {
		// ストリーミングモード: リアルタイム出力
		result, cmdErr = executeBashWithStreaming(out, cmd)
	} else {
		// 従来モード: バッファリング出力
		output, err := cmd.CombinedOutput()
		result = string(output)
		cmdErr = err
	}

	if cmdErr != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", cmdErr, result)
	}

	if strings.TrimSpace(result) == "" {
		result = "(no output)"
	}

	result = TruncateWithFile(result)

	return result
}

// ExecuteBashWithContext はContext対応でシェルコマンドを実行
// コンテキストがキャンセルされた場合、部分結果を返す
func ExecuteBashWithContext(ctx context.Context, command string) string {
	return ExecuteBashWithContextAndOutput(ctx, common.DefaultOutput(), command)
}

// ExecuteBashWithContextAndOutput はContext対応でシェルコマンドを実行する。
func ExecuteBashWithContextAndOutput(ctx context.Context, out common.Output, command string) string {
	return ExecuteBashWithContextAndPromptIOAndConfig(ctx, ui.NewPromptIO(nil, out.StdoutWriter(), out.StderrWriter(), nil), config.DefaultConfig(), command)
}

// ExecuteBashWithContextAndPromptIOAndConfig はContext対応でシェルコマンドを実行する。
func ExecuteBashWithContextAndPromptIOAndConfig(ctx context.Context, promptIO ui.PromptIO, cfg *config.Config, command string) string {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if command == "" {
		return "Error: command is empty"
	}

	reason, msg, ok := checkAndConfirmBash(promptIO, cfg, command)
	if !ok {
		return msg
	}

	// Execute with context — auto-approved 理由を Running 表示に統合
	printRunning(out, command, reason)
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	} else {
		out.Yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cmd.Dir = "."
	}

	// プロセスグループを設定（子プロセスも一緒に終了させるため）
	setSysProcAttr(cmd)

	// ストリーミング設定を確認
	streamOutput := cfg.Streaming.StreamBashOutput

	var result string
	var cmdErr error

	if streamOutput {
		result, cmdErr = executeBashWithStreamingAndContext(ctx, out, cmd)
	} else {
		output, err := cmd.CombinedOutput()
		result = string(output)
		cmdErr = err
	}

	// コンテキストキャンセルの場合は部分結果を返す
	if ctx.Err() != nil {
		out.Yellow.Println("\n⚠️  Command interrupted. Partial output returned.")
		// プロセスグループ全体を終了
		killProcessGroup(cmd)
		return fmt.Sprintf("Command interrupted.\nPartial output:\n%s", result)
	}

	if cmdErr != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", cmdErr, result)
	}

	if strings.TrimSpace(result) == "" {
		result = "(no output)"
	}

	result = TruncateWithFile(result)

	return result
}

// executeBashWithStreamingAndContext はContext対応でストリーミング出力
func executeBashWithStreamingAndContext(ctx context.Context, out common.Output, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	var outputBuf strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup
	doneCh := make(chan struct{})

	// stdout ストリーミング
	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutputWithContext(ctx, out, stdout, &outputBuf, &mu, false)
	}()

	// stderr ストリーミング
	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutputWithContext(ctx, out, stderr, &outputBuf, &mu, true)
	}()

	// 完了を待機
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		// コンテキストキャンセル - プロセスグループを終了
		killProcessGroup(cmd)
		return outputBuf.String(), ctx.Err()
	case <-doneCh:
		err := cmd.Wait()
		return outputBuf.String(), err
	}
}

// streamOutputWithContext はContext対応でストリーミング出力
func streamOutputWithContext(ctx context.Context, out common.Output, pipe io.Reader, buf *strings.Builder, mu *sync.Mutex, isStderr bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		if isStderr {
			out.Red.Println(line)
		} else {
			out.Println(line)
		}

		mu.Lock()
		buf.WriteString(line + "\n")
		mu.Unlock()
	}
}

// executeBashWithStreaming はコマンド出力をリアルタイムでストリーミング
func executeBashWithStreaming(out common.Output, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	var outputBuf strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup

	// stdout ストリーミング
	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutput(out, stdout, &outputBuf, &mu, false)
	}()

	// stderr ストリーミング (赤色で表示)
	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutput(out, stderr, &outputBuf, &mu, true)
	}()

	wg.Wait()
	err = cmd.Wait()

	return outputBuf.String(), err
}

// printRunning は bash 実行開始メッセージを表示する。
// auto-approved の場合は理由を統合して 1 行で表示。
// ユーザー確認済み（reason が空）の場合は Running のみ。
func printRunning(out common.Output, command string, reason AutoApproveReason) {
	if reason != "" {
		out.Green.Printf("▶ Running (auto-approved: %s): %s\n", reason, command)
	} else {
		out.Green.Printf("▶ Running: %s\n", command)
	}
}

// AutoApproveReason は bash 自動承認時の理由ラベル
type AutoApproveReason string

const (
	approveNone             AutoApproveReason = ""
	approveVerificationBash AutoApproveReason = "Safe bash"
	approveSafeShellCmd     AutoApproveReason = "Safe bash"
	approveSafeBuiltin      AutoApproveReason = "Safe bash"
)

// checkAndConfirmBash は共通のセキュリティチェック + 確認UIを実行。
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

	// 1. 常時ブロック対象
	for _, blocked := range alwaysBlockedCommands {
		if strings.Contains(command, blocked) {
			out.Red.Printf("🚫 Blocked dangerous command: %s\n", command)
			return approveNone, "Error: This command is blocked for safety", false
		}
	}

	// eval コマンドをブロック（チェーン内も対象）
	for _, part := range splitChainCommand(command) {
		if isEvalInvocation(strings.TrimSpace(part)) {
			out.Red.Printf("🚫 Blocked dangerous command: eval\n")
			return approveNone, "Error: eval is blocked for safety", false
		}
	}

	// 2. safety level チェック（strict/moderate/permissive）
	if err := CheckBashSafetyWithOutput(out, command, bashCfg); err != "" {
		return approveNone, err, false
	}

	// === 判定順序 ===
	// 3. 分類（discovery / destructive / redirect-write / verification）
	shellCat := config.ClassifyShellCommand(command)

	// 4. discovery bash — 全 mode で原則自動許可しない（safe_shell_commands でも bypass 不可）
	if shellCat == config.ShellDiscovery && !policy.AllowDiscoveryBash {
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Cyan.Printf("🔍 Discovery Shell / 探索系シェルコマンド\n")
		out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Yellow.Println("💡 Tip: Use gather_context first for code exploration; fall back only to the low-level investigation tools that are actually visible in the current surface when exact control is required")
		reason, msg, ok := confirmBashPromptWithReason(promptIO, command)
		return reason, msg, ok
	}

	// 5. always_confirm / 強制確認（mode より優先、safe_shell_commands で bypass 不可）
	if shellCat == config.ShellDestructive && policy.ShouldConfirm(config.ConfirmBashDestructive) {
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Cyan.Printf("⚠️  Destructive Shell / 破壊的シェルコマンド\n")
		out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		reason, msg, ok := confirmBashPromptWithReason(promptIO, command)
		return reason, msg, ok
	}
	if shellCat == config.ShellRedirectWrite && policy.ShouldConfirm(config.ConfirmBashRedirectWrite) {
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Cyan.Printf("📝 Redirect Write Shell / リダイレクト書き込み\n")
		out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		reason, msg, ok := confirmBashPromptWithReason(promptIO, command)
		return reason, msg, ok
	}

	// 6. mode による自動承認（verification 系のみ）
	if shellCat == config.ShellVerification {
		if IsSafeCommand(command, bashCfg) && policy.AutoApproveVerificationBash {
			return approveVerificationBash, "", true
		}
	}

	// 7. safe_shell_commands は verification 系の escape hatch のみ。
	// destructive / redirect-write / discovery には適用しない。
	if shellCat == config.ShellVerification && isSafeShellCommand(command, policy.SafeShellCommands) {
		return approveSafeShellCmd, "", true
	}

	// 8. 組み込み safe command（verification 系のみ、上で mode チェック済みでなくても確認なしで通す）
	if IsSafeCommand(command, bashCfg) {
		return approveSafeBuiltin, "", true
	}

	// 9. その他 — 確認プロンプト
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Cyan.Printf("⚙️  Shell Command / シェルコマンド実行\n")
	out.Cyan.Printf("📜 Command / コマンド: %s\n", command)
	out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	out.Yellow.Println("⚠️  Warning: This command may modify your system / 警告: システムに変更が加わる可能性があります")
	reason, msg, ok := confirmBashPromptWithReason(promptIO, command)
	return reason, msg, ok
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

// streamOutput はパイプからの出力をリアルタイムで表示しバッファに保存
func streamOutput(out common.Output, pipe io.Reader, buf *strings.Builder, mu *sync.Mutex, isStderr bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()

		// リアルタイム表示
		if isStderr {
			out.Red.Println(line)
		} else {
			out.Println(line)
		}

		// バッファに保存
		mu.Lock()
		buf.WriteString(line + "\n")
		mu.Unlock()
	}
}

// CheckBashSafety performs safety level checks
// Returns error message if check fails, empty string otherwise
func CheckBashSafety(command string, cfg config.BashConfig) string {
	return CheckBashSafetyWithOutput(common.DefaultOutput(), command, cfg)
}

// CheckBashSafetyWithOutput performs safety checks with explicit output writers.
func CheckBashSafetyWithOutput(out common.Output, command string, cfg config.BashConfig) string {
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
					out.Red.Printf("🚫 Inline edit not allowed: %s\n", command)
					out.Yellow.Println("💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable")
					return "Error: Inline edit commands are not allowed"
				}
			}
		}
		// Check dangerous pipe patterns only
		for _, pattern := range dangerousPipePatterns {
			if strings.Contains(command, pattern) {
				out.Red.Printf("🚫 Dangerous pipe pattern: %s\n", command)
				return "Error: Dangerous pipe pattern detected"
			}
		}
		return ""

	case "moderate":
		// Block inline edits
		if !cfg.AllowInlineEdit {
			for _, inline := range inlineEditCommands {
				if strings.Contains(command, inline) {
					out.Red.Printf("🚫 Inline edit not allowed: %s\n", command)
					out.Yellow.Println("💡 Tip: Set bash.allow_inline_edit: true in config.yaml to enable")
					return "Error: Inline edit commands are not allowed"
				}
			}
		}
		// Check dangerous pipe patterns
		for _, pattern := range dangerousPipePatterns {
			if strings.Contains(command, pattern) {
				out.Red.Printf("🚫 Dangerous pipe pattern: %s\n", command)
				return "Error: Dangerous pipe pattern detected"
			}
		}
		// Check moderate separators (except safe commands)
		if !IsSafeCommand(command, cfg) {
			for _, sep := range moderateSeparators {
				if strings.Contains(command, sep) {
					out.Red.Printf("🚫 Blocked command injection attempt: %s\n", command)
					out.Yellow.Println("⚠️  Command contains potentially dangerous separator characters.")
					out.Yellow.Println("   Allowed separators in moderate mode: | > >> <")
					return "Error: Command injection attempt detected (separator found)"
				}
			}
			// Check redirects
			if !cfg.AllowRedirect {
				for _, redir := range []string{">", ">>", "<"} {
					if strings.Contains(command, redir) {
						out.Red.Printf("🚫 Redirect not allowed: %s\n", command)
						out.Yellow.Println("💡 Tip: Set bash.allow_redirect: true in config.yaml to enable")
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
				out.Red.Printf("🚫 Inline edit not allowed: %s\n", command)
				out.Yellow.Println("💡 Tip: Set bash.safety_level: moderate in config.yaml")
				return "Error: Inline edit commands are not allowed"
			}
		}
		// strict: セパレータを含むコマンドは safe command でも一律ブロック
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
}

// IsSafeCommand はコマンドが自動実行可能な安全コマンドかを判定する。
// verification 系のみ自動許可。discovery 系は safe_commands に明示追加されていても
// verification として扱う（探索用途の bypass にならない）。
// チェーンコマンド（&&, ||, ;）は全パーツが安全な場合のみ true を返す。
func IsSafeCommand(command string, cfg config.BashConfig) bool {
	parts := splitChainCommand(command)
	for _, part := range parts {
		if !isSingleCommandSafe(part, cfg) {
			return false
		}
	}
	return true
}

// splitChainCommand はコマンドを &&, ||, ; で分割する
// パイプ | は分割しない（パイプチェーンは別途 dangerousPipePatterns でチェック）
func splitChainCommand(command string) []string {
	var parts []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	i := 0
	for i < len(command) {
		ch := command[i]
		// クォート状態の追跡
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteByte(ch)
			i++
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteByte(ch)
			i++
			continue
		}
		// クォート内ならそのまま追加
		if inSingle || inDouble {
			current.WriteByte(ch)
			i++
			continue
		}
		// セパレータチェック（クォート外のみ）
		if i+1 < len(command) && command[i:i+2] == "&&" {
			if part := strings.TrimSpace(current.String()); part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			i += 2
			continue
		}
		if i+1 < len(command) && command[i:i+2] == "||" {
			if part := strings.TrimSpace(current.String()); part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			i += 2
			continue
		}
		if ch == ';' {
			if part := strings.TrimSpace(current.String()); part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			i++
			continue
		}
		current.WriteByte(ch)
		i++
	}
	if part := strings.TrimSpace(current.String()); part != "" {
		parts = append(parts, part)
	}
	return parts
}

// isSingleCommandSafe は単一コマンド（チェーンなし）が安全かチェックする。
// verification 系のみ true を返す。discovery 系は false。
func isSingleCommandSafe(command string, cfg config.BashConfig) bool {
	trimmed := strings.TrimSpace(command)

	// verification 系の組み込み安全コマンド
	for safe := range verificationSafeCommands {
		if matchCommandPrefix(trimmed, safe) {
			return true
		}
	}
	// ユーザー設定の safe_shell_commands / bash.safe_commands
	for _, safe := range cfg.SafeCommands {
		if matchCommandPrefix(trimmed, safe) {
			return true
		}
	}
	return false
}

// matchCommandPrefix はコマンドが安全なプレフィックスで始まるか判定する
// プレフィックスの直後が空白・タブ・EOFのいずれかであることを確認
func matchCommandPrefix(command, prefix string) bool {
	if command == prefix {
		return true
	}
	if strings.HasPrefix(command, prefix) {
		next := command[len(prefix)]
		return next == ' ' || next == '\t'
	}
	return false
}

func isEvalInvocation(command string) bool {
	if !strings.HasPrefix(command, "eval") {
		return false
	}
	if len(command) == len("eval") {
		return true
	}
	next := command[len("eval")]
	// "evaluate" などの英数字連結は除外し、記号開始は eval 呼び出しとして扱う
	isAlphaNumOrUnderscore := (next >= 'a' && next <= 'z') ||
		(next >= 'A' && next <= 'Z') ||
		(next >= '0' && next <= '9') ||
		next == '_'
	return !isAlphaNumOrUnderscore
}
