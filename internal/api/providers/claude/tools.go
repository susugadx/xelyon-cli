package claude

import (
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// ClaudeTool は Anthropic Claude API 用のツール定義
// OpenAI とは異なり、フラットな構造で input_schema を使用
type ClaudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// toolDefinitions - Claude API 用の全ツール定義
// OpenAI tools.go と同じツールセットを Claude 形式で定義
var toolDefinitions = map[string]ClaudeTool{
	// ===== File Operations =====
	"read_file": {
		Name:        "read_file",
		Description: "Reads file contents from the filesystem. Use this to examine source code, configuration files, or any text files.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":       map[string]interface{}{"type": "string", "description": "Absolute or relative file path to read"},
				"start_line": map[string]interface{}{"type": "integer", "description": "Start line number (1-indexed, optional)"},
				"end_line":   map[string]interface{}{"type": "integer", "description": "End line number (1-indexed, optional)"},
			},
			"required": []string{"path"},
		},
	},
	"write_file": {
		Name:        "write_file",
		Description: "Creates a new file or overwrites an existing file with the provided content. Automatically creates a backup (.bak) of existing files.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":    map[string]interface{}{"type": "string", "description": "File path to write to"},
				"content": map[string]interface{}{"type": "string", "description": "Content to write to the file"},
			},
			"required": []string{"path", "content"},
		},
	},
	"str_replace": {
		Name:        "str_replace",
		Description: "Replaces a specific string in a file with new content. Use this for precise edits when you know the exact text to replace.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":       map[string]interface{}{"type": "string", "description": "File path to edit"},
				"old_str":    map[string]interface{}{"type": "string", "description": "Exact string to find and replace"},
				"new_str":    map[string]interface{}{"type": "string", "description": "New string to replace with"},
				"start_line": map[string]interface{}{"type": "string", "description": "Start line number to limit search scope (optional)"},
				"end_line":   map[string]interface{}{"type": "string", "description": "End line number to limit search scope (optional)"},
			},
			"required": []string{"path", "old_str", "new_str"},
		},
	},
	"delete_file": {
		Name:        "delete_file",
		Description: "Deletes a file permanently. Creates a backup before deletion.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File path to delete"},
			},
			"required": []string{"path"},
		},
	},
	"list_dir": {
		Name:        "list_dir",
		Description: "Lists files and directories in the specified path.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Directory path to list"},
			},
			"required": []string{"path"},
		},
	},
	"restore_backup": {
		Name:        "restore_backup",
		Description: "Restores a file from its backup.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":        map[string]interface{}{"type": "string", "description": "Original file path"},
				"backup_path": map[string]interface{}{"type": "string", "description": "Backup file path to restore from"},
			},
			"required": []string{"path", "backup_path"},
		},
	},
	"list_backups": {
		Name:        "list_backups",
		Description: "Lists available backup files for a given file.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File path to find backups for"},
			},
			"required": []string{"path"},
		},
	},

	// ===== Git Operations =====
	"git_commit": {
		Name:        "git_commit",
		Description: "Creates a git commit with staged changes.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{"type": "string", "description": "Commit message"},
			},
			"required": []string{"message"},
		},
	},
	"git_checkout": {
		Name:        "git_checkout",
		Description: "Checks out a branch or commit.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ref": map[string]interface{}{"type": "string", "description": "Branch name or commit hash"},
			},
			"required": []string{"ref"},
		},
	},

	// ===== Search Operations =====
	"search_code": {
		Name:        "search_code",
		Description: "Searches for code patterns using ripgrep. Returns matching lines with context.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{"type": "string", "description": "Search pattern (regex supported)"},
				"path":    map[string]interface{}{"type": "string", "description": "Directory or file to search in (optional)"},
				"include": map[string]interface{}{"type": "string", "description": "File pattern to include (e.g., '*.go')"},
			},
			"required": []string{"pattern"},
		},
	},
	"search_file": {
		Name:        "search_file",
		Description: "Searches for files by name pattern using fd.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{"type": "string", "description": "File name pattern (glob supported)"},
				"path":    map[string]interface{}{"type": "string", "description": "Directory to search in (optional)"},
			},
			"required": []string{"pattern"},
		},
	},
	"web_search": {
		Name:        "web_search",
		Description: "Searches the web for information using Serper API.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query"},
			},
			"required": []string{"query"},
		},
	},
	"ast_grep": {
		Name:        "ast_grep",
		Description: "Searches code using AST patterns for semantic matching.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern":  map[string]interface{}{"type": "string", "description": "AST pattern to search for"},
				"path":     map[string]interface{}{"type": "string", "description": "Directory or file to search in"},
				"language": map[string]interface{}{"type": "string", "description": "Programming language (go, python, javascript, etc.)"},
			},
			"required": []string{"pattern", "language"},
		},
	},
	"grep_replace": {
		Name:        "grep_replace",
		Description: "Performs search and replace across multiple files using patterns.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern":     map[string]interface{}{"type": "string", "description": "Search pattern (regex)"},
				"replacement": map[string]interface{}{"type": "string", "description": "Replacement string"},
				"path":        map[string]interface{}{"type": "string", "description": "Directory to search in"},
				"include":     map[string]interface{}{"type": "string", "description": "File pattern to include (e.g., '*.go')"},
			},
			"required": []string{"pattern", "replacement"},
		},
	},

	// ===== Development Operations =====
	"run_test": {
		Name:        "run_test",
		Description: "Runs tests for the project.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":    map[string]interface{}{"type": "string", "description": "Test file or directory path"},
				"verbose": map[string]interface{}{"type": "boolean", "description": "Enable verbose output"},
			},
			"required": []string{},
		},
	},
	"format": {
		Name:        "format",
		Description: "Formats code according to language standards (gofmt, prettier, etc.).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File or directory to format"},
			},
			"required": []string{"path"},
		},
	},
	"lint": {
		Name:        "lint",
		Description: "Runs linter on the code.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File or directory to lint"},
			},
			"required": []string{"path"},
		},
	},
	"http_request": {
		Name:        "http_request",
		Description: "Makes an HTTP request and returns the response.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"method":  map[string]interface{}{"type": "string", "description": "HTTP method (GET, POST, PUT, DELETE, etc.)"},
				"url":     map[string]interface{}{"type": "string", "description": "Request URL"},
				"headers": map[string]interface{}{"type": "object", "description": "Request headers"},
				"body":    map[string]interface{}{"type": "string", "description": "Request body (for POST/PUT)"},
			},
			"required": []string{"method", "url"},
		},
	},
	"bash": {
		Name:        "bash",
		Description: "Executes a bash command. Use with caution.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{"type": "string", "description": "Bash command to execute"},
			},
			"required": []string{"command"},
		},
	},
	"move_file": {
		Name:        "move_file",
		Description: "Moves or renames a file.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"source":      map[string]interface{}{"type": "string", "description": "Source file path"},
				"destination": map[string]interface{}{"type": "string", "description": "Destination file path"},
			},
			"required": []string{"source", "destination"},
		},
	},
	"delete_lines": {
		Name:        "delete_lines",
		Description: "Deletes specific lines from a file.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":       map[string]interface{}{"type": "string", "description": "File path"},
				"start_line": map[string]interface{}{"type": "integer", "description": "Start line number (1-indexed)"},
				"end_line":   map[string]interface{}{"type": "integer", "description": "End line number (1-indexed, inclusive)"},
			},
			"required": []string{"path", "start_line", "end_line"},
		},
	},

	// ===== LSP Operations =====
	"lsp_references": {
		Name:        "lsp_references",
		Description: "Finds all references to a symbol using LSP.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
			},
			"required": []string{"path", "line", "character"},
		},
	},
	"lsp_definition": {
		Name:        "lsp_definition",
		Description: "Goes to the definition of a symbol using LSP.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
			},
			"required": []string{"path", "line", "character"},
		},
	},
	"lsp_hover": {
		Name:        "lsp_hover",
		Description: "Gets hover information for a symbol using LSP.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
			},
			"required": []string{"path", "line", "character"},
		},
	},
	"lsp_diagnostics": {
		Name:        "lsp_diagnostics",
		Description: "Gets diagnostics (errors, warnings) for a file using LSP.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "File path"},
			},
			"required": []string{"path"},
		},
	},
	"lsp_rename": {
		Name:        "lsp_rename",
		Description: "Renames a symbol across the codebase using LSP.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "File path"},
				"line":      map[string]interface{}{"type": "integer", "description": "Line number (1-indexed)"},
				"character": map[string]interface{}{"type": "integer", "description": "Character position (1-indexed)"},
				"new_name":  map[string]interface{}{"type": "string", "description": "New name for the symbol"},
			},
			"required": []string{"path", "line", "character", "new_name"},
		},
	},
}

