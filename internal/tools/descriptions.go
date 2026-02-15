package tools

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// FC プロバイダー: GetToolDefinitions() 経由で JSON schema の description に使用。
// 非FC プロバイダー: system.go の Available Tools テキストと手動同期。
var ToolDescriptions = map[string]string{
	// File Operations
	"read_file":   "Read file contents. Use to examine source code, config, or any text file. Supports optional line range.",
	"read_files":  "Read multiple files in one call (max 10). Each path supports optional line range: \"path\" or \"path:start-end\". More efficient than multiple read_file calls when you need 2+ files.",
	"write_file":  "Create or overwrite a file. Use str_replace for partial edits to existing files.",
	"str_replace": "Edit existing file by replacing a specific string. You MUST read_file first - never edit a file you haven't read in this session. Use this for ALL edits to existing files regardless of size.",
	"delete_file": "Delete a file permanently.",
	"list_dir":    "List files and directories at the specified path.",

	// Search & Discovery
	"search_code":  "Search code content for a pattern using ripgrep. Supports regex.",
	"search_file":  "Search for files by name pattern using find.",
	"grep_replace": "Bulk regex find-and-replace across multiple files. Always specify path to limit scope.",
	"web_search":   "Search the web for information using Serper API.",

	// Development Tools
	"bash": "Execute a shell command. Use for git, npm, pip, make, go test, go fmt, curl, etc.",

	// Code Navigation
	"lsp_find": "Find a symbol by name and run LSP action (definition/references/implementations). Falls back to grep if LSP unavailable.",

	// Planning Tools
	"ask_user_question": "Ask the user a clarification question before planning. Use only when requirements are ambiguous.",
	"create_plan":       "Create and save a new execution plan with title, summary, and steps.",
	"get_plan":          "Retrieve a saved plan by ID or filename.",
	"list_plans":        "List saved plans. Supports optional status filter and limit.",
	"update_plan":       "Update a plan: change status, add/remove/update steps, change title or summary.",
	"delete_plan":       "Permanently delete a saved plan by ID or filename.",
}
