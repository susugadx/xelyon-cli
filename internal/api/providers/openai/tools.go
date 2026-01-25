package openai

import (
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// toolDefinitions - OpenAI Chat Completions API 用の全ツール定義
// Gemini と同じツールセットを OpenAI 形式で定義
var toolDefinitions = map[string]api.OpenAIToolFunction{
	// ===== File Operations =====
	"read_file": {
		Name:        "read_file",
		Description: "Reads file contents from the filesystem. Use this to examine source code, configuration files, or any text files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":       map[string]interface{}{"type": "string", "description": "Absolute or relative file path to read"},
				"start_line": map[string]interface{}{"type": "integer", "description": "Start line number (1-indexed, optional)"},
				"end_line":   map[string]interface{}{"type": "integer", "description": "End line number (1-indexed, optional)"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	},
	"write_file": {
		Name:        "write_file",
		Description: "Creates a new file or overwrites an existing file with the provided content. Automatically creates a backup (.bak) of existing files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":    map[string]interface{}{"type": "string", "description": "File path to write to"},
				"content": map[string]interface{}{"type": "string", "description": "Content to write to the file"},
			},
			"required":             []string{"path", "content"},
			"additionalProperties": false,
		},
	},
	"str_replace": {
		Name:        "str_replace",
		Description: "Replaces a specific string in a file with new content. Use this for precise edits when you know the exact text to replace.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":       map[string]interface{}{"type": "string", "description": "File path to edit"},
				"old_str":    map[string]interface{}{"type": "string", "description": "Exact string to find and replace"},
				"new_str":    map[string]interface{}{"type": "string", "description": "New string to replace with"},
				"start_line": map[string]interface{}{"type": "string", "description": "Start line number to limit search scope (optional)"},
				"end_line":   map[string]interface{}{"type": "string", "description": "End line number to limit search scope (optional)"},
			},
			"required":             []string{"path", "old_str", "new_str"},
			"additionalProperties": false,
		},
	},
	"delete_file": {
		Name:        "delete_file",
		Description: "Deletes a file permanently. Creates a backup before deletion.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File path to delete"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	},
	"list_dir": {
		Name:        "list_dir",
		Description: "Lists files and directories in the specified path.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Directory path to list"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	},
	"restore_backup": {
		Name:        "restore_backup",
		Description: "Restores a file from its backup.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":        map[string]interface{}{"type": "string", "description": "Original file path"},
				"backup_path": map[string]interface{}{"type": "string", "description": "Backup file path to restore from"},
			},
			"required":             []string{"path", "backup_path"},
			"additionalProperties": false,
		},
	},
	"list_backups": {
		Name:        "list_backups",
		Description: "Lists available backup files for a given file.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File path to find backups for"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	},

	// ===== Git Operations =====
	"git_commit": {
		Name:        "git_commit",
		Description: "Creates a git commit with the specified message.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{"type": "string", "description": "Commit message"},
			},
			"required":             []string{"message"},
			"additionalProperties": false,
		},
	},
	"git_checkout": {
		Name:        "git_checkout",
		Description: "Switches branches or restores files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{"type": "string", "description": "Branch name or file path to checkout"},
			},
			"required":             []string{"target"},
			"additionalProperties": false,
		},
	},

	// ===== Search Operations =====
	"search_code": {
		Name:        "search_code",
		Description: "Searches for a pattern in source code files using ripgrep.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{"type": "string", "description": "Search pattern (regex supported)"},
				"path":    map[string]interface{}{"type": "string", "description": "Directory or file to search in (optional)"},
			},
			"required":             []string{"pattern"},
			"additionalProperties": false,
		},
	},
	"search_file": {
		Name:        "search_file",
		Description: "Searches for files by name pattern using find.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{"type": "string", "description": "File name pattern (glob or partial name)"},
				"path":    map[string]interface{}{"type": "string", "description": "Directory to search in (optional)"},
			},
			"required":             []string{"pattern"},
			"additionalProperties": false,
		},
	},
	"web_search": {
		Name:        "web_search",
		Description: "Searches the web for information using Serper API.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query"},
			},
			"required":             []string{"query"},
			"additionalProperties": false,
		},
	},
	"ast_grep": {
		Name:        "ast_grep",
		Description: "Performs structural code search using AST patterns (Tree-sitter based).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{"type": "string", "description": "AST pattern to search for"},
				"lang":    map[string]interface{}{"type": "string", "description": "Language: go, python, javascript, typescript, rust, c, cpp", "enum": []string{"go", "python", "javascript", "typescript", "rust", "c", "cpp"}},
				"path":    map[string]interface{}{"type": "string", "description": "Directory or file to search in (optional)"},
			},
			"required":             []string{"pattern", "lang"},
			"additionalProperties": false,
		},
	},
	"grep_replace": {
		Name:        "grep_replace",
		Description: "Performs bulk find-and-replace across multiple files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern":      map[string]interface{}{"type": "string", "description": "Search pattern (regex)"},
				"replacement":  map[string]interface{}{"type": "string", "description": "Replacement string"},
				"path":         map[string]interface{}{"type": "string", "description": "Directory to search in"},
				"file_pattern": map[string]interface{}{"type": "string", "description": "File glob pattern (e.g., *.go)"},
				"dry_run":      map[string]interface{}{"type": "string", "description": "Set to 'true' for preview without changes"},
			},
			"required":             []string{"pattern", "replacement"},
			"additionalProperties": false,
		},
	},

	// ===== Development Operations =====
	"run_test": {
		Name:        "run_test",
		Description: "Runs tests for the specified path or the entire project.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Test file or directory path (optional)"},
			},
			"additionalProperties": false,
		},
	},
	"format": {
		Name:        "format",
		Description: "Formats source code file (go fmt, prettier, etc.).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File path to format"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	},
	"lint": {
		Name:        "lint",
		Description: "Runs linter on the specified file with optional auto-fix.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":     map[string]interface{}{"type": "string", "description": "File path to lint"},
				"auto_fix": map[string]interface{}{"type": "string", "description": "Set to 'true' to auto-fix issues"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	},
	"http_request": {
		Name:        "http_request",
		Description: "Executes HTTP requests to external APIs.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"method":  map[string]interface{}{"type": "string", "description": "HTTP method", "enum": []string{"GET", "POST", "PUT", "DELETE", "PATCH"}},
				"url":     map[string]interface{}{"type": "string", "description": "Request URL"},
				"headers": map[string]interface{}{"type": "string", "description": "Request headers as JSON string"},
				"body":    map[string]interface{}{"type": "string", "description": "Request body"},
				"timeout": map[string]interface{}{"type": "string", "description": "Timeout in seconds (default: 30)"},
			},
			"required":             []string{"method", "url"},
			"additionalProperties": false,
		},
	},
	"bash": {
		Name:        "bash",
		Description: "Executes a shell command. Use for system operations, running scripts, or commands not covered by other tools.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{"type": "string", "description": "Shell command to execute"},
			},
			"required":             []string{"command"},
			"additionalProperties": false,
		},
	},

	// ===== LSP Tools (Code Intelligence) =====
	"lsp_references": {
		Name:        "lsp_references",
		Description: "Find all references to a symbol at the specified position. Requires LSP server installed (e.g., gopls for Go).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
			},
			"required":             []string{"path", "line", "character"},
			"additionalProperties": false,
		},
	},
	"lsp_definition": {
		Name:        "lsp_definition",
		Description: "Go to the definition of a symbol at the specified position. Requires LSP server installed.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
			},
			"required":             []string{"path", "line", "character"},
			"additionalProperties": false,
		},
	},
	"lsp_hover": {
		Name:        "lsp_hover",
		Description: "Get type information and documentation for a symbol at the specified position. Requires LSP server installed.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
			},
			"required":             []string{"path", "line", "character"},
			"additionalProperties": false,
		},
	},
	"lsp_diagnostics": {
		Name:        "lsp_diagnostics",
		Description: "Get errors and warnings for a file from the LSP server. Requires LSP server installed.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File path to get diagnostics for"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	},
	"lsp_rename": {
		Name:        "lsp_rename",
		Description: "Preview rename changes for a symbol at the specified position. Requires LSP server installed.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
				"new_name":  map[string]interface{}{"type": "string", "description": "New name for the symbol"},
			},
			"required":             []string{"path", "line", "character", "new_name"},
			"additionalProperties": false,
		},
	},
}

