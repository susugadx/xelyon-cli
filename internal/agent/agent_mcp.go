package agent

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
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

// configureMCPTools は MCP ツール定義を provider に 1 回だけ登録する（debugは OpenAI / Gemini のみ）
func configureMCPTools(provider api.Provider, mcpTools []mcp.MCPTool, errOut io.Writer) {
	if provider == nil || len(mcpTools) == 0 {
		return
	}

	mcpProvider, ok := provider.(api.MCPProvider)
	if !ok {
		return
	}

	providerName := strings.ToLower(provider.Name())
	debugEnabled := false
	debugLabel := ""

	// provider 名ごとの分岐ポイント（debug ログ設定 / 将来の差し替え入口）
	switch providerName {
	case "openai":
		if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
			debugEnabled = true
			debugLabel = "OpenAI"
		}
	case "gemini":
		if os.Getenv("XELYON_DEBUG_GEMINI") == "1" {
			debugEnabled = true
			debugLabel = "Gemini"
		}
	}

	toolDefs := make([]api.ToolDefinition, 0, len(mcpTools))
	for _, t := range mcpTools {
		// ツール名: mcp_{serverName}_{toolName}
		name := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(t.ServerName), sanitizeToolName(t.Name))

		if debugEnabled && errOut != nil {
			_, _ = fmt.Fprintf(errOut, "[DEBUG %s] MCP tool registered: %s\n", debugLabel, name)
		}

		// mcp.MCPTool → api.ToolDefinition 変換は api.ConvertMCPToolToToolDefinition のみを使用
		toolDefs = append(toolDefs, api.ConvertMCPToolToToolDefinition(name, t.Description, t.InputSchema))
	}

	// SetMCPTools は 1 回だけ呼ぶ
	mcpProvider.SetMCPTools(toolDefs)
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
