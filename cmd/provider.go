package cmd

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
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
