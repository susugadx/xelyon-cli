package tools

import "github.com/susugadx/xelyon-cli/internal/toolmeta"

// ToolDescriptions は全ビルトインツールの Description を一元管理する。
// GetToolDefinitions() 経由で JSON schema の description に使用。
var ToolDescriptions = toolmeta.DescriptionMap()
