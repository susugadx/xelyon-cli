package cmd

import (
	"fmt"
	"os"
	"strings"

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
	if flagValue != "" {
		debugLog("provider from flag: %s", flagValue)
		return flagValue
	}

	if envValue := os.Getenv("XELYON_PROVIDER"); envValue != "" {
		debugLog("provider from env: %s", envValue)
		return envValue
	}

	if configValue != "" {
		debugLog("provider from config: %s", configValue)
		return configValue
	}

	debugLog("using default provider: deepseek")
	return "deepseek"
}

// createProvider はプロバイダー名からProviderを生成（テスト可能）
func createProvider(providerName string) (api.Provider, error) {
	name := strings.ToLower(providerName)

	// api.NewProvider は内部で環境変数をチェックし、プロバイダーを生成する
	return api.NewProvider(name)
}

// getProvider は環境変数/設定ファイルからProviderを取得
// 優先順位: CLI flag > 環境変数 > 設定ファイル > デフォルト
func getProvider(cfg *config.Config) api.Provider {
	debugLog("getProvider: providerFlag=%q", providerFlag)

	var configProvider string
	if cfg != nil {
		configProvider = cfg.DefaultProvider
		debugLog("getProvider: config.DefaultProvider=%q", configProvider)
	}

	// 優先順位に従ってプロバイダー名を解決
	providerName := resolveProviderName(providerFlag, configProvider)
	debugLog("getProvider: final provider=%q", providerName)

	return getProviderByName(providerName)
}

// getProviderByName はプロバイダー名から Provider インスタンスを生成
// エラー時は os.Exit(1) を呼び出す（CLIエントリーポイント用）
func getProviderByName(providerName string) api.Provider {
	provider, err := createProvider(providerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if strings.Contains(err.Error(), "unknown provider") {
			fmt.Fprintf(os.Stderr, "Supported providers: %s\n", strings.Join(config.GetDisplayProviders(), ", "))
		}
		os.Exit(1)
	}
	return provider
}
