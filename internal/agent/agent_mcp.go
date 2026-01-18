package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/mcp"
)

// buildMCPToolsPrompt はMCPツール用のシステムプロンプトを構築する
func buildMCPToolsPrompt(mcpManager *mcp.Manager) string {
	mcpTools := mcpManager.GetTools()
	if len(mcpTools) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## MCP Tools (External Integrations)\n")
	sb.WriteString("These tools connect to external services. **USE THEM when the user's request matches their capabilities.**\n")
	sb.WriteString("Do NOT say \"I cannot access this service\" - you CAN via these MCP tools.\n\n")

	// サーバーごとにグループ化
	serverTools := make(map[string][]mcp.MCPTool)
	for _, t := range mcpTools {
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
	for serverName, tools := range serverTools {
		sb.WriteString(fmt.Sprintf("### %s Server\n", serverName))
		for _, t := range tools {
			toolName := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(serverName), sanitizeToolName(t.Name))
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", toolName, t.Description))
		}
		sb.WriteString("\n")
	}

	// GitHub専用ガイドを追加
	if hasGitHub {
		sb.WriteString(buildGitHubMCPGuide())
	}

	return sb.String()
}

// sanitizeToolName はツール名から特殊文字を除去する
func sanitizeToolName(name string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)
}

// buildGitHubMCPGuide はGitHub MCP専用の使用ガイドを生成する
func buildGitHubMCPGuide() string {
	return `### GitHub MCP Usage Guide (IMPORTANT)

**Tool Call Format:**
` + "```" + `
{"tool": "mcp_github_<action>", "args": {"owner": "...", "repo": "...", ...}}
` + "```" + `

**Common Tools & Arguments:**

| Tool | Required Args | Description |
|------|---------------|-------------|
| mcp_github_get_issue | owner, repo, issue_number | Get issue details |
| mcp_github_list_issues | owner, repo | List all issues |
| mcp_github_create_issue | owner, repo, title, body | Create new issue |
| mcp_github_get_file_contents | owner, repo, path, branch(opt) | Get file content |
| mcp_github_list_pull_requests | owner, repo | List all PRs |
| mcp_github_create_pull_request | owner, repo, title, body, head, base | Create PR |
| mcp_github_search_code | q | Search code (q="keyword repo:owner/repo") |
| mcp_github_get_workflow_runs | owner, repo | Check CI status |

**Example:**
` + "```" + `
{"tool": "mcp_github_get_issue", "args": {"owner": "susugadx", "repo": "xelyon-cli", "issue_number": "72"}}
` + "```" + `

**CRITICAL RULES:**
1. ALWAYS use MCP tools for GitHub operations - you have direct access
2. Do NOT say "I can't access GitHub" or "Please use the GitHub web UI"
3. **INFORMATION-ONLY requests**: When user asks to "get", "show", "list", or "fetch" something, ONLY display the result. Do NOT automatically start implementing/fixing based on the fetched content
4. **Wait for explicit instruction**: After showing fetched information, wait for user to tell you what to do next
5. If a tool fails, report the error and suggest alternatives

`
}

// detectGitHubIntent はユーザー入力にGitHub関連キーワードがあるか検出する
func detectGitHubIntent(input string) bool {
	keywords := []string{
		"issue", "イシュー", "issues",
		"pull request", "pr", "プルリクエスト", "プルリク",
		"actions", "アクション", "ci", "workflow", "ワークフロー",
		"github", "ギットハブ", "gh",
		"repository", "リポジトリ", "repo",
	}
	lower := strings.ToLower(input)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// HasGitHubMCP はGitHub MCPサーバーが接続されているか確認する
func (a *Agent) HasGitHubMCP() bool {
	if a.mcpManager == nil {
		return false
	}
	for _, t := range a.mcpManager.GetTools() {
		if strings.Contains(strings.ToLower(t.ServerName), "github") {
			return true
		}
	}
	return false
}

// AddGitHubHint はGitHub関連リクエストにシステムヒントを追加する
func (a *Agent) AddGitHubHint(input string) string {
	if !a.HasGitHubMCP() {
		return input
	}
	if detectGitHubIntent(input) {
		return input + "\n\n[SYSTEM HINT: Use MCP GitHub tools for this request. Do NOT suggest using the web UI.]"
	}
	return input
}
