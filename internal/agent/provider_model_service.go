package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// ProviderModelState は現在の provider/model 選択状態の read-only snapshot。
type ProviderModelState struct {
	CurrentProvider   string
	ProviderConfigKey string
	CurrentModel      string
}

// ModelSwitchOutcome は current provider 内の model 切り替え結果。
type ModelSwitchOutcome struct {
	OldModel      string
	NewModel      string
	ConfigSaved   bool
	ValidationErr error
	LoadConfigErr error
	SaveConfigErr error
}

// ProviderSwitchOutcome は provider と model の session 切り替え結果。
type ProviderSwitchOutcome struct {
	RequestedProvider    string
	OldProvider          string
	NewProvider          string
	OldProviderConfigKey string
	NewProviderConfigKey string
	OldModel             string
	NewModel             string
	HistoryCleared       bool
}

// CurrentProviderModelState は現在の provider/model 選択状態を返す。
func (a *Agent) CurrentProviderModelState() ProviderModelState {
	if a == nil {
		return ProviderModelState{}
	}
	return ProviderModelState{
		CurrentProvider:   a.ProviderName,
		ProviderConfigKey: a.currentProviderConfigKey(),
		CurrentModel:      a.CurrentModel,
	}
}

// SwitchModelForCurrentProvider は current provider の model を切り替え、
// default_model と provider_models の保存同期を試みる。表示は呼び出し側が担う。
func (a *Agent) SwitchModelForCurrentProvider(newModel string) ModelSwitchOutcome {
	outcome := ModelSwitchOutcome{NewModel: newModel}
	if a == nil {
		return outcome
	}

	outcome.OldModel = a.CurrentModel
	if err := validateProviderModelSelection(a.cfg(), a.ProviderName, a.currentProviderConfigKey(), newModel, true); err != nil {
		outcome.ValidationErr = err
		return outcome
	}

	cfg, err := loadConfigForCommand()
	if err != nil {
		a.clearCurrentProviderCache()
		a.setCurrentModelAndSync(newModel)
		outcome.LoadConfigErr = err
		return outcome
	}

	cfg.DefaultModel = newModel
	providerKey := a.syncDefaultModelToProviderForModel(cfg, newModel)
	if err := validateProviderModelSelection(cfg, a.ProviderName, providerKey, newModel, true); err != nil {
		outcome.ValidationErr = err
		return outcome
	}
	if err := validateGeminiFunctionCallingConfigForSaveIfRelevant(cfg, a.ProviderName, cfg.DefaultProvider); err != nil {
		outcome.ValidationErr = err
		return outcome
	}

	a.clearCurrentProviderCache()
	a.setCurrentModelAndSync(newModel)

	if err := saveConfigForCommand(cfg); err != nil {
		outcome.SaveConfigErr = err
		return outcome
	}

	a.setRuntimeConfig(cfg)
	outcome.ConfigSaved = true
	return outcome
}

// ConfigureAzureDeployment は Azure deployment と catalog_model を config に保存する。
// session/provider の切り替えは呼び出し側が SwitchProviderModel で明示する。
func (a *Agent) ConfigureAzureDeployment(deployment string, catalogModel string) error {
	cfg, _, _, err := azureDeploymentConfigForCommand(deployment, catalogModel)
	if err != nil {
		return err
	}

	if err := saveConfigForCommand(cfg); err != nil {
		return err
	}
	if a != nil {
		a.setRuntimeConfig(cfg)
	}
	return nil
}

// ConfigureAndSwitchAzureDeployment は Azure setup を検証し、
// provider switch が成功した場合だけ config に保存する。
func (a *Agent) ConfigureAndSwitchAzureDeployment(deployment string, catalogModel string) (ProviderSwitchOutcome, error) {
	cfg, deployment, _, err := azureDeploymentConfigForCommand(deployment, catalogModel)
	if err != nil {
		return ProviderSwitchOutcome{}, err
	}

	outcome, err := a.switchProviderModelWithConfig("azure", deployment, cfg)
	if err != nil {
		return outcome, err
	}

	if a != nil {
		a.setRuntimeConfig(cfg)
	}
	if err := saveConfigForCommand(cfg); err != nil {
		return outcome, err
	}
	return outcome, nil
}

