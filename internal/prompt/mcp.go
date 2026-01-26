package prompt

import (
	"fmt"
	"strings"
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
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)
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
		sb.WriteString(fmt.Sprintf("### %s Server\n", serverName))
		for _, t := range serverToolList {
			toolName := fmt.Sprintf("mcp_%s_%s", SanitizeToolName(serverName), SanitizeToolName(t.Name))
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", toolName, t.Description))
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
	return `### GitHub MCP Usage Guide (IMPORTANT)

**CONTEXT INFERENCE (NEVER ask "which repository?"):**
- Infer owner/repo from: git remote, XELYON.md, directory name
- Current repository context is always available - use it
- Only ask if genuinely ambiguous (e.g., cross-repo operations)

**Tool Call Format:**
` + "```" + `
{"tool": "mcp_github_<action>", "args": {"owner": "...", "repo": "...", ...}}
` + "```" + `

**Common Tools & Arguments:**

| Tool | Required Args | Array Args (use [] not string) |
|------|---------------|--------------------------------|
| mcp_github_get_issue | owner, repo, issue_number | - |
| mcp_github_list_issues | owner, repo | labels: ["bug"] |
| mcp_github_create_issue | owner, repo, title, body | labels: ["enhancement"], assignees: ["user"] |
| mcp_github_update_issue | owner, repo, issue_number | labels: [], assignees: [] |
| mcp_github_get_file_contents | owner, repo, path | - |
| mcp_github_list_pull_requests | owner, repo | - |
| mcp_github_create_pull_request | owner, repo, title, body, head, base | - |
| mcp_github_search_code | q | - |
| mcp_github_get_workflow_runs | owner, repo | - |

**Array Argument Examples:**
` + "```" + `json
// ✅ CORRECT - labels/assignees must be arrays
{"tool": "mcp_github_create_issue", "args": {"owner": "o", "repo": "r", "title": "Bug", "body": "...", "labels": ["bug", "priority"], "assignees": ["user"]}}

// ❌ WRONG - labels must be array, not string
{"tool": "mcp_github_create_issue", "args": {"labels": "bug"}}
` + "```" + `

**RESPONSE STYLE:**
- Be helpful and proactive, not transactional
- Summarize results concisely but informatively (not just raw JSON)
- Suggest relevant follow-up actions when appropriate

**CRITICAL RULES:**
1. ALWAYS use MCP tools for GitHub operations - you have direct access
2. Do NOT say "I can't access GitHub" or "Please use the GitHub web UI"
3. **INFORMATION-ONLY requests**: When user asks to "get", "show", "list", or "fetch", display the result and wait for explicit instruction. Do NOT automatically start implementing or fixing
4. If a tool fails, report the error and suggest alternatives

`
}
