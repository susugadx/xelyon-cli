package search

import (
	"os"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
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
			"pattern": map[string]interface{}{"type": "string", "description": "Search query. Comma-separated patterns are the default for related code discovery. Prefer one combined multi-pattern query over repeated narrow searches. Examples: 'planSearchRoute,searchRouteTrace,router_test.go', 'VisualRow,syncCursorToViewportTop,charSelectionColumnsForLine'. Max 10 patterns."},
			"intent": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"impact"},
				"description": "Optional high-level intent. intent=impact expands a single pattern into a conservative related multi-pattern search for shared-change impact analysis. Already comma-separated patterns are used as-is.",
			},
			"path":        map[string]interface{}{"type": "string", "description": "Directory or file to search in (default: current directory)"},
			"file_filter": map[string]interface{}{"type": "string", "description": "Filter by language type (e.g. go, py, ts) or glob pattern (e.g. *_test.go, Dockerfile*). Glob characters (*, ?, [) trigger glob mode, otherwise uses rg --type."},
			"mode": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"auto", "symbol", "literal", "regex"},
				"description": "Search mode. Default auto routes symbol-like queries to symbol resolution when available, otherwise to literal or regex search.",
			},
			"is_regex": map[string]interface{}{
				"type":        "boolean",
				"description": "Legacy compatibility flag. true maps to mode=regex, false maps to mode=literal when mode is omitted.",
			},
		},
		"required":             []string{"pattern"},
		"additionalProperties": false,
	}
}

func (t *SearchCodeTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	opts := SearchOptions{
		Pattern: args["pattern"],
		Intent:  args["intent"],
		Path:    args["path"],
		Mode:    string(SearchModeAuto),
	}
	if filter := args["file_filter"]; filter != "" {
		if containsGlobChar(filter) {
			opts.FilePattern = filter
		} else {
			opts.FileType = filter
		}
	}

	if mode := args["mode"]; mode != "" {
		opts.Mode = mode
	}

	if args["is_regex"] != "" {
		if b, err := strconv.ParseBool(args["is_regex"]); err == nil {
			opts.IsRegex = b
			opts.LegacyIsRegexSet = true
		}
	}

	opts.LocatorRegistry = execCtx.EffectiveLocatorRegistry()
	opts.ProjectMap = execCtx.ProjectMap
	opts.ProjectMapRootPath = execCtx.ProjectMapRootPath
	opts.ProjectMapStateKey = execCtx.ProjectMapStateKey
	opts.InvocationCWD = execCtx.InvocationCWD
	if lspClient := execCtx.EffectiveLSPClient(); lspClient != nil {
		if cwd := strings.TrimSpace(execCtx.InvocationCWD); cwd != "" {
			opts.LSPClient = navigation.NewLSPAdapter(lspClient, cwd)
		} else if cwd, err := os.Getwd(); err == nil {
			opts.LSPClient = navigation.NewLSPAdapter(lspClient, cwd)
		}
	}
	result := ExecuteSearchCodeWithConfig(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), opts)
	return result, nil, nil
}

// containsGlobChar は文字列に glob 文字（*, ?, [）が含まれるかを返す。
func containsGlobChar(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// RegisterTools registers search tools with the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&WebSearchTool{})
	registry.Register(&SearchCodeTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
