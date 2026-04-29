package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const azureDefaultPlaceholderDeployment = "azure-gpt-5.4"

// SwitchProvider はプロバイダーを切り替える
func (a *Agent) SwitchProvider(providerName string) error {
	return a.switchProvider(providerName, "")
}

func (a *Agent) switchProvider(providerName, requestedModel string) error {
	out := a.output()
	errOut := a.errorOutput()
	requestedProviderName := providerName
	modelLookupProviderName := config.ActiveProviderConfigKey(providerName)
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
	nextProviderConfigKey := modelLookupProviderName
	if aware, ok := provider.(providerConfigKeyAware); ok {
		if key := config.ActiveProviderConfigKey(aware.ProviderConfigKey()); key != "" {
			nextProviderConfigKey = key
		}
	}
	if nextProviderConfigKey == "" {
		nextProviderConfigKey = runtimeProviderName
	}

	// runtime 設定から新しいプロバイダーのデフォルトモデルを取得
	cfg := a.cfg()
	newModel := cfg.GetSelectedModelForProvider(nextProviderConfigKey)
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel != "" {
		newModel = requestedModel
	}
	if err := validateProviderModelSelection(cfg, runtimeProviderName, nextProviderConfigKey, newModel, requestedModel != ""); err != nil {
		return err
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
	if oldProvider != "" && !config.SameProviderRuntimeIdentity(oldProvider, runtimeProviderName) {
		// プロバイダー切り替え時は tool_calls のフォーマットが互換でない場合があるため、履歴を破棄する
		hadConversation := a.hasConversationState()
		if err := a.resetConversationState(); err != nil {
			return fmt.Errorf("failed to reset conversation state during provider switch: %w", err)
		}
		if hadConversation {
			yellow.Fprintln(out, "🗑️  History cleared after provider switch to avoid incompatible tool-call history")
		}
	}
	a.CurrentProvider = provider
	a.ProviderName = runtimeProviderName
	a.ProviderConfigKey = nextProviderConfigKey
	a.setCurrentModel(newModel)

	// 統計情報をリセット（プロバイダー切り替え時）
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.Provider = runtimeProviderName
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
			a.Stats.AddUsageForConfig(a.cfg(), u)
		})
	}

	// MCPToolProviderインターフェースを実装するプロバイダーにMCPツールを設定
	if a.mcpManager != nil {
		configureMCPTools(provider, a.mcpManager.GetTools(), errOut)
	}

	a.syncCurrentDerivedRuntimeState()

	green.Fprintf(out, "✅ Provider: %s → %s\n", oldProvider, runtimeProviderName)
	green.Fprintf(out, "✅ Model: %s → %s\n", oldModel, newModel)
	return nil
}

func validateProviderModelSelection(cfg *config.Config, runtimeProviderName, providerConfigKey, model string, explicitModel bool) error {
	if config.CanonicalProviderName(runtimeProviderName) != "azure" {
		return nil
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("azure OpenAI deployment is required: set provider_models.azure.default_model or use '/use azure <deployment>'")
	}
	if !strings.EqualFold(model, azureDefaultPlaceholderDeployment) {
		return nil
	}
	if explicitModel {
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("azure OpenAI deployment is required: set provider_models.azure.default_model or use '/use azure <deployment>'")
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel(providerConfigKey)); explicit != "" {
		return nil
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("azure")); explicit != "" {
		return nil
	}
	return fmt.Errorf(
		"azure OpenAI deployment is not configured. Set provider_models.azure.default_model or run '/use azure <deployment>'",
	)
}

// IsAPIKeyAvailable は指定されたプロバイダーのAPIキーが利用可能かチェック
func IsAPIKeyAvailable(provider string) bool {
	return config.ProviderHasAvailableCredential(provider)
}
