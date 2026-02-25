package search

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// WebSearchTool is the tool for web search
type WebSearchTool struct{}

// Name returns the tool name
func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
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

// SearchCodeTool is the tool for code search
type SearchCodeTool struct{}

func (t *SearchCodeTool) Name() string { return "search_code" }

func (t *SearchCodeTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *SearchCodeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern":       map[string]interface{}{"type": "string", "description": "Search pattern (regex). Comma-separated for parallel multi-search (e.g. 'handleSSE,parseResponse')"},
			"path":          map[string]interface{}{"type": "string", "description": "Directory or file to search in (default: current directory)"},
			"file_pattern":  map[string]interface{}{"type": "string", "description": "File glob pattern to filter (e.g., *.go, *.ts)"},
			"context_lines": map[string]interface{}{"type": "integer", "description": "Number of context lines around matches (default: 3, max: 10)"},
			"token_budget":  map[string]interface{}{"type": "integer", "description": "Approximate token budget for results (default: 3000, max: 6000)"},
		},
		"required":             []string{"pattern"},
		"additionalProperties": false,
	}
}

func (t *SearchCodeTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	result := ExecuteSearchCode(
		args["pattern"], args["path"], args["file_pattern"],
		args["context_lines"], args["token_budget"],
	)
	return result, nil, nil
}

// RegisterTools registers search tools with the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&WebSearchTool{})
	registry.Register(&SearchCodeTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