// GetClaudeToolDefinitions は組み込みツール定義を Claude 形式で返す
func GetClaudeToolDefinitions() []ClaudeTool {
	tools := make([]ClaudeTool, 0, len(toolDefinitions))
	for _, def := range toolDefinitions {
		tools = append(tools, def)
	}
	return tools
}

// ConvertOpenAIToolToClaude は OpenAI 形式のツールを Claude 形式に変換する
func ConvertOpenAIToolToClaude(tool api.OpenAIToolFunction) ClaudeTool {
	return ClaudeTool{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: tool.Parameters,
	}
}

// GetCombinedClaudeTools は組み込みツール + MCPツールを返す
func GetCombinedClaudeTools(mcpTools []api.OpenAIToolFunction) []ClaudeTool {
	tools := GetClaudeToolDefinitions()
	for _, mcp := range mcpTools {
		tools = append(tools, ConvertOpenAIToolToClaude(mcp))
	}
	return tools
}

// internalToolCall は内部ツール呼び出し形式（JSON出力順序を保証）
type internalToolCall struct {
	ID   string         `json:"id,omitempty"` // Claude の tool_use ID
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// ConvertToolUseToToolJSON は Claude の tool_use を内部JSON形式に変換
// Returns: {"id": "toolu_01ABC...", "tool": "read_file", "args": {"path": "/path/to/file"}}
func ConvertToolUseToToolJSON(id, name string, input map[string]interface{}) (string, error) {
	// input の型を map[string]any に変換
	args := make(map[string]any, len(input))
	for k, v := range input {
		args[k] = v
	}

	toolCall := internalToolCall{
		ID:   id,
		Tool: name,
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
