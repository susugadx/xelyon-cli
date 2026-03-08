package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// printToolArgs はツールの引数を簡潔に表示する（Execute/PreviewToolCallで共通使用）
func printToolArgs(w io.Writer, tc *ToolCall) {
	switch tc.Tool {
	case "read_file":
		_, _ = fmt.Fprintf(w, "   File: %s\n", tc.Args["path"])
	case "write_file":
		lines := strings.Split(tc.Args["content"], "\n")
		_, _ = fmt.Fprintf(w, "   File: %s (%d lines)\n", tc.Args["path"], len(lines))
	case "str_replace":
		_, _ = fmt.Fprintf(w, "   File: %s\n", tc.Args["path"])
	case "bash":
		_, _ = fmt.Fprintf(w, "   Command: %s\n", truncate(tc.Args["command"], 60))
	case "list_dir":
		path := tc.Args["path"]
		if path == "" {
			path = "."
		}
		_, _ = fmt.Fprintf(w, "   Directory: %s\n", path)
	case "git_add", "git_commit", "git_push", "git_status", "git_diff", "git_log",
		"git_branch", "git_checkout", "git_stash":
		// Git操作は引数を簡潔に表示
		for k, v := range tc.Args {
			if v != "" {
				_, _ = fmt.Fprintf(w, "   %s: %s\n", k, truncate(v, 60))
			}
		}
	case "copy_file":
		_, _ = fmt.Fprintf(w, "   Source: %s\n", tc.Args["src"])
		_, _ = fmt.Fprintf(w, "   Destination: %s\n", tc.Args["dest"])
	case "delete_file":
		_, _ = fmt.Fprintf(w, "   File: %s\n", tc.Args["path"])
	case "lint":
		path := tc.Args["path"]
		if path == "" {
			path = "."
		}
		_, _ = fmt.Fprintf(w, "   Path: %s\n", path)
		if tc.Args["auto_fix"] == "true" {
			_, _ = fmt.Fprintf(w, "   Auto-fix: enabled\n")
		}
	case "search_code":
		_, _ = fmt.Fprintf(w, "   Pattern: %s\n", tc.Args["pattern"])
		if tc.Args["path"] != "" {
			_, _ = fmt.Fprintf(w, "   Path: %s\n", tc.Args["path"])
		}
		if tc.Args["file_pattern"] != "" {
			_, _ = fmt.Fprintf(w, "   File pattern: %s\n", tc.Args["file_pattern"])
		}
	case "web_search":
		_, _ = fmt.Fprintf(w, "   Query: %s\n", tc.Args["query"])
	default:
		// その他のツール（MCPツール等）
		if len(tc.Args) > 0 {
			for k, v := range tc.Args {
				_, _ = fmt.Fprintf(w, "   %s: %s\n", k, truncate(v, 60))
			}
		}
	}
	_, _ = fmt.Fprintln(w)
}

// ExecuteWithContext は実行コンテキスト付きでツールを実行する。
func ExecuteWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	execCtx = normalizeExecutionContext(execCtx)
	result, change := executeCoreWithContext(execCtx, tc)

	_, _ = fmt.Fprintln(execCtx.Stdout, ui.FormatToolLine(ui.ToolDisplayInfo{
		ToolName: tc.Tool,
		Args:     tc.Args,
		Result:   result,
		Error:    strings.HasPrefix(strings.TrimSpace(result), "Error:"),
	}))

	// ツール出力の折りたたみ表示（bashはストリーミング表示済みなので除外）
	if !isStreamingTool(tc.Tool) && shouldShowCollapsedOutput(result) {
		displayCollapsedOutput(execCtx.Stdout, result)
	}

	return result, change
}

// ExecuteQuietWithContext は実行コンテキスト付きでツールを実行するが、wrapper 出力を抑制する。
func ExecuteQuietWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	execCtx = normalizeExecutionContext(execCtx)
	restoreQuiet := common.PushQuietMode()
	defer restoreQuiet()
	return executeCoreWithContext(execCtx, tc)
}

func executeCoreWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	// デフォルト値の設定（Registry実行前）
	// list_dir, git_addでpathが空の場合"."を設定
	if tc.Args["path"] == "" {
		switch tc.Tool {
		case "list_dir", "git_add":
			tc.Args["path"] = "."
		}
	}

	// Registry経由でツール実行
	result, change := execCtx.EffectiveRegistry().ExecuteWithContext(execCtx, tc)

	// ファイル変更系ツールの場合、キャッシュを無効化
	invalidateToolCache(execCtx, tc)

	// ツール出力が空の場合は補完
	if strings.TrimSpace(result) == "" {
		result = "(no output)"
	}

	return result, change
}

