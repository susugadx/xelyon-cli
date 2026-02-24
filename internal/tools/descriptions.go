package tools

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// FC プロバイダー: GetToolDefinitions() 経由で JSON schema の description に使用。
// 非FC プロバイダー: system.go の Available Tools テキストと手動同期。
var ToolDescriptions = map[string]string{
	// File Operations
	"read_file":   "Read file contents. Use to examine source code, config, or any text file. Supports optional line range.",
	"read_files":  "Read multiple files in one call (max 10). Each path supports optional line range: \"path\" or \"path:start-end\". More efficient than multiple read_file calls when you need 2+ files.",
	"write_file":  "Create or overwrite a file. Use str_replace for partial edits to existing files.",
	"str_replace": "Edit existing file by replacing a specific string. You MUST read_file first for old_str mode. Line-range mode (old_str empty + start_line/end_line): can edit lines covered by search_code results without read_file. Batch mode: pass edits=[{old_str,new_str},...] for multiple replacements in one call.",
	"delete_file": "Delete a file permanently.",
	"list_dir":    "List files and directories at the specified path.",

	// Search & Discovery
	"search_code":  "Search code using ripgrep (rg) with grep fallback. Supports multiple comma-separated patterns for parallel search. Groups results by file with context lines and block annotations. Marks matched line ranges as read, enabling str_replace line-range mode (start_line/end_line) without read_file.",
	"grep_replace": "Bulk regex find-and-replace across multiple files. Always specify path to limit scope.",
	"web_search":   "Search the web for information using Serper API.",

	// Development Tools
	"bash": "Execute a shell command. Use for git, npm, pip, make, go test, go fmt, curl, etc.",

	// Planning Tools
	"ask_user_question": "Ask the user a clarification question before planning. Use only when requirements are ambiguous.",
	"create_plan":       "Create and save a new execution plan with title, summary, and steps.",
	"get_plan":          "Retrieve a saved plan by ID or filename. Falls back to last created plan if omitted.",
	"list_plans":        "List saved plans. Supports optional status filter and limit.",
	"update_plan":       "Update a plan: change status, add/remove/update steps, change title or summary. Falls back to last created plan if id omitted.",
	"delete_plan":       "Permanently delete a plan by ID or filename. Falls back to last created plan if omitted.",
}