func azureDeploymentConfigForCommand(deployment string, catalogModel string) (*config.Config, string, string, error) {
	deployment = strings.TrimSpace(deployment)
	catalogModel = strings.TrimSpace(catalogModel)
	if deployment == "" {
		return nil, "", "", fmt.Errorf("azure deployment is required")
	}
	if catalogModel == "" {
		return nil, "", "", fmt.Errorf("azure catalog_model is required")
	}

	cfg, err := loadConfigForCommand()
	if err != nil {
		return nil, "", "", err
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	applyAzureDeploymentConfig(cfg, deployment, catalogModel)
	return cfg, deployment, catalogModel, nil
}

func applyAzureDeploymentConfig(cfg *config.Config, deployment string, catalogModel string) {
	cfg.DefaultProvider = "azure"
	cfg.DefaultModel = deployment
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: deployment,
		CatalogModel: catalogModel,
	})
	syncAzureDeploymentOverrideCatalog(cfg, deployment, catalogModel)
}

func syncAzureDeploymentOverrideCatalog(cfg *config.Config, deployment string, catalogModel string) {
	if cfg == nil {
		return
	}
	override, ok := cfg.ModelOverrideForProvider("azure", deployment)
	if !ok {
		return
	}
	override.CatalogModel = catalogModel
	_ = cfg.PatchProviderModelConfig("azure", func(pm *config.ProviderModelConfig) {
		if pm.ModelOverrides == nil {
			pm.ModelOverrides = map[string]config.ModelOverride{}
		}
		pm.ModelOverrides[deployment] = override
	})
}

// SwitchProviderModel は provider と optional model を session に適用する。
// config 保存と user-facing 表示は行わない。
func (a *Agent) SwitchProviderModel(providerName, requestedModel string) (ProviderSwitchOutcome, error) {
	return a.switchProviderModelWithConfig(providerName, requestedModel, a.cfg())
}

func (a *Agent) switchProviderModelWithConfig(providerName, requestedModel string, cfg *config.Config) (ProviderSwitchOutcome, error) {
	outcome := ProviderSwitchOutcome{RequestedProvider: providerName}
	if a == nil {
		return outcome, fmt.Errorf("agent is nil")
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	outcome.OldProvider = a.ProviderName
	outcome.OldProviderConfigKey = a.currentProviderConfigKey()
	outcome.OldModel = a.CurrentModel

	requestedProviderName := providerName
	modelLookupProviderName := config.ActiveProviderConfigKey(providerName)
	runtimeProviderName := config.CanonicalProviderName(providerName)
	if runtimeProviderName == "" {
		return outcome, fmt.Errorf("unknown provider: %s", requestedProviderName)
	}
	if modelLookupProviderName == "" {
		modelLookupProviderName = runtimeProviderName
	}

	// API キー存在チェック
	if !IsAPIKeyAvailable(runtimeProviderName) {
		return outcome, fmt.Errorf("%s のAPIキーが設定されていません", requestedProviderName)
	}

	// プロバイダーインスタンス作成
	provider, err := api.NewProvider(modelLookupProviderName)
	if err != nil {
		return outcome, fmt.Errorf("プロバイダーの初期化に失敗しました: %w", err)
	}
	api.ApplyRuntimeConfig(provider, cfg)
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
	newModel := cfg.GetSelectedModelForProvider(nextProviderConfigKey)
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel != "" {
		newModel = requestedModel
	}
	if err := validateProviderModelSelection(cfg, runtimeProviderName, nextProviderConfigKey, newModel, requestedModel != ""); err != nil {
		return outcome, err
	}

	a.clearCurrentProviderCache()

	if outcome.OldProvider != "" && !config.SameProviderRuntimeIdentity(outcome.OldProvider, runtimeProviderName) {
		// プロバイダー切り替え時は tool_calls のフォーマットが互換でない場合があるため、履歴を破棄する
		hadConversation := a.hasConversationState()
		if err := a.resetConversationState(); err != nil {
			return outcome, fmt.Errorf("failed to reset conversation state during provider switch: %w", err)
		}
		outcome.HistoryCleared = hadConversation
	}

	a.CurrentProvider = provider
	a.ProviderName = runtimeProviderName
	a.ProviderConfigKey = nextProviderConfigKey
	a.setCurrentModel(newModel)

	// 統計情報をリセット（プロバイダー切り替え時）
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.ResetUsageForProvider(runtimeProviderName, newModel)
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
		configureMCPTools(provider, a.mcpManager.GetTools(), a.errorOutput())
	}

	a.syncCurrentDerivedRuntimeState()

	outcome.NewProvider = runtimeProviderName
	outcome.NewProviderConfigKey = nextProviderConfigKey
	outcome.NewModel = newModel
	return outcome, nil
}

func (a *Agent) clearCurrentProviderCache() {
	if a == nil || a.CurrentProvider == nil {
		return
	}
	if cacheClearable, ok := a.CurrentProvider.(api.CacheClearable); ok {
		cacheClearable.ClearCache()
	}
}