// invalidateToolCache はファイル変更系ツール実行後にキャッシュを無効化
func invalidateToolCache(execCtx ExecutionContext, tc *ToolCall) {
	cache := execCtx.EffectiveToolCache()
	if cache == nil {
		return
	}

	switch tc.Tool {
	// ファイル内容を変更するツール → ファイルキャッシュ＆検索キャッシュ無効化
	case "write_file", "str_replace", "format", "lint":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				cache.InvalidateFile(absPath)
				cache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// ファイルを削除するツール → ファイル＆ディレクトリ＆検索キャッシュ無効化
	case "delete_file":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				cache.InvalidateFile(absPath)
				cache.InvalidateDir(filepath.Dir(absPath))
				cache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// コピーはコピー先のディレクトリキャッシュ＆検索キャッシュを無効化
	case "copy_file":
		if dest := tc.Args["dest"]; dest != "" {
			if absPath, err := filepath.Abs(dest); err == nil {
				cache.InvalidateDir(filepath.Dir(absPath))
				cache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// ディレクトリ作成は検索結果に影響しないため検索キャッシュはクリアしない
	case "create_dir":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				cache.InvalidateDir(filepath.Dir(absPath))
			}
		}

	// git checkout でファイルが復元される可能性
	case "git_checkout":
		// 全キャッシュクリア（どのファイルが変更されるか分からない）
		cache.Clear()

	// bash: read-only コマンドならキャッシュを保持、それ以外は全クリア
	case "bash":
		if cmd := tc.Args["command"]; !isBashReadOnly(cmd) {
			cache.Clear()
		}
	}
}

// readOnlyCommands は bash 実行後にキャッシュクリアが不要なコマンドのプレフィックス。
// defaultSafeCommands のうちファイルを変更しないもののみ（go mod tidy 等は除外）。
var readOnlyCommands = []string{
	"ls", "cat", "pwd", "echo", "which",
	"head", "tail", "wc", "grep", "rg", "find",
	"sed -n", "diff", "file", "du", "stat",
	"md5sum", "sha256sum",
	"git status", "git log", "git diff", "git branch",
	"git ls-files", "git show", "git remote",
	"go version", "go test", "go vet", "go env",
	"node -v", "npm -v", "npm list",
	"python --version", "pip list",
	"env", "printenv", "date", "uname", "whoami", "id",
	"tree", "less", "more", "sort", "uniq", "cut", "tr",
}

// isBashReadOnly はコマンドが read-only（キャッシュクリア不要）かどうかを判定する。
// パイプ、リダイレクト、連結演算子を含む場合は安全側に倒して false を返す。
func isBashReadOnly(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return true
	}

	// パイプ・リダイレクト・連結を含む場合は unsafe
	for _, ch := range []string{"|", ">", ">>", "&&", ";"} {
		if strings.Contains(trimmed, ch) {
			return false
		}
	}

	// 先頭コマンドが read-only リストにマッチするか
	for _, prefix := range readOnlyCommands {
		if trimmed == prefix {
			return true
		}
		if strings.HasPrefix(trimmed, prefix) && len(trimmed) > len(prefix) {
			next := trimmed[len(prefix)]
			if next == ' ' || next == '\t' {
				return true
			}
		}
	}

	return false
}

// PreviewToolCallWithWriter は指定 writer にツール情報を表示する（実行はしない）。
func PreviewToolCallWithWriter(w io.Writer, tc *ToolCall) {
	if w == nil {
		w = os.Stdout
	}
	color.New(color.FgCyan).Fprintf(w, "🔧 Tool: %s (Dry Run)\n", tc.Tool)
	printToolArgs(w, tc)
}

// PreviewToolCall displays tool information without executing it
func PreviewToolCall(tc *ToolCall) {
	PreviewToolCallWithWriter(os.Stdout, tc)
}

// isStreamingTool はストリーミング出力を行うツールか判定
// これらのツールは実行中にリアルタイム出力するため、折りたたみ表示は不要
func isStreamingTool(toolName string) bool {
	switch toolName {
	case "bash":
		// bashはストリーミング設定時にリアルタイム出力
		return true
	default:
		return false
	}
}

func shouldShowCollapsedOutput(output string) bool {
	return strings.Contains(output, "\n")
}

// displayCollapsedOutput はツール出力を折りたたみ表示
func displayCollapsedOutput(w io.Writer, output string) {
	// エラー出力や短い成功メッセージはそのまま表示
	if strings.HasPrefix(output, "Error:") ||
		strings.HasPrefix(output, "Successfully") ||
		strings.HasPrefix(output, "Cancelled") ||
		strings.HasPrefix(output, "No ") {
		// 1行メッセージはそのまま
		if !strings.Contains(output, "\n") {
			color.New(color.Faint).Fprintf(w, "⎿  %s\n", output)
			return
		}
	}

	// 折りたたみ表示
	formatted := ui.FormatToolOutput(output, ui.GetMaxVisibleLines())
	_, _ = fmt.Fprint(w, formatted)
}
