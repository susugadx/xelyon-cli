package agent

import (
	"fmt"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/mcptool"
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

func mcpToolDefinitions(mcpTools []mcp.MCPTool) []mcptool.Definition {
	defs := make([]mcptool.Definition, 0, len(mcpTools))
	for _, tool := range mcpTools {
		defs = append(defs, mcptool.Definition{
			ServerName:  tool.ServerName,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return defs
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

	providerName := config.CanonicalProviderName(provider.Name())
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
	seen := make(map[string]bool, len(mcpTools))
	for _, t := range mcpTools {
		// ツール名: mcp_{serverName}_{toolName}
		name := mcpnames.ExportedToolName(t.ServerName, t.Name)
		if seen[name] {
			continue
		}
		seen[name] = true

		if debugEnabled && errOut != nil {
			_, _ = fmt.Fprintf(errOut, "[DEBUG %s] MCP tool registered: %s\n", debugLabel, name)
		}

		// mcp.MCPTool → api.ToolDefinition 変換は api.ConvertMCPToolToToolDefinition のみを使用
		toolDefs = append(toolDefs, api.ConvertMCPToolToToolDefinition(name, t.Description, t.InputSchema))
	}

	// SetMCPTools は 1 回だけ呼ぶ
	mcpProvider.SetMCPTools(toolDefs)
}
