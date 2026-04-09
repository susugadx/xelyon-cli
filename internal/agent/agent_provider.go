package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// SwitchProvider はプロバイダーを切り替える
func (a *Agent) SwitchProvider(providerName string) error {
	out := a.output()
	errOut := a.errorOutput()
	requestedProviderName := providerName
	modelLookupProviderName := config.NormalizeProviderName(providerName)
	runtimeProviderName := config.CanonicalProviderName(providerName)
	if runtimeProviderName == "" {
		return fmt.Errorf("unknown provider: %s", requestedProviderName)
	}
	if modelLookupProviderName == "" {
		modelLookupProviderName = runtimeProviderName
	}

	// API キー存在チェック
	if !IsAPIKeyAvailable(runtimeProviderName) {
		return fmt.Errorf("%s のAPIキーが設定されていません", requestedProviderName)
	}

	// プロバイダーインスタンス作成
	provider, err := api.NewProvider(modelLookupProviderName)
	if err != nil {
		return fmt.Errorf("プロバイダーの初期化に失敗しました: %w", err)
	}
	api.ApplyRuntimeConfig(provider, a.cfg())
	api.ApplyUIRuntime(provider, a.ui())

	// runtime 設定から新しいプロバイダーのデフォルトモデルを取得
	cfg := a.cfg()
	newModel := cfg.GetSelectedModelForProvider(modelLookupProviderName)

	// 既存プロバイダーのキャッシュをクリア（サポートしている場合）
	if a.CurrentProvider != nil {
		if cacheClearable, ok := a.CurrentProvider.(api.CacheClearable); ok {
			cacheClearable.ClearCache()
		}
	}

	// プロバイダー切り替え
	oldProvider := a.ProviderName
	oldModel := a.CurrentModel
	if oldProvider != "" && !config.SameProviderRuntimeIdentity(oldProvider, runtimeProviderName) {
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
	a.ProviderName = runtimeProviderName
	a.ProviderConfigKey = modelLookupProviderName
	a.CurrentModel = newModel
	a.syncSessionModel()

	// 統計情報をリセット（プロバイダー切り替え時）
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.Provider = runtimeProviderName
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

	green.Fprintf(out, "✅ Provider: %s → %s\n", oldProvider, runtimeProviderName)
	green.Fprintf(out, "✅ Model: %s → %s\n", oldModel, newModel)
	return nil
}

// IsAPIKeyAvailable は指定されたプロバイダーのAPIキーが利用可能かチェック
func IsAPIKeyAvailable(provider string) bool {
	switch config.CanonicalProviderName(provider) {
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
