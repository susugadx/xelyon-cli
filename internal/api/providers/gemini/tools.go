package gemini

import (
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// toolDefinitions - 全35ツールの定義
var toolDefinitions = map[string]api.GeminiFunctionDeclaration{
	// ===== File Operations =====
	"read_file": {
		Name:        "read_file",
		Description: "Reads file contents from the filesystem. Use this to examine source code, configuration files, or any text files.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":       {Type: "string", Description: "Absolute or relative file path to read"},
				"start_line": {Type: "integer", Description: "Start line number (1-indexed, optional)"},
				"end_line":   {Type: "integer", Description: "End line number (1-indexed, optional)"},
			},
			Required: []string{"path"},
		},
	},
	"write_file": {
		Name:        "write_file",
		Description: "Creates a new file or overwrites an existing file with the provided content. Automatically creates a backup (.bak) of existing files.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":    {Type: "string", Description: "File path to write to"},
				"content": {Type: "string", Description: "Content to write to the file"},
			},
			Required: []string{"path", "content"},
		},
	},
	"str_replace": {
		Name:        "str_replace",
		Description: "Replaces a specific string in a file with new content. Use this for precise edits when you know the exact text to replace.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":       {Type: "string", Description: "File path to edit"},
				"old_str":    {Type: "string", Description: "Exact string to find and replace"},
				"new_str":    {Type: "string", Description: "New string to replace with"},
				"start_line": {Type: "string", Description: "Start line number to limit search scope (optional)"},
				"end_line":   {Type: "string", Description: "End line number to limit search scope (optional)"},
			},
			Required: []string{"path", "old_str", "new_str"},
		},
	},
	"delete_file": {
		Name:        "delete_file",
		Description: "Deletes a file permanently. Creates a backup before deletion.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to delete"},
			},
			Required: []string{"path"},
		},
	},
	"list_dir": {
		Name:        "list_dir",
		Description: "Lists files and directories in the specified path.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path": {Type: "string", Description: "Directory path to list"},
			},
			Required: []string{"path"},
		},
	},
	"restore_backup": {
		Name:        "restore_backup",
		Description: "Restores a file from its backup.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":        {Type: "string", Description: "Original file path"},
				"backup_path": {Type: "string", Description: "Backup file path to restore from"},
			},
			Required: []string{"path", "backup_path"},
		},
	},
	"list_backups": {
		Name:        "list_backups",
		Description: "Lists available backup files for a given file.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to find backups for"},
			},
			Required: []string{"path"},
		},
	},

	// ===== Git Operations =====
	"git_commit": {
		Name:        "git_commit",
		Description: "Creates a git commit with the specified message.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"message": {Type: "string", Description: "Commit message"},
			},
			Required: []string{"message"},
		},
	},
	"git_checkout": {
		Name:        "git_checkout",
		Description: "Switches branches or restores files.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"target": {Type: "string", Description: "Branch name or file path to checkout"},
			},
			Required: []string{"target"},
		},
	},

	// ===== Search Operations =====
	"search_code": {
		Name:        "search_code",
		Description: "Searches for a pattern in source code files using ripgrep.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"pattern": {Type: "string", Description: "Search pattern (regex supported)"},
				"path":    {Type: "string", Description: "Directory or file to search in (optional)"},
			},
			Required: []string{"pattern"},
		},
	},
	"search_file": {
		Name:        "search_file",
		Description: "Searches for files by name pattern using find.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"pattern": {Type: "string", Description: "File name pattern (glob or partial name)"},
				"path":    {Type: "string", Description: "Directory to search in (optional)"},
			},
			Required: []string{"pattern"},
		},
	},
	"web_search": {
		Name:        "web_search",
		Description: "Searches the web for information using Serper API.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"query": {Type: "string", Description: "Search query"},
			},
			Required: []string{"query"},
		},
	},
	"ast_grep": {
		Name:        "ast_grep",
		Description: "Performs structural code search using AST patterns (Tree-sitter based).",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"pattern": {Type: "string", Description: "AST pattern to search for"},
				"lang":    {Type: "string", Description: "Language: go, python, javascript, typescript, rust, c, cpp", Enum: []string{"go", "python", "javascript", "typescript", "rust", "c", "cpp"}},
				"path":    {Type: "string", Description: "Directory or file to search in (optional)"},
			},
			Required: []string{"pattern", "lang"},
		},
	},
	"grep_replace": {
		Name:        "grep_replace",
		Description: "Performs bulk find-and-replace across multiple files.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"pattern":      {Type: "string", Description: "Search pattern (regex)"},
				"replacement":  {Type: "string", Description: "Replacement string"},
				"path":         {Type: "string", Description: "Directory to search in"},
				"file_pattern": {Type: "string", Description: "File glob pattern (e.g., *.go)"},
				"dry_run":      {Type: "string", Description: "Set to 'true' for preview without changes"},
			},
			Required: []string{"pattern", "replacement"},
		},
	},

	// ===== Development Operations =====
	"run_test": {
		Name:        "run_test",
		Description: "Runs tests for the specified path or the entire project.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path": {Type: "string", Description: "Test file or directory path (optional)"},
			},
		},
	},
	"format": {
		Name:        "format",
		Description: "Formats source code file (go fmt, prettier, etc.).",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to format"},
			},
			Required: []string{"path"},
		},
	},
	"lint": {
		Name:        "lint",
		Description: "Runs linter on the specified file with optional auto-fix.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":     {Type: "string", Description: "File path to lint"},
				"auto_fix": {Type: "string", Description: "Set to 'true' to auto-fix issues"},
			},
			Required: []string{"path"},
		},
	},
	"http_request": {
		Name:        "http_request",
		Description: "Executes HTTP requests to external APIs.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"method":  {Type: "string", Description: "HTTP method", Enum: []string{"GET", "POST", "PUT", "DELETE", "PATCH"}},
				"url":     {Type: "string", Description: "Request URL"},
				"headers": {Type: "string", Description: "Request headers as JSON string"},
				"body":    {Type: "string", Description: "Request body"},
				"timeout": {Type: "string", Description: "Timeout in seconds (default: 30)"},
			},
			Required: []string{"method", "url"},
		},
	},
	"bash": {
		Name:        "bash",
		Description: "Executes a shell command. Use for system operations, running scripts, or commands not covered by other tools.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"command": {Type: "string", Description: "Shell command to execute"},
			},
			Required: []string{"command"},
		},
	},

	// ===== LSP Tools (Code Intelligence) =====
	"lsp_references": {
		Name:        "lsp_references",
		Description: "Find all references to a symbol at the specified position. Requires LSP server installed (e.g., gopls for Go).",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":      {Type: "string", Description: "File path"},
				"line":      {Type: "integer", Description: "Line number (1-indexed)"},
				"character": {Type: "integer", Description: "Character position (1-indexed)"},
			},
			Required: []string{"path", "line", "character"},
		},
	},
	"lsp_definition": {
		Name:        "lsp_definition",
		Description: "Go to the definition of a symbol at the specified position. Requires LSP server installed.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":      {Type: "string", Description: "File path"},
				"line":      {Type: "integer", Description: "Line number (1-indexed)"},
				"character": {Type: "integer", Description: "Character position (1-indexed)"},
			},
			Required: []string{"path", "line", "character"},
		},
	},
	"lsp_hover": {
		Name:        "lsp_hover",
		Description: "Get type information and documentation for a symbol at the specified position. Requires LSP server installed.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":      {Type: "string", Description: "File path"},
				"line":      {Type: "integer", Description: "Line number (1-indexed)"},
				"character": {Type: "integer", Description: "Character position (1-indexed)"},
			},
			Required: []string{"path", "line", "character"},
		},
	},
	"lsp_diagnostics": {
		Name:        "lsp_diagnostics",
		Description: "Get errors and warnings for a file from the LSP server. Requires LSP server installed.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to get diagnostics for"},
			},
			Required: []string{"path"},
		},
	},
	"lsp_rename": {
		Name:        "lsp_rename",
		Description: "Preview rename changes for a symbol at the specified position. Requires LSP server installed.",
		Parameters: &api.GeminiParameterSchema{
			Type: "object",
			Properties: map[string]api.GeminiPropertyDef{
				"path":      {Type: "string", Description: "File path"},
				"line":      {Type: "integer", Description: "Line number (1-indexed)"},
				"character": {Type: "integer", Description: "Character position (1-indexed)"},
				"new_name":  {Type: "string", Description: "New name for the symbol"},
			},
			Required: []string{"path", "line", "character", "new_name"},
		},
	},
}

