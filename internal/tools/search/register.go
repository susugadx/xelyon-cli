package search

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// SearchCodeTool is the tool for code search
type SearchCodeTool struct{}

// Name returns the tool name
func (t *SearchCodeTool) Name() string {
	return "search_code"
}

func (t *SearchCodeTool) Description() string {
	return "Searches for a pattern in source code files using ripgrep."
}

func (t *SearchCodeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{"type": "string", "description": "Search pattern (regex supported)"},
			"path":    map[string]interface{}{"type": "string", "description": "Directory or file to search in (optional)"},
		},
		"required":             []string{"pattern"},
		"additionalProperties": false,
	}
}

// Run executes the code search tool
func (t *SearchCodeTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteSearchCode(args["pattern"], args["path"])
	return output, nil, nil
}

// SearchFileTool is the tool for file search
type SearchFileTool struct{}

// Name returns the tool name
func (t *SearchFileTool) Name() string {
	return "search_file"
}

func (t *SearchFileTool) Description() string {
	return "Searches for files by name pattern using find."
}

func (t *SearchFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{"type": "string", "description": "File name pattern (glob or partial name)"},
			"path":    map[string]interface{}{"type": "string", "description": "Directory to search in (optional)"},
		},
		"required":             []string{"pattern"},
		"additionalProperties": false,
	}
}

// Run executes the file search tool
func (t *SearchFileTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteSearchFile(args["pattern"], args["path"])
	return output, nil, nil
}

// WebSearchTool is the tool for web search
type WebSearchTool struct{}

// Name returns the tool name
func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Searches the web for information using Serper API."
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "Search query"},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
}

// Run executes the web search tool
func (t *WebSearchTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteWebSearch(args["query"])
	return output, nil, nil
}

// AstGrepTool is the tool for ast-grep search
type AstGrepTool struct{}

// Name returns the tool name
func (t *AstGrepTool) Name() string {
	return "ast_grep"
}

func (t *AstGrepTool) Description() string {
	return "Performs structural code search using AST patterns (Tree-sitter based)."
}

func (t *AstGrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{"type": "string", "description": "AST pattern to search for"},
			"lang":    map[string]interface{}{"type": "string", "description": "Language: go, python, javascript, typescript, rust, c, cpp", "enum": []string{"go", "python", "javascript", "typescript", "rust", "c", "cpp"}},
			"path":    map[string]interface{}{"type": "string", "description": "Directory or file to search in (optional)"},
		},
		"required":             []string{"pattern", "lang"},
		"additionalProperties": false,
	}
}

// Run executes the ast-grep tool
func (t *AstGrepTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteAstGrep(args["pattern"], args["lang"], args["path"])
	return output, nil, nil
}

// GrepReplaceTool is the tool for bulk grep replace
type GrepReplaceTool struct{}

// Name returns the tool name
func (t *GrepReplaceTool) Name() string {
	return "grep_replace"
}

func (t *GrepReplaceTool) Description() string {
	return "Performs bulk find-and-replace across multiple files."
}

func (t *GrepReplaceTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
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
	}
}

// Run executes the grep replace tool
func (t *GrepReplaceTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	dryRun := args["dry_run"] == "true"
	if !dryRun {
		yellow.Printf("🔄 Bulk replace across files\n")
		fmt.Printf("  Pattern: %s\n", args["pattern"])
		fmt.Printf("  Replacement: %s\n", args["replacement"])
		fmt.Printf("  Path: %s\n", args["path"])
		fmt.Printf("  File pattern: %s\n", args["file_pattern"])

		decision := common.Confirm("Execute replacement? ")
		if decision.Action == common.ConfirmNo {
			return "Replacement cancelled by user", nil, nil
		}
		if decision.Action == common.ConfirmComment {
			return fmt.Sprintf("Replacement cancelled with comment: %s", decision.Comment), nil, nil
		}
	}
	result, _, err := ExecuteGrepReplace(args["pattern"], args["replacement"], args["path"], args["file_pattern"], dryRun)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}
	return result, nil, nil
}

// RegisterTools registers search tools with the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&SearchCodeTool{})
	registry.Register(&SearchFileTool{})
	registry.Register(&WebSearchTool{})
	registry.Register(&AstGrepTool{})
	registry.Register(&GrepReplaceTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
