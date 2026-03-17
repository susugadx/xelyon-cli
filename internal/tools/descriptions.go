package tools

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// GetToolDefinitions() 経由で JSON schema の description に使用。
var ToolDescriptions = map[string]string{
	// File Operations
	"read_file":   "Read file contents. Single: {path} reads up to 300 lines from the top by default; use start_line/end_line only when you already know the target section; use symbol to read specific function/type by name (e.g. symbol=\"Build,HandleRequest\", Go files only). Prefer inspect_symbol when the exact Go symbol name is already known. Batch: {paths: [\"path1\", \"path2:10-20\"]} for multiple files in one call (max 10). Batch is preferred when reading 2+ independent files in the same turn.",
	"write_file":  "Create or overwrite a file. Uses 0644 for new files and preserves permissions on overwrite. Use str_replace for partial edits to existing files.",
	"str_replace": "Edit existing file. PREFERRED: Line-range mode (old_str empty + start_line/end_line) after search_code — no read_file needed. FALLBACK: old_str mode requires read_file first. Batch mode: pass edits=[{old_str,new_str},...] for multiple replacements in one call.",
	"delete_file": "Delete a file permanently.",
	"list_dir":    "Preferred for directory exploration and choosing the next file/subtree. Returns a compact summary with representative names, counts, and types. Ignores .git, node_modules, vendor by default. Use depth parameter (1-3) for recursive listing.",

	// Search & Discovery
	"inspect_symbol": "Preferred for exact Go symbol names. Use instead of search_code+read_file when the target function/type/method name is already known. Supports receiver-qualified method queries such as Config.Build or (*Config).Build. Returns definition body, callers, references, and related tests in one call. Summary mode (default) is compact; full mode expands limits. Returns line-numbered source for direct str_replace line-range edits. Go files only (Phase 1).",
	"search_code":    "Search project code with smart caching and result classification ([def]/[ref]/[call]). Supports multiple comma-separated patterns for parallel search; prefer one multi-pattern search over serial searches when the patterns are independent. Also supports file_type filtering, fixed-string mode, multiline mode, and hidden/ignored file inclusion. Results are grouped by file with context lines and block annotations, and matched line ranges are marked as read for direct str_replace line-range edits.",
	"web_search":     "Search the web and return summarized findings plus source URLs, not full page contents. Uses provider-native search. Configure web_search.provider to choose openai, gemini, or claude when the main provider lacks native search support. For deeper coverage, run multiple targeted searches with narrower queries.",

	// Development Tools
	"bash": "Execute a shell command. Use for: git operations, npm/pip install, make, go test, go fmt, curl, compilers. Commands like cat/ls/grep auto-approve. Dangerous commands require confirmation.",

	// Planning Tools
	"ask_user_question": "Ask the user a clarification question before planning. Use only when requirements are ambiguous.",
}
