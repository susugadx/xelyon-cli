package agent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// SwitchProvider はプロバイダーを切り替える
func (a *Agent) SwitchProvider(providerName string) error {
	// API キー存在チェック
	if !IsAPIKeyAvailable(providerName) {
		return fmt.Errorf("%s のAPIキーが設定されていません", providerName)
	}

	// プロバイダーインスタンス作成
	provider, err := api.NewProvider(providerName)
	if err != nil {
		return fmt.Errorf("プロバイダーの初期化に失敗しました: %w", err)
	}

	// 設定ファイルから新しいプロバイダーのデフォルトモデルを取得
	cfg, err := config.LoadConfig()
	if err != nil {
		yellow.Printf("Warning: Failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}
	newModel := cfg.GetModelForProvider(providerName)
	if newModel == "" {
		// フォールバック: プロバイダー別のハードコードされたデフォルト
		switch providerName {
		case "deepseek":
			newModel = "deepseek-chat"
		case "openai":
			newModel = "gpt-5.2"
		case "gemini":
			newModel = "gemini-3.1-pro-preview"
		case "claude":
			newModel = "claude-sonnet-4-6"
		case "ollama":
			newModel = "qwen2.5-coder:7b"
		case "groq":
			newModel = "meta-llama/llama-4-scout-17b-16e-instruct"
		case "openrouter":
			newModel = "anthropic/claude-sonnet-4.6"
		case "bedrock":
			newModel = "global.anthropic.claude-sonnet-4-6-v1"
		default:
			newModel = "default-model"
		}
	}

	// 既存プロバイダーのキャッシュをクリア（サポートしている場合）
	if a.CurrentProvider != nil {
		if cacheClearable, ok := a.CurrentProvider.(api.CacheClearable); ok {
			cacheClearable.ClearCache()
		}
	}

	// プロバイダー切り替え
	oldProvider := a.ProviderName
	oldModel := a.CurrentModel
	a.CurrentProvider = provider
	a.ProviderName = providerName
	a.CurrentModel = newModel

	// 統計情報をリセット（プロバイダー切り替え時）
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.Provider = providerName
		a.Stats.Model = newModel
		a.Stats.InputTokens = 0
		a.Stats.OutputTokens = 0
		a.Stats.LastUsage = nil
		a.statsMu.Unlock()
	}

	// Usage callback を設定（プロバイダーがサポートしている場合）
	if reporter, ok := provider.(api.UsageReporter); ok {
		reporter.SetUsageCallback(func(u api.Usage) {
			a.statsMu.Lock()
			defer a.statsMu.Unlock()
			a.Stats.AddUsage(u)
		})
	}

	// MCPToolProviderインターフェースを実装するプロバイダーにMCPツールを設定
	if a.mcpManager != nil {
		mcpTools := a.mcpManager.GetTools()
		if len(mcpTools) > 0 {
			// Gemini MCP ツール設定
			if mcpProvider, ok := provider.(api.MCPProvider); ok {
				debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
				var mcpToolsConverted []api.ToolDefinition
				for _, t := range mcpTools {
					name := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(t.ServerName), sanitizeToolName(t.Name))

					// InputSchema を map[string]interface{} に変換
					var params map[string]interface{}
					if len(t.InputSchema) > 0 {
						if err := json.Unmarshal(t.InputSchema, &params); err != nil {
							// 変換失敗時は空のパラメータを使用
							params = nil
						}
					}

					if debug {
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini] MCP tool registered: %s\n", name)
					}
					mcpToolsConverted = append(mcpToolsConverted, api.ToolDefinition{
						Name:        name,
						Description: t.Description,
						Parameters:  params,
					})
				}
				mcpProvider.SetMCPTools(mcpToolsConverted)
			}

			// OpenAI MCP ツール設定
			if openaiMCPProvider, ok := provider.(api.MCPProvider); ok {
				debug := os.Getenv("XELYON_DEBUG_OPENAI") == "1"
				var mcpFunctions []api.ToolDefinition
				for _, t := range mcpTools {
					name := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(t.ServerName), sanitizeToolName(t.Name))
					fn := api.ConvertMCPToolToToolDefinition(name, t.Description, t.InputSchema)
					mcpFunctions = append(mcpFunctions, fn)

					if debug {
						fmt.Fprintf(os.Stderr, "[DEBUG OpenAI] MCP tool registered: %s\n", name)
					}
				}
				openaiMCPProvider.SetMCPTools(mcpFunctions)
			}
		}
	}

	green.Printf("✅ Provider: %s → %s\n", oldProvider, providerName)
	green.Printf("✅ Model: %s → %s\n", oldModel, newModel)
	return nil
}

// IsAPIKeyAvailable は指定されたプロバイダーのAPIキーが利用可能かチェック
func IsAPIKeyAvailable(provider string) bool {
	switch provider {
	case "deepseek":
		return os.Getenv("DEEPSEEK_API_KEY") != ""
	case "openai":
		return os.Getenv("OPENAI_API_KEY") != ""
	case "claude":
		return os.Getenv("ANTHROPIC_API_KEY") != ""
	case "gemini":
		return os.Getenv("GEMINI_API_KEY") != ""
	case "groq":
		return os.Getenv("GROQ_API_KEY") != ""
	case "openrouter":
		return os.Getenv("OPENROUTER_API_KEY") != ""
	case "ollama":
		return true // Ollama はローカルなのでキー不要
	case "bedrock":
		return true // AWS 認証チェーン（IAM ロール等）を許可
	default:
		return false
	}
}
