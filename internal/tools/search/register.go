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
