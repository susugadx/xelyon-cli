package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/setup"
)

// debugLog はデバッグログを出力（XELYON_DEBUG=1 の場合のみ）
func debugLog(format string, args ...interface{}) {
	if os.Getenv("XELYON_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// resolveProviderName はプロバイダー名を優先順位に従って解決
// 優先順位: CLI flag > 環境変数 > 設定ファイル > デフォルト
func resolveProviderName(flagValue, configValue string) string {
	if normalizedFlag := config.NormalizeProviderName(flagValue); normalizedFlag != "" {
		debugLog("provider from flag: %s -> %s", flagValue, normalizedFlag)
		return normalizedFlag
	}

	if envValue := config.NormalizeProviderName(os.Getenv("XELYON_PROVIDER")); envValue != "" {
		debugLog("provider from env: %s", envValue)
		return envValue
	}

	if normalizedConfig := config.NormalizeProviderName(configValue); normalizedConfig != "" {
		debugLog("provider from config: %s -> %s", configValue, normalizedConfig)
		return normalizedConfig
	}

	debugLog("using default provider: deepseek")
	return "deepseek"
}

// createProvider はプロバイダー名からProviderを生成（テスト可能）
func createProvider(providerName string) (api.Provider, error) {
	name := config.NormalizeProviderName(providerName)

	// api.NewProvider は内部で環境変数をチェックし、プロバイダーを生成する
	return api.NewProvider(name)
}

func resolveRequiredProvider(providerName string) (api.Provider, error) {
	providerName = config.NormalizeProviderName(providerName)
	if providerName == "" {
		providerName = "deepseek"
	}
	if !llmcatalog.IsKnownProvider(providerName) {
		return nil, fmt.Errorf("unknown provider: %s\nSupported providers: %s", providerName, strings.Join(config.GetDisplayProviders(), ", "))
	}
	if setup.ProviderSetupRequired(providerName) {
		return nil, errors.New(setup.ProviderSetupRequiredMessage(providerName))
	}
	provider, err := createProvider(providerName)
	if err != nil {
		return nil, providerCreationError(providerName, err)
	}
	return provider, nil
}

func resolveInteractiveProvider(providerName string) (api.Provider, error) {
	providerName = config.NormalizeProviderName(providerName)
	if providerName == "" {
		providerName = "deepseek"
	}
	if !llmcatalog.IsKnownProvider(providerName) {
		return nil, fmt.Errorf("unknown provider: %s\nSupported providers: %s", providerName, strings.Join(config.GetDisplayProviders(), ", "))
	}
	if setup.ProviderSetupRequired(providerName) {
		return api.NewUnavailableProvider(providerName, setup.ProviderSetupRequiredMessage(providerName)), nil
	}
	provider, err := createProvider(providerName)
	if err != nil {
		if isProviderSetupError(providerName, err) {
			return api.NewUnavailableProvider(providerName, setup.ProviderSetupRequiredMessage(providerName)), nil
		}
		return nil, providerCreationError(providerName, err)
	}
	return provider, nil
}

func providerCreationError(providerName string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "unknown provider") {
		return fmt.Errorf("%w\nSupported providers: %s", err, strings.Join(config.GetDisplayProviders(), ", "))
	}
	if isProviderSetupError(providerName, err) {
		return errors.New(setup.ProviderSetupRequiredMessage(providerName))
	}
	return err
}

func isProviderSetupError(providerName string, err error) bool {
	if err == nil {
		return false
	}
	providerName = config.NormalizeProviderName(providerName)
	if llmcatalog.IsKnownProvider(providerName) && setup.ProviderSetupRequired(providerName) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not set") ||
		strings.Contains(message, "login required") ||
		strings.Contains(message, "not logged in")
}
