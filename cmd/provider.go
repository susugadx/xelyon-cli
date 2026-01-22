package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// providerConfig はプロバイダーの設定を保持
type providerConfig struct {
	envKey       string                             // 環境変数名（空の場合はAPIキー不要）
	defaultValue string                             // デフォルト値（OllamaのベースURLなど）
	constructor  func(keyOrURL string) api.Provider // Provider生成関数
}

// providerConfigs はプロバイダー設定のマップ（DRY化）
var providerConfigs = map[string]providerConfig{
	"deepseek": {
		envKey:      "DEEPSEEK_API_KEY",
		constructor: func(key string) api.Provider { return api.NewDeepSeekProvider(key) },
	},
	"openai": {
		envKey:      "OPENAI_API_KEY",
		constructor: func(key string) api.Provider { return api.NewOpenAIProvider(key) },
	},
	"gemini": {
		envKey:      "GEMINI_API_KEY",
		constructor: func(key string) api.Provider { return api.NewGeminiProvider(key) },
	},
	"claude": {
		envKey:      "ANTHROPIC_API_KEY",
		constructor: func(key string) api.Provider { return api.NewClaudeProvider(key) },
	},
	"anthropic": {
		envKey:      "ANTHROPIC_API_KEY",
		constructor: func(key string) api.Provider { return api.NewClaudeProvider(key) },
	},
	"ollama": {
		envKey:       "",
		defaultValue: "http://localhost:11434",
		constructor:  func(url string) api.Provider { return api.NewOllamaProvider(url) },
	},
	"groq": {
		envKey:      "GROQ_API_KEY",
		constructor: func(key string) api.Provider { return api.NewGroqProvider(key) },
	},
}

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

	cfg, ok := providerConfigs[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (supported: deepseek, openai, gemini, claude, ollama, groq)", providerName)
	}

	// APIキーが不要なプロバイダー（Ollama）
	if cfg.envKey == "" {
		value := os.Getenv("OLLAMA_BASE_URL")
		if value == "" {
			value = cfg.defaultValue
		}
		return cfg.constructor(value), nil
	}

	// APIキーが必要なプロバイダー
	apiKey := os.Getenv(cfg.envKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set", cfg.envKey)
	}

	return cfg.constructor(apiKey), nil
}

// getProvider は環境変数/設定ファイルからProviderを取得
// 優先順位: CLI flag > 環境変数 > 設定ファイル > デフォルト
func getProvider() api.Provider {
	debugLog("getProvider: providerFlag=%q", providerFlag)

	// 設定ファイルからデフォルトプロバイダーを取得
	var configProvider string
	cfg, err := config.LoadConfig()
	if err != nil {
		debugLog("getProvider: config.LoadConfig() error: %v", err)
	} else if cfg != nil {
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
			fmt.Fprintln(os.Stderr, "Supported providers: deepseek, openai, gemini, claude, ollama, groq")
		}
		os.Exit(1)
	}
	return provider
}
