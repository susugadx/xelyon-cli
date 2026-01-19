package api

import "encoding/json"

// GeminiFunctionDeclaration - Gemini API用の関数宣言
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  *GeminiParameterSchema `json:"parameters,omitempty"`
}

// GeminiParameterSchema - パラメータスキーマ
type GeminiParameterSchema struct {
	Type       string                       `json:"type"`
	Properties map[string]GeminiPropertyDef `json:"properties,omitempty"`
	Required   []string                     `json:"required,omitempty"`
}

// GeminiPropertyDef - プロパティ定義
type GeminiPropertyDef struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// GeminiToolConfig - API リクエスト用ツール設定
type GeminiToolConfig struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"function_declarations"`
}

// toolDefinitions - 全35ツールの定義
var toolDefinitions = map[string]GeminiFunctionDeclaration{
	// ===== File Operations =====
	"read_file": {
		Name:        "read_file",
		Description: "Reads file contents from the filesystem. Use this to examine source code, configuration files, or any text files.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
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
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":    {Type: "string", Description: "File path to write to"},
				"content": {Type: "string", Description: "Content to write to the file"},
			},
			Required: []string{"path", "content"},
		},
	},
	"str_replace": {
		Name:        "str_replace",
		Description: "Replaces a specific string in a file with new content. Use this for precise edits when you know the exact text to replace.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":       {Type: "string", Description: "File path to edit"},
				"old_str":    {Type: "string", Description: "Exact string to find and replace"},
				"new_str":    {Type: "string", Description: "New string to replace with"},
				"start_line": {Type: "string", Description: "Start line number to limit search scope (optional)"},
				"end_line":   {Type: "string", Description: "End line number to limit search scope (optional)"},
			},
			Required: []string{"path", "old_str", "new_str"},
		},
	},
	"append_file": {
		Name:        "append_file",
		Description: "Appends content to the end of an existing file.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":    {Type: "string", Description: "File path to append to"},
				"content": {Type: "string", Description: "Content to append"},
			},
			Required: []string{"path", "content"},
		},
	},
	"prepend_file": {
		Name:        "prepend_file",
		Description: "Inserts content at the beginning of an existing file.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":    {Type: "string", Description: "File path to prepend to"},
				"content": {Type: "string", Description: "Content to insert at the beginning"},
			},
			Required: []string{"path", "content"},
		},
	},
	"insert_after": {
		Name:        "insert_after",
		Description: "Inserts content after the first line matching a pattern.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":    {Type: "string", Description: "File path to edit"},
				"pattern": {Type: "string", Description: "Pattern to find (first match)"},
				"content": {Type: "string", Description: "Content to insert after the matched line"},
			},
			Required: []string{"path", "pattern", "content"},
		},
	},
	"insert_before": {
		Name:        "insert_before",
		Description: "Inserts content before the first line matching a pattern.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":    {Type: "string", Description: "File path to edit"},
				"pattern": {Type: "string", Description: "Pattern to find (first match)"},
				"content": {Type: "string", Description: "Content to insert before the matched line"},
			},
			Required: []string{"path", "pattern", "content"},
		},
	},
	"copy_file": {
		Name:        "copy_file",
		Description: "Copies a file from source to destination.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"src":  {Type: "string", Description: "Source file path"},
				"dest": {Type: "string", Description: "Destination file path"},
			},
			Required: []string{"src", "dest"},
		},
	},
	"move_file": {
		Name:        "move_file",
		Description: "Moves or renames a file.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"src":  {Type: "string", Description: "Source file path"},
				"dest": {Type: "string", Description: "Destination file path"},
			},
			Required: []string{"src", "dest"},
		},
	},
	"delete_file": {
		Name:        "delete_file",
		Description: "Deletes a file permanently. Creates a backup before deletion.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to delete"},
			},
			Required: []string{"path"},
		},
	},
	"delete_lines": {
		Name:        "delete_lines",
		Description: "Deletes a range of lines from a file.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":       {Type: "string", Description: "File path to edit"},
				"start_line": {Type: "string", Description: "Start line number (1-indexed)"},
				"end_line":   {Type: "string", Description: "End line number (1-indexed)"},
			},
			Required: []string{"path", "start_line", "end_line"},
		},
	},
	"list_dir": {
		Name:        "list_dir",
		Description: "Lists files and directories in the specified path.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "Directory path to list"},
			},
			Required: []string{"path"},
		},
	},
	"create_dir": {
		Name:        "create_dir",
		Description: "Creates a new directory (including parent directories if needed).",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "Directory path to create"},
			},
			Required: []string{"path"},
		},
	},
	"restore_backup": {
		Name:        "restore_backup",
		Description: "Restores a file from its backup.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":        {Type: "string", Description: "Original file path"},
				"backup_path": {Type: "string", Description: "Backup file path to restore from"},
			},
			Required: []string{"path", "backup_path"},
		},
	},
	"list_backups": {
		Name:        "list_backups",
		Description: "Lists available backup files for a given file.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to find backups for"},
			},
			Required: []string{"path"},
		},
	},

	// ===== Git Operations =====
	"git_status": {
		Name:        "git_status",
		Description: "Shows the current git status including staged, unstaged, and untracked files.",
		Parameters:  nil,
	},
	"git_diff": {
		Name:        "git_diff",
		Description: "Shows git diff for a specific file or all files.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to diff (optional, shows all if empty)"},
			},
		},
	},
	"git_log": {
		Name:        "git_log",
		Description: "Shows recent git commit history.",
		Parameters:  nil,
	},
	"git_add": {
		Name:        "git_add",
		Description: "Stages files for commit.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to stage (use '.' for all)"},
			},
			Required: []string{"path"},
		},
	},
	"git_commit": {
		Name:        "git_commit",
		Description: "Creates a git commit with the specified message.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"message": {Type: "string", Description: "Commit message"},
			},
			Required: []string{"message"},
		},
	},
	"git_push": {
		Name:        "git_push",
		Description: "Pushes commits to the remote repository.",
		Parameters:  nil,
	},
	"git_branch": {
		Name:        "git_branch",
		Description: "Manages git branches (list, create, delete).",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"action":      {Type: "string", Description: "Action: list, create, delete", Enum: []string{"list", "create", "delete"}},
				"branch_name": {Type: "string", Description: "Branch name (required for create/delete)"},
			},
			Required: []string{"action"},
		},
	},
	"git_checkout": {
		Name:        "git_checkout",
		Description: "Switches branches or restores files.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"target": {Type: "string", Description: "Branch name or file path to checkout"},
			},
			Required: []string{"target"},
		},
	},
	"git_stash": {
		Name:        "git_stash",
		Description: "Manages git stash (save, list, pop, apply, drop).",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"action":  {Type: "string", Description: "Action: save, list, pop, apply, drop", Enum: []string{"save", "list", "pop", "apply", "drop"}},
				"message": {Type: "string", Description: "Stash message (for save action)"},
			},
			Required: []string{"action"},
		},
	},

	// ===== Search Operations =====
	"search_code": {
		Name:        "search_code",
		Description: "Searches for a pattern in source code files using ripgrep.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"pattern": {Type: "string", Description: "Search pattern (regex supported)"},
				"path":    {Type: "string", Description: "Directory or file to search in (optional)"},
			},
			Required: []string{"pattern"},
		},
	},
	"search_file": {
		Name:        "search_file",
		Description: "Searches for files by name pattern using find.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"pattern": {Type: "string", Description: "File name pattern (glob or partial name)"},
				"path":    {Type: "string", Description: "Directory to search in (optional)"},
			},
			Required: []string{"pattern"},
		},
	},
	"web_search": {
		Name:        "web_search",
		Description: "Searches the web for information using Serper API.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"query": {Type: "string", Description: "Search query"},
			},
			Required: []string{"query"},
		},
	},
	"ast_grep": {
		Name:        "ast_grep",
		Description: "Performs structural code search using AST patterns (Tree-sitter based).",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
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
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
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
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "Test file or directory path (optional)"},
			},
		},
	},
	"format": {
		Name:        "format",
		Description: "Formats source code file (go fmt, prettier, etc.).",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path": {Type: "string", Description: "File path to format"},
			},
			Required: []string{"path"},
		},
	},
	"lint": {
		Name:        "lint",
		Description: "Runs linter on the specified file with optional auto-fix.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"path":     {Type: "string", Description: "File path to lint"},
				"auto_fix": {Type: "string", Description: "Set to 'true' to auto-fix issues"},
			},
			Required: []string{"path"},
		},
	},
	"diff_files": {
		Name:        "diff_files",
		Description: "Compares two files and shows differences.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"file1":   {Type: "string", Description: "First file path"},
				"file2":   {Type: "string", Description: "Second file path"},
				"context": {Type: "string", Description: "Number of context lines (default: 3)"},
			},
			Required: []string{"file1", "file2"},
		},
	},
	"http_request": {
		Name:        "http_request",
		Description: "Executes HTTP requests to external APIs.",
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
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
		Parameters: &GeminiParameterSchema{
			Type: "object",
			Properties: map[string]GeminiPropertyDef{
				"command": {Type: "string", Description: "Shell command to execute"},
			},
			Required: []string{"command"},
		},
	},
}

// GetGeminiToolDefinitions returns all tool definitions for Function Calling API
func GetGeminiToolDefinitions() []GeminiToolConfig {
	declarations := make([]GeminiFunctionDeclaration, 0, len(toolDefinitions))
	for _, def := range toolDefinitions {
		declarations = append(declarations, def)
	}
	return []GeminiToolConfig{{FunctionDeclarations: declarations}}
}

// GetToolDefinitionNames returns all defined tool names for testing
func GetToolDefinitionNames() []string {
	names := make([]string, 0, len(toolDefinitions))
	for name := range toolDefinitions {
		names = append(names, name)
	}
	return names
}

// GeminiFunctionCall - Function Call response from Gemini
type GeminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// convertFunctionCallToToolJSON converts Gemini FunctionCall to internal tool JSON format
// Returns: {"tool": "read_file", "args": {"path": "/path/to/file"}}
func convertFunctionCallToToolJSON(fc *GeminiFunctionCall) string {
	toolCall := map[string]any{
		"tool": fc.Name,
		"args": fc.Args,
	}
	jsonBytes, _ := json.Marshal(toolCall)
	return string(jsonBytes)
}
