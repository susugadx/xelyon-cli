package search

import (
	"strconv"

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
func (t *WebSearchTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteWebSearch(execCtx, args["query"])
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
			"pattern":         map[string]interface{}{"type": "string", "description": "Search pattern (regex). Comma-separated for parallel multi-search (e.g. 'handleSSE,parseResponse')"},
			"path":            map[string]interface{}{"type": "string", "description": "Directory or file to search in (default: current directory)"},
			"file_pattern":    map[string]interface{}{"type": "string", "description": "File glob pattern to filter (e.g., *.go, *.ts)"},
			"file_type":       map[string]interface{}{"type": "string", "description": "File type filter using rg --type (e.g., go, py, js, ts). Preferred over file_pattern for known languages."},
			"is_regex":        map[string]interface{}{"type": "boolean", "description": "Whether pattern is regex (default: true). Set false for literal string search."},
			"multiline":       map[string]interface{}{"type": "boolean", "description": "Enable multiline matching for struct/interface/multi-line import search."},
			"include_hidden":  map[string]interface{}{"type": "boolean", "description": "Include hidden files (dotfiles) in search. Default: false."},
			"include_ignored": map[string]interface{}{"type": "boolean", "description": "Include files ignored by .gitignore. Default: false."},
			"context_lines":   map[string]interface{}{"type": "integer", "description": "Number of context lines around matches (default: 3, max: 10)"},
			"output_mode":     map[string]interface{}{"type": "string", "description": "Output mode: omit or use 'full' for grouped code snippets, or 'manifest' for files and match counts only"},
		},
		"required":             []string{"pattern"},
		"additionalProperties": false,
	}
}

func (t *SearchCodeTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	opts := SearchOptions{
		Pattern:     args["pattern"],
		Path:        args["path"],
		FilePattern: args["file_pattern"],
		FileType:    args["file_type"],
		CtxLines:    -1,
		IsRegex:     true, // デフォルト true
	}

	if args["is_regex"] != "" {
		if b, err := strconv.ParseBool(args["is_regex"]); err == nil {
			opts.IsRegex = b
		}
	}

	if args["multiline"] != "" {
		if b, err := strconv.ParseBool(args["multiline"]); err == nil {
			opts.Multiline = b
		}
	}
	if args["include_hidden"] != "" {
		if b, err := strconv.ParseBool(args["include_hidden"]); err == nil {
			opts.IncludeHidden = b
		}
	}
	if args["include_ignored"] != "" {
		if b, err := strconv.ParseBool(args["include_ignored"]); err == nil {
			opts.IncludeIgnored = b
		}
	}
	switch args["output_mode"] {
	case "manifest":
		opts.OutputMode = "manifest"
	case "full":
		opts.OutputMode = "full"
		// default: OutputMode stays "" for grouped code snippets
	}

	if args["context_lines"] != "" {
		if n, err := strconv.Atoi(args["context_lines"]); err == nil {
			opts.CtxLines = n
		}
	}

	result := ExecuteSearchCodeWithConfig(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), opts)
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
