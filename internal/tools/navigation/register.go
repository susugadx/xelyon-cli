package navigation

import (
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// InspectSymbolTool は inspect_symbol ツール。
type InspectSymbolTool struct{}

// Name はツール名を返す。
func (t *InspectSymbolTool) Name() string { return "inspect_symbol" }

// Description はツールの説明を返す。
func (t *InspectSymbolTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

// Parameters はツールのパラメータ定義を返す。
func (t *InspectSymbolTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"symbol": map[string]interface{}{
				"type":        "string",
				"description": "Exact symbol name to inspect (function, type, method, variable, constant)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Optional file or directory path to narrow candidates",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Output mode: 'summary' (default, compact) or 'full' (expanded limits)",
			},
		},
		"required":             []string{"symbol"},
		"additionalProperties": false,
	}
}

// Run はツールを実行する。
func (t *InspectSymbolTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	symbol := args["symbol"]
	path := args["path"]
	mode := args["mode"]

	result := navigation.InspectSymbol(symbol, path, mode)
	return result, nil, nil
}

// RegisterTools は navigation ツールをレジストリに登録する。
func RegisterTools(registry *tools.Registry) {
	registry.Register(&InspectSymbolTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
