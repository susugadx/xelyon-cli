package mcpnames

import "strings"

// SanitizePart は MCP exported tool name に使う server/tool 名の安全な断片を返す。
func SanitizePart(name string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)
}

// ExportedToolName は provider/prompt/registry で使う MCP tool 名を返す。
func ExportedToolName(serverName, toolName string) string {
	return "mcp_" + SanitizePart(serverName) + "_" + SanitizePart(toolName)
}

// IsExportedToolName は provider/prompt/registry で使う MCP tool 名かを返す。
func IsExportedToolName(name string) bool {
	return strings.HasPrefix(name, "mcp_")
}
