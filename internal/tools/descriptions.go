package tools

import "github.com/susugadx/xelyon-cli/internal/toolmeta"

// ToolDescriptions は全ビルトインツールの Description snapshot を返す互換 surface。
// 新規 caller は ToolDescription を使い、toolmeta を source of truth として参照する。
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
