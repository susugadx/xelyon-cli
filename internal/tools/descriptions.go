package tools

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// FC プロバイダー: GetToolDefinitions() 経由で JSON schema の description に使用。
// 非FC プロバイダー: system.go の Available Tools テキストと手動同期。
var ToolDescriptions = map[string]string{
	// File Operations
	"read_file":   "Read file contents. Single: {path, start_line?, end_line?}. Batch: {paths: [\"path1\", \"path2:10-20\"]} for multiple files in one call (max 10). Batch is preferred when reading 2+ files.",
	"write_file":  "Create or overwrite a file. Use str_replace for partial edits to existing files.",
	"str_replace": "Edit existing file. PREFERRED: Line-range mode (old_str empty + start_line/end_line) after search_code — no read_file needed. FALLBACK: old_str mode requires read_file first. Batch mode: pass edits=[{old_str,new_str},...] for multiple replacements in one call.",
	"delete_file": "Delete a file permanently.",
	"list_dir":    "List files and directories at the specified path.",

	// Search & Discovery
	"search_code": "Search code using ripgrep (rg) with grep fallback. Supports multiple comma-separated patterns for parallel search. Groups results by file with context lines and block annotations. Marks matched line ranges as read, enabling str_replace line-range mode (start_line/end_line) without read_file.",
	"web_search":  "Search the web for information using Serper API.",

	// Development Tools
	"bash": "Execute a shell command. Use for git, npm, pip, make, go test, go fmt, curl, etc.",

	// Planning Tools
	"ask_user_question": "Ask the user a clarification question before planning. Use only when requirements are ambiguous.",
}
