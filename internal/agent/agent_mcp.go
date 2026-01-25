package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

// buildMCPToolsPrompt はMCPツール用のシステムプロンプトを構築する
func buildMCPToolsPrompt(mcpManager *mcp.Manager) string {
	mcpTools := mcpManager.GetTools()
	if len(mcpTools) == 0 {
		return ""
	}

	// mcp.MCPTool を prompt.MCPTool に変換
	promptTools := make([]prompt.MCPTool, len(mcpTools))
	for i, t := range mcpTools {
		promptTools[i] = prompt.MCPTool{
			ServerName:  t.ServerName,
			Name:        t.Name,
			Description: t.Description,
		}
	}

	return prompt.BuildMCPToolsPrompt(promptTools)
}

// sanitizeToolName はツール名から特殊文字を除去する（prompt.SanitizeToolName のエイリアス）
func sanitizeToolName(name string) string {
	return prompt.SanitizeToolName(name)
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
