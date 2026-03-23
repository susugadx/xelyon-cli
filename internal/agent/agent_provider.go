package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// SwitchProvider はプロバイダーを切り替える
func (a *Agent) SwitchProvider(providerName string) error {
	out := a.output()
	errOut := a.errorOutput()

	// API キー存在チェック
	if !IsAPIKeyAvailable(providerName) {
		return fmt.Errorf("%s のAPIキーが設定されていません", providerName)
	}

	// プロバイダーインスタンス作成
	provider, err := api.NewProvider(providerName)
	if err != nil {
		return fmt.Errorf("プロバイダーの初期化に失敗しました: %w", err)
	}
	api.ApplyRuntimeConfig(provider, a.cfg())
	api.ApplyUIRuntime(provider, a.ui())

	// runtime 設定から新しいプロバイダーのデフォルトモデルを取得
	cfg := a.cfg()
	newModel := cfg.GetModelForProvider(providerName)
	if newModel == "" {
		// フォールバック: プロバイダー別のハードコードされたデフォルト
		switch providerName {
		case "deepseek":
			newModel = "deepseek-chat"
		case "openai":
			newModel = "gpt-5.2"
		case "gemini":
			newModel = "gemini-3.1-pro-preview-customtools"
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
	if oldProvider != "" && oldProvider != providerName {
		// プロバイダー切り替え時は tool_calls のフォーマットが互換でない場合があるため、履歴を破棄する
		a.historyMu.Lock()
		hadHistory := len(a.History) > 0
		a.History = []api.Message{}
		a.historyMu.Unlock()
		if hadHistory {
			yellow.Fprintln(out, "🗑️  History cleared after provider switch to avoid incompatible tool-call history")
		}
	}
	a.CurrentProvider = provider
	a.ProviderName = providerName
	a.CurrentModel = newModel
	a.syncSessionModel()

	// 統計情報をリセット（プロバイダー切り替え時）
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.Provider = providerName
		a.Stats.Model = newModel
		a.Stats.InputTokens = 0
		a.Stats.CachedInputTokens = 0
		a.Stats.CacheCreationTokens = 0
		a.Stats.OutputTokens = 0
		a.Stats.ThinkingTokens = 0
		a.Stats.ToolExecutions = make(map[string]int)
		a.Stats.LastUsage = nil
		a.Stats.LastTurnUsage = nil
		a.Stats.LastTurnCost = 0
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
		configureMCPTools(provider, a.mcpManager.GetTools(), errOut)
	}

	a.rebuildSystemPromptForCurrentProvider()

	green.Fprintf(out, "✅ Provider: %s → %s\n", oldProvider, providerName)
	green.Fprintf(out, "✅ Model: %s → %s\n", oldModel, newModel)
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
