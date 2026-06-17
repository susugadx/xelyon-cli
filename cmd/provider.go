package cmd

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/cliruntime"
)

// resolveProviderName はプロバイダー名を優先順位に従って解決
// 優先順位: CLI flag > 環境変数 > 設定ファイル > デフォルト
func resolveProviderName(flagValue, configValue string) string {
	return cliruntime.ResolveProviderName(flagValue, configValue)
}

// createProvider はプロバイダー名からProviderを生成（テスト可能）
func createProvider(providerName string) (api.Provider, error) {
	return cliruntime.CreateProvider(providerName)
}

func resolveRequiredProvider(providerName string) (api.Provider, error) {
	return cliruntime.ResolveRequiredProvider(providerName)
}

func resolveInteractiveProvider(providerName string) (api.Provider, error) {
	return cliruntime.ResolveInteractiveProvider(providerName)
}

func isProviderSetupError(providerName string, err error) bool {
	return cliruntime.IsProviderSetupError(providerName, err)
}
