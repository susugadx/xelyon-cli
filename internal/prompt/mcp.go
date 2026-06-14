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

	// GitHub MCPツールがあるかチェック
	hasGitHub := false
	for serverName := range serverTools {
		if strings.Contains(strings.ToLower(serverName), "github") {
			hasGitHub = true
			break
		}
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

	// GitHub専用ガイドを追加
	if hasGitHub {
		sb.WriteString(BuildGitHubMCPGuide())
	}

	return sb.String()
}

// BuildGitHubMCPGuide generates the GitHub MCP usage guide.
func BuildGitHubMCPGuide() string {
	return `### GitHub MCP Usage Guide

**CONTEXT INFERENCE:** Infer owner/repo from git remote, project config, or directory name. NEVER ask "which repository?"

**CRITICAL: Array arguments (labels, assignees) must be [] not string:**
` + "```" + `json
// ✅ CORRECT
{"tool": "mcp_github_create_issue", "args": {"owner": "o", "repo": "r", "title": "Bug", "body": "...", "labels": ["bug"], "assignees": ["user"]}}
// ❌ WRONG
{"tool": "mcp_github_create_issue", "args": {"labels": "bug"}}
` + "```" + `

**RULES:**
- NEVER use MCP tools to inspect or edit local repo files - use the visible local tools in this session instead
- For repo-local investigation, prefer gather_context first. Lower-level local investigation tools are expert overrides only when they are actually exposed.
- ALWAYS use MCP tools for GitHub operations - never say "use GitHub web UI"
- Information-only requests (get, show, list): display result and STOP. Do NOT start implementing
- If a tool fails, report the error and suggest alternatives
`
}