// GetOpenAIToolDefinitions は組み込みツール定義を OpenAI 形式で返す
func GetOpenAIToolDefinitions() []api.OpenAITool {
	tools := make([]api.OpenAITool, 0, len(toolDefinitions))
	for _, def := range toolDefinitions {
		defCopy := def // ループ変数のコピー
		tools = append(tools, api.OpenAITool{
			Type:     "function",
			Function: &defCopy,
		})
	}
	return tools
}

// GetCombinedOpenAITools は組み込みツール + MCPツールを返す
func GetCombinedOpenAITools(mcpTools []api.OpenAIToolFunction) []api.OpenAITool {
	tools := GetOpenAIToolDefinitions()
	for _, mcp := range mcpTools {
		mcpCopy := mcp // ループ変数のコピー
		tools = append(tools, api.OpenAITool{
			Type:     "function",
			Function: &mcpCopy,
		})
	}
	return tools
}

// GetResponsesToolDefinitions は Responses API 用のツール定義を返す
// Responses API は Chat Completions と異なりフラットな形式
func GetResponsesToolDefinitions(mcpTools []api.OpenAIToolFunction) []ResponsesTool {
	tools := make([]ResponsesTool, 0, len(toolDefinitions)+len(mcpTools))

	// 組み込みツール
	for _, def := range toolDefinitions {
		tools = append(tools, ResponsesTool{
			Type:        "function",
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}

	// MCPツール
	for _, mcp := range mcpTools {
		tools = append(tools, ResponsesTool{
			Type:        "function",
			Name:        mcp.Name,
			Description: mcp.Description,
			Parameters:  mcp.Parameters,
		})
	}

	return tools
}

// internalToolCall は内部ツール呼び出し形式（JSON出力順序を保証）
type internalToolCall struct {
	ID   string         `json:"id,omitempty"` // Function Calling 用の tool_call_id
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// ConvertToolCallToToolJSON は OpenAI tool_call を内部JSON形式に変換
// Returns: {"id": "call_xxx", "tool": "read_file", "args": {"path": "/path/to/file"}}
func ConvertToolCallToToolJSON(tc *api.OpenAIToolCall) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return "", err
	}
	toolCall := internalToolCall{
		ID:   tc.ID,
		Tool: tc.Function.Name,
		Args: args,
	}
	jsonBytes, err := json.Marshal(toolCall)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// GetToolDefinitionNames は定義済みツール名一覧を返す（テスト用）
func GetToolDefinitionNames() []string {
	names := make([]string, 0, len(toolDefinitions))
	for name := range toolDefinitions {
		names = append(names, name)
	}
	return names
}