// GetGeminiToolDefinitions returns all tool definitions for Function Calling API
func GetGeminiToolDefinitions() []api.GeminiToolConfig {
	declarations := make([]api.GeminiFunctionDeclaration, 0, len(toolDefinitions))
	for _, def := range toolDefinitions {
		declarations = append(declarations, def)
	}
	return []api.GeminiToolConfig{{FunctionDeclarations: declarations}}
}

// GetToolDefinitionNames returns all defined tool names for testing
func GetToolDefinitionNames() []string {
	names := make([]string, 0, len(toolDefinitions))
	for name := range toolDefinitions {
		names = append(names, name)
	}
	return names
}

// internalToolCall は内部ツール呼び出し形式（JSON出力順序を保証）
type internalToolCall struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// convertFunctionCallToToolJSON converts Gemini FunctionCall to internal tool JSON format
// Returns: {"tool": "read_file", "args": {"path": "/path/to/file"}}
// NOTE: 構造体を使用して "tool" が "args" より前に来ることを保証
// ParseToolCalls は {"tool" パターンで検索するため、この順序が重要
func convertFunctionCallToToolJSON(fc *api.GeminiFunctionCall) string {
	toolCall := internalToolCall{
		Tool: fc.Name,
		Args: fc.Args,
	}
	jsonBytes, _ := json.Marshal(toolCall)
	return string(jsonBytes)
}

// GetCombinedToolDefinitions は組み込みツール + MCPツールの定義を返す
func GetCombinedToolDefinitions(mcpTools []api.GeminiFunctionDeclaration) []api.GeminiToolConfig {
	declarations := make([]api.GeminiFunctionDeclaration, 0, len(toolDefinitions)+len(mcpTools))

	// 組み込みツール
	for _, def := range toolDefinitions {
		declarations = append(declarations, def)
	}

	// MCPツール
	declarations = append(declarations, mcpTools...)

	return []api.GeminiToolConfig{{FunctionDeclarations: declarations}}
}
