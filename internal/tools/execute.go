package tools

import (
	"fmt"
	"strings"
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
	case "search_code", "search_file":
		fmt.Printf("   Pattern: %s\n", tc.Args["pattern"])
		if tc.Args["path"] != "" {
			fmt.Printf("   Path: %s\n", tc.Args["path"])
		}
	case "insert_after", "insert_before":
		fmt.Printf("   File: %s\n", tc.Args["path"])
		fmt.Printf("   Pattern: %s\n", truncate(tc.Args["pattern"], 60))
		fmt.Printf("   Content: %s\n", truncate(tc.Args["content"], 60))
	case "copy_file":
		fmt.Printf("   Source: %s\n", tc.Args["src"])
		fmt.Printf("   Destination: %s\n", tc.Args["dest"])
	case "delete_lines":
		fmt.Printf("   File: %s\n", tc.Args["path"])
		fmt.Printf("   Lines: %s-%s\n", tc.Args["start_line"], tc.Args["end_line"])
	case "delete_file":
		fmt.Printf("   File: %s\n", tc.Args["path"])
	case "move_file":
		fmt.Printf("   Source: %s\n", tc.Args["src"])
		fmt.Printf("   Destination: %s\n", tc.Args["dest"])
	case "lint":
		path := tc.Args["path"]
		if path == "" {
			path = "."
		}
		fmt.Printf("   Path: %s\n", path)
		if tc.Args["auto_fix"] == "true" {
			fmt.Printf("   Auto-fix: enabled\n")
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
	// list_dir, git_add, search_code, search_fileでpathが空の場合"."を設定
	if tc.Args["path"] == "" {
		switch tc.Tool {
		case "list_dir", "git_add", "search_code", "search_file":
			tc.Args["path"] = "."
		}
	}

	// Registry経由でツール実行
	result, change := DefaultRegistry.Execute(tc)

	// ツール結果はAIに渡すのみ（各ツールが出す要約表示で十分）
	return result, change
}

// PreviewToolCall displays tool information without executing it
func PreviewToolCall(tc *ToolCall) {
	cyan.Printf("🔧 Tool: %s (Dry Run)\n", tc.Tool)
	printToolArgs(tc)
}
