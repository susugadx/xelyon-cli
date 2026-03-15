package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/dev"
)

const (
	// compactBashMinLen はbash成功時にverifyコマンドを圧縮する最小出力長。
	// この長さ未満の出力はそのまま返す。
	compactBashMinLen = 500

	// compactTailLines は失敗時に残す末尾行数。
	compactTailLines = 30

	// compactCommandMaxLen はサマリー内のコマンド文字列最大長。
	compactCommandMaxLen = 120
)

// compactToolResult はtool実行結果をHistory格納用に圧縮する。
// readTracker によるカウントを先に行い、その後 deduplicateToolResult で重複判定する。
// dedupe 結果にも必要に応じて guidance を追記する。
func (a *Agent) compactToolResult(toolCall *tools.ToolCall, result string) string {
	// 1. read_file の micro-read tracking（dedupe より先にカウントを進める）
	//    同一ファイルの完全重複 read も「read 行動」としてカウントする。
	guidance := a.readTracker.record(toolCall)

	// 2. 重複チェック（元の全文で判定する必要がある）
	deduped := a.deduplicateToolResult(toolCall.Tool, result)
	if deduped != result {
		// 重複参照に差し替えられた場合でも、guidance があれば追記する。
		// dedupe 結果はモデルに返るため、guidance を載せないと soft guard が効かない。
		if guidance != "" {
			deduped = appendGuidanceWithConfirm(a.readTracker, toolCall.Args["path"], deduped, guidance)
		}
		return deduped
	}

	// 3. guidance の追記（dedupe されなかった通常パス）
	if guidance != "" {
		result = appendGuidanceWithConfirm(a.readTracker, toolCall.Args["path"], result, guidance)
	}

	// 4. ツール種別に応じた圧縮
	var compacted string
	switch toolCall.Tool {
	case "bash":
		command := toolCall.Args["command"]
		compacted = compactBash(command, result)
	case "read_file":
		compacted = compactReadFile(toolCall.Args, result)
	case "search_code":
		compacted = compactSearchCode(result)
	case "inspect_symbol":
		compacted = compactInspectSymbol(result)
	case "list_dir":
		compacted = compactListDir(result)
	default:
		compacted = result
	}

	// 5. compaction 効果を記録（1文字でも短くなった場合のみ）
	a.recordCompaction(toolCall.Tool, len(result), len(compacted))

	return compacted
}

// recordCompaction は compaction の効果を ToolObservability に記録する。
func (a *Agent) recordCompaction(toolName string, beforeLen, afterLen int) {
	saved := beforeLen - afterLen
	if saved <= 0 {
		return
	}
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats == nil {
		return
	}
	obs := &a.Stats.ToolObs
	obs.CompactedToolResults++
	obs.CompactedBytesSaved += saved
	if obs.CompactedByTool == nil {
		obs.CompactedByTool = make(map[string]int)
	}
	obs.CompactedByTool[toolName]++
}

// compactBash はbashの実行結果を圧縮する。
func compactBash(command, result string) string {
	if isBashError(result) {
		return tailLines(result, compactTailLines)
	}

	// 成功: 短い出力はそのまま
	if len(result) < compactBashMinLen {
		return result
	}

	// 成功 + 長い出力: verifyコマンドなら1行サマリー
	if dev.IsVerifyCommand(command) {
		return fmt.Sprintf("OK: %s (exit 0)", truncateCommand(command, compactCommandMaxLen))
	}

	// git/CI系の構造化圧縮（閾値ベース）
	if isGitDiff(command) {
		return compactGitDiff(result)
	}
	if isGitLog(command) {
		return compactGitLog(result)
	}
	if isGitShow(command) {
		return compactGitShow(result)
	}
	if isGitBlame(command) {
		return compactGitBlame(result)
	}
	if isGHRunLog(command) {
		return compactGHRunLog(result)
	}

	// ログ系の末尾圧縮
	if isLogCommand(command) {
		return compactLogOutput(result)
	}

	return result
}

// ── ログ系コマンド ──

// logPrefixes はログ出力系コマンドのプレフィックス。
var logPrefixes = []string{
	"kubectl logs",
	"docker logs", "docker compose logs",
	"journalctl",
	"heroku logs",
	"aws logs get",
	"gcloud logging read",
	"az monitor log",
	"stern",
}

// logKeywords はログ出力系コマンドを示すキーワード。
var logKeywords = []string{"logs", "log-events", "logging"}

// isLogCommand はログ出力系コマンドか判定する。
func isLogCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	for _, prefix := range logPrefixes {
		if cmd == prefix || (strings.HasPrefix(cmd, prefix) && len(cmd) > len(prefix) &&
			(cmd[len(prefix)] == ' ' || cmd[len(prefix)] == '\t')) {
			return true
		}
	}
	// キーワードフォールバック
	for _, part := range strings.Fields(cmd) {
		lower := strings.ToLower(part)
		for _, kw := range logKeywords {
			if lower == kw {
				return true
			}
		}
	}
	return false
}

const logTailLines = 50

// compactLogOutput はログ出力を末尾N行に切り詰める。
// ログは最新の末尾が一番重要なため、tailのみ残す。
func compactLogOutput(result string) string {
	lines := strings.Split(result, "\n")
	if len(lines) <= logTailLines {
		return result
	}
	return fmt.Sprintf("[... %d lines total, showing last %d ...]\n%s",
		len(lines), logTailLines, strings.Join(lines[len(lines)-logTailLines:], "\n"))
}

// isBashError はbash実行結果がエラーかを判定する。
// bash.go の出力フォーマットに基づく:
//   - "Error: ..." (exit ≠ 0)
//   - "Command interrupted." (context cancelled)
func isBashError(result string) bool {
	return strings.HasPrefix(result, "Error:") ||
		strings.HasPrefix(result, "Command interrupted.")
}

// tailLines は文字列の末尾n行を返す。全体がn行以下の場合はそのまま返す。
// 切り詰めた場合は先頭に "[... N lines omitted ...]" を付加。
func tailLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	omitted := len(lines) - n
	return fmt.Sprintf("[... %d lines omitted ...]\n%s", omitted, strings.Join(lines[len(lines)-n:], "\n"))
}

// truncateCommand はコマンド文字列を最大maxLen文字に切り詰める。
func truncateCommand(command string, maxLen int) string {
	if len(command) <= maxLen {
		return command
	}
	return command[:maxLen-3] + "..."
}
