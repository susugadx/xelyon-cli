package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

// printToolArgs はツールの引数を簡潔に表示する（Execute/PreviewToolCallで共通使用）
func printToolArgs(tc *ToolCall) {
	switch tc.Tool {
	case "read_file":
		fmt.Printf("   File: %s\n", tc.Args["path"])
	case "write_file":
		lines := strings.Split(tc.Args["content"], "\n")
		fmt.Printf("   File: %s (%d lines)\n", tc.Args["path"], len(lines))
	case "str_replace":
		fmt.Printf("   File: %s\n", tc.Args["path"])
	case "bash":
		fmt.Printf("   Command: %s\n", truncate(tc.Args["command"], 60))
	case "list_dir":
		path := tc.Args["path"]
		if path == "" {
			path = "."
		}
		fmt.Printf("   Directory: %s\n", path)
	case "git_add", "git_commit", "git_push", "git_status", "git_diff", "git_log",
		"git_branch", "git_checkout", "git_stash":
		// Git操作は引数を簡潔に表示
		for k, v := range tc.Args {
			if v != "" {
				fmt.Printf("   %s: %s\n", k, truncate(v, 60))
			}
		}
	case "copy_file":
		fmt.Printf("   Source: %s\n", tc.Args["src"])
		fmt.Printf("   Destination: %s\n", tc.Args["dest"])
	case "delete_file":
		fmt.Printf("   File: %s\n", tc.Args["path"])
	case "lint":
		path := tc.Args["path"]
		if path == "" {
			path = "."
		}
		fmt.Printf("   Path: %s\n", path)
		if tc.Args["auto_fix"] == "true" {
			fmt.Printf("   Auto-fix: enabled\n")
		}
	case "search_code":
		fmt.Printf("   Pattern: %s\n", tc.Args["pattern"])
		if tc.Args["path"] != "" {
			fmt.Printf("   Path: %s\n", tc.Args["path"])
		}
		if tc.Args["file_pattern"] != "" {
			fmt.Printf("   File pattern: %s\n", tc.Args["file_pattern"])
		}
	case "web_search":
		fmt.Printf("   Query: %s\n", tc.Args["query"])
	default:
		// その他のツール（MCPツール等）
		if len(tc.Args) > 0 {
			for k, v := range tc.Args {
				fmt.Printf("   %s: %s\n", k, truncate(v, 60))
			}
		}
	}
	fmt.Println()
}

// Execute はツールを実行（Registry経由）
func Execute(tc *ToolCall) (string, *FileChange) {
	cyan.Printf("🔧 Tool: %s\n", tc.Tool)
	printToolArgs(tc)

	// デフォルト値の設定（Registry実行前）
	// list_dir, git_addでpathが空の場合"."を設定
	if tc.Args["path"] == "" {
		switch tc.Tool {
		case "list_dir", "git_add":
			tc.Args["path"] = "."
		}
	}

	// Registry経由でツール実行
	result, change := DefaultRegistry.Execute(tc)

	// ファイル変更系ツールの場合、キャッシュを無効化
	invalidateToolCache(tc)

	// ツール出力が空の場合は補完
	if strings.TrimSpace(result) == "" {
		result = "(no output)"
	}

	// ツール出力の折りたたみ表示（bashはストリーミング表示済みなので除外）
	if !isStreamingTool(tc.Tool) && result != "" {
		displayCollapsedOutput(result)
	}

	return result, change
}

// invalidateToolCache はファイル変更系ツール実行後にキャッシュを無効化
func invalidateToolCache(tc *ToolCall) {
	if GlobalToolCache == nil {
		return
	}

	switch tc.Tool {
	// ファイル内容を変更するツール → ファイルキャッシュ＆検索キャッシュ無効化
	case "write_file", "str_replace", "format", "lint":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				GlobalToolCache.InvalidateFile(absPath)
				GlobalToolCache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// ファイルを削除するツール → ファイル＆ディレクトリ＆検索キャッシュ無効化
	case "delete_file":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				GlobalToolCache.InvalidateFile(absPath)
				GlobalToolCache.InvalidateDir(filepath.Dir(absPath))
				GlobalToolCache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// コピーはコピー先のディレクトリキャッシュ＆検索キャッシュを無効化
	case "copy_file":
		if dest := tc.Args["dest"]; dest != "" {
			if absPath, err := filepath.Abs(dest); err == nil {
				GlobalToolCache.InvalidateDir(filepath.Dir(absPath))
				GlobalToolCache.InvalidateSearchCacheForFile(absPath)
			}
		}

	// ディレクトリ作成は検索結果に影響しないため検索キャッシュはクリアしない
	case "create_dir":
		if path := tc.Args["path"]; path != "" {
			if absPath, err := filepath.Abs(path); err == nil {
				GlobalToolCache.InvalidateDir(filepath.Dir(absPath))
			}
		}

	// git checkout でファイルが復元される可能性
	case "git_checkout":
		// 全キャッシュクリア（どのファイルが変更されるか分からない）
		GlobalToolCache.Clear()

	// bash: read-only コマンドならキャッシュを保持、それ以外は全クリア
	case "bash":
		if cmd := tc.Args["command"]; !isBashReadOnly(cmd) {
			GlobalToolCache.Clear()
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

// PreviewToolCall displays tool information without executing it
func PreviewToolCall(tc *ToolCall) {
	cyan.Printf("🔧 Tool: %s (Dry Run)\n", tc.Tool)
	printToolArgs(tc)
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

// displayCollapsedOutput はツール出力を折りたたみ表示
func displayCollapsedOutput(output string) {
	// エラー出力や短い成功メッセージはそのまま表示
	if strings.HasPrefix(output, "Error:") ||
		strings.HasPrefix(output, "Successfully") ||
		strings.HasPrefix(output, "Cancelled") ||
		strings.HasPrefix(output, "No ") {
		// 1行メッセージはそのまま
		if !strings.Contains(output, "\n") {
			dim.Printf("⎿  %s\n", output)
			return
		}
	}

	// 折りたたみ表示
	formatted := ui.FormatToolOutput(output, ui.GetMaxVisibleLines())
	fmt.Print(formatted)
}
