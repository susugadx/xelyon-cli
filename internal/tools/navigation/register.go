package navigation

import (
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// InspectSymbolTool は inspect_symbol のツールラッパー（テスト専用）。
// production では search_code が navigation.InspectSymbolAuto を直接呼ぶため、この型は使用されない。
type InspectSymbolTool struct{}

// Name はツール名を返す。
func (t *InspectSymbolTool) Name() string { return "inspect_symbol" }

// Description はツールの説明を返す。
func (t *InspectSymbolTool) Description() string {
	return "Internal: Go symbol lookup used by search_code fast path."
}

// Parameters はツールのパラメータ定義を返す。
func (t *InspectSymbolTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"symbol": map[string]interface{}{
				"type":        "string",
				"description": "Exact symbol name to inspect (for example: Build, Config.Build, (*Config).Build)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Optional file or directory path to narrow candidates",
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
	result := navigation.InspectSymbol(symbol, path, "full")
	return result, nil, nil
}

// RegisterTools は inspect_symbol をレジストリに登録する（テスト用）。
// 公開ツールとしては廃止済みのため DefaultRegistry には登録しない。
func RegisterTools(registry *tools.Registry) {
	registry.Register(&InspectSymbolTool{})
}
