package prompt

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/mcpnames"
)

// MCPTool represents an MCP tool for prompt generation.
// This is a simplified struct to avoid circular imports with internal/mcp.
type MCPTool struct {
	ServerName  string
	Name        string
	Description string
}

// SanitizeToolName removes special characters from tool names.
func SanitizeToolName(name string) string {
	return mcpnames.SanitizePart(name)
}

// BuildMCPToolsPrompt generates the MCP tools section for the system prompt.
func BuildMCPToolsPrompt(tools []MCPTool) string {
	if len(tools) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## MCP Tools (External Integrations)\n")
	sb.WriteString("These tools connect to external services. **USE THEM when the user's request matches their capabilities.**\n")
	sb.WriteString("Do NOT say \"I cannot access this service\" - you CAN via these MCP tools.\n\n")

	// サーバーごとにグループ化
	serverTools := make(map[string][]MCPTool)
	for _, t := range tools {
		serverTools[t.ServerName] = append(serverTools[t.ServerName], t)
	}

	// 各サーバーのツールを列挙
	for serverName, serverToolList := range serverTools {
		sb.WriteString("### ")
		sb.WriteString(serverName)
		sb.WriteString(" Server\n")
		for _, t := range serverToolList {
			toolName := mcpnames.ExportedToolName(serverName, t.Name)
			sb.WriteString("- **")
			sb.WriteString(toolName)
			sb.WriteString("**: ")
			sb.WriteString(t.Description)
			sb.WriteByte('\n')
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
