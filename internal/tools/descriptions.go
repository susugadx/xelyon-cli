package tools

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// GetToolDefinitions() 経由で JSON schema の description に使用。
var ToolDescriptions = map[string]string{
	// File Operations
	"gather_context": "Primary investigation tool. Pass what code context you need in query; the runtime routes to direct reads, directory listing, auto search, structured impact, reranked recommended reads, and bounded compact evidence prefetch. Use this by default unless you need exact low-level control.",
	"read_file":      "Exact-content reader. gather_context remains the default investigation front door. When read_file is visible on apply_patch surfaces, use it only as an edit exact-control override after you already know the exact file or range. On legacy surfaces it remains a low-level expert override. Reads files (1-10). Default detail=auto returns full content when feasible and outline for large whole-file reads. Optional detail: compact for locator targets or explicit path ranges, full, outline. Do not re-read files already returned.",
	"write_file":     "Create or overwrite a file. Uses 0644 for new files and preserves permissions on overwrite. Prefer the primary edit tool for partial edits to existing files.",
	"str_replace":    "Precise edits to existing files. Copy exact old_str and existing context from actual gather_context, read_file, or search_code output; do not invent old_str. Write new_str as the intended replacement based on that verified context. For multiple edits in the same file, prefer same-file edits=[{old_str,new_str},...] based on actual content. Line-range mode (old_str empty + start_line/end_line) is an advanced fallback only when fresh tool output provides an exact range; do not guess line ranges or use it to avoid evidence.",
	"delete_file":    "Delete a file permanently.",
	"list_dir":       "Low-level directory summary tool. gather_context is the default front door for directory investigation and direct routing. list_dir stays hidden on current gather_context-first agent surfaces; when directly exposed, treat it as a low-level/internal override. Returns a compact summary with representative names, counts, and types. Ignores .git, node_modules, vendor by default. Use depth parameter (1-3) for recursive listing.",

	// Search & Discovery
	"search_code": "Low-level code discovery tool. gather_context remains the default investigation front door. When a legacy surface exposes search_code, keep it as an expert override only when gather_context is clearly insufficient. Uses language-aware routing across symbol-aware resolution, literal search, and regex search. Prefer mode=auto, short symbol queries when possible, and regex only when needed. For related code discovery, comma-separated multi-pattern search is the default. Use one combined query for target + helpers + references/callers + tests instead of serial narrow searches whenever possible. intent=impact is the high-level entrypoint for shared-change impact analysis when you only have one starting symbol. Returns contextual matches and may provide richer definition/reference/test results for symbol-like queries in supported languages. Supports comma-separated patterns for parallel multi-search and file filtering via ripgrep-like built-in language aliases (e.g. go, py, c) or globs (e.g. *_test.go).",
	"web_search":  "Search the web and return summarized findings plus source URLs, not full page contents. For deeper coverage, run multiple targeted searches with narrower queries.",

	// Development Tools
	"bash": "Execute a shell command. Use for: git operations, npm/pip install, make, go test, go fmt, curl, compilers. Commands like cat/ls/grep auto-approve. Dangerous commands require confirmation.",

	// Planning Tools
	"ask_user_question": "Ask the user a clarification question before planning. Use only when requirements are ambiguous.",

	// Sub-agent Tools
	"spawn_agent": "Spawn a sub-agent for a well-scoped task. Set task_type to: explore (default, read-only investigation), edit (targeted file modifications), or verify (run build/test/lint). Sub-agents run in isolated context and return only their final report.",
	"wait_agent":  "Wait for sub-agents to complete and receive their results.",
}
