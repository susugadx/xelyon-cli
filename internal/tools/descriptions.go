package tools

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// GetToolDefinitions() 経由で JSON schema の description に使用。
var ToolDescriptions = map[string]string{
	// File Operations
	"read_file":   "Read files (1-10). Returns full content. Do not re-read files already returned.",
	"write_file":  "Create or overwrite a file. Uses 0644 for new files and preserves permissions on overwrite. Prefer the primary edit tool for partial edits to existing files.",
	"str_replace": "Edit existing file. PREFERRED: Line-range mode (old_str empty + start_line/end_line) after search_code — no read_file needed. FALLBACK: old_str mode requires read_file first. Batch mode: pass edits=[{old_str,new_str},...] for multiple replacements in one call.",
	"delete_file": "Delete a file permanently.",
	"list_dir":    "Preferred for directory exploration and choosing the next file/subtree. Returns a compact summary with representative names, counts, and types. Ignores .git, node_modules, vendor by default. Use depth parameter (1-3) for recursive listing.",

	// Search & Discovery
	"search_code": "Search and discover project code. Uses language-aware routing across symbol-aware resolution, literal search, and regex search. Prefer mode=auto, short symbol queries when possible, and regex only when needed. Returns contextual matches and may provide richer definition/reference/test results for symbol-like queries in supported languages. Supports comma-separated patterns for parallel multi-search and file filtering (e.g. go, *_test.go).",
	"web_search":  "Search the web and return summarized findings plus source URLs, not full page contents. For deeper coverage, run multiple targeted searches with narrower queries.",

	// Development Tools
	"bash": "Execute a shell command. Use for: git operations, npm/pip install, make, go test, go fmt, curl, compilers. Commands like cat/ls/grep auto-approve. Dangerous commands require confirmation.",

	// Planning Tools
	"ask_user_question": "Ask the user a clarification question before planning. Use only when requirements are ambiguous.",

	// Sub-agent Tools
	"spawn_agent": "Spawn a sub-agent for a well-scoped task. Set task_type to: explore (default, read-only investigation), edit (targeted file modifications), or verify (run build/test/lint). Sub-agents run in isolated context and return only their final report.",
	"wait_agent":  "Wait for sub-agents to complete and receive their results.",
}
