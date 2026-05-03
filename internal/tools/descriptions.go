package tools

import "github.com/susugadx/xelyon-cli/internal/toolmeta"

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// GetToolDefinitions() 経由で JSON schema の description に使用。
var ToolDescriptions = toolmeta.DescriptionMap()

// ToolDescription は指定ツールの説明文を返す。
// 未定義ツールは空文字を返す。
func ToolDescription(name string) string {
	spec, ok := toolmeta.Lookup(name)
	if !ok {
		return ""
	}
	return spec.Description
}
