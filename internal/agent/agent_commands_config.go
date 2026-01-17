package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// handleModelCommand はモデルの表示・切り替えを処理
func handleModelCommand(agent *Agent, args []string) bool {
	// 引数なし → 現在のモデルとプロバイダーを表示
	if len(args) == 0 {
		fmt.Printf("🤖 Current model: %s\n", agent.CurrentModel)
		fmt.Printf("🌐 Provider: %s\n", agent.ProviderName)
		yellow.Println("\nUsage: /model <model-name>")
		yellow.Println("Enter any model name supported by your provider.")

		// Ollamaの場合だけインストール済みモデルを表示
		if agent.ProviderName == "ollama" {
			if ollamaProvider, ok := agent.CurrentProvider.(*api.OllamaProvider); ok {
				models, err := ollamaProvider.ListModels()
				if err != nil {
					yellow.Printf("\nWarning: Could not list Ollama models: %v\n", err)
				} else if len(models) > 0 {
					yellow.Println("\nInstalled Ollama models:")
					for _, model := range models {
						fmt.Printf("  - %s\n", model)
					}
				}
			}
		}
		return true
	}

	// /model <model-name> → モデル切り替え
	newModel := args[0]

	// モデルを切り替え
	oldModel := agent.CurrentModel
	agent.CurrentModel = newModel

	green.Printf("✅ Model switched: %s → %s\n", oldModel, newModel)

	// 設定ファイルにも保存
	cfg, err := config.LoadConfig()
	if err != nil {
		yellow.Printf("Warning: Failed to load config: %v\n", err)
		return true
	}

	cfg.DefaultModel = newModel
	if err := config.SaveConfig(cfg); err != nil {
		yellow.Printf("Warning: Failed to save config: %v\n", err)
		yellow.Println("Model switched for this session only")
		return true
	}

	green.Println("💾 Default model saved to config")
	return true
}

// handleConfigCommand は設定の表示・変更を処理
func handleConfigCommand(args []string) bool {
	cfg, err := config.LoadConfig()
	if err != nil {
		red.Printf("Failed to load config: %v\n", err)
		return true
	}

	// 引数なし → 簡易表示
	if len(args) == 0 {
		cyan.Println("⚙️  Current Configuration:")
		fmt.Printf("  default_model: %s\n", cfg.DefaultModel)
		fmt.Printf("  default_provider: %s\n", cfg.DefaultProvider)
		yellow.Println("\nUsage:")
		yellow.Println("  /config show               - Show all settings with diff from defaults")
		yellow.Println("  /config model <model-name> - Change default model")
		return true
	}

	// /config show → 全設定をデフォルトとの差分付きで表示
	if args[0] == "show" {
		fmt.Print(config.ShowConfig(cfg))
		return true
	}

	// /config model <model-name> → モデル変更
	if len(args) >= 2 && args[0] == "model" {
		newModel := args[1]

		// 設定更新（バリデーションなし、任意のモデル名を受け付ける）
		cfg.DefaultModel = newModel
		if err := config.SaveConfig(cfg); err != nil {
			red.Printf("Failed to save config: %v\n", err)
			return true
		}

		green.Printf("✅ Default model updated to: %s\n", newModel)
		yellow.Println("Restart CLI for changes to take effect")
		return true
	}

	yellow.Println("Usage:")
	yellow.Println("  /config show               - Show all settings with diff from defaults")
	yellow.Println("  /config model <model-name> - Change default model")
	return true
}

// handleUseCommand はプロバイダーを切り替える
func handleUseCommand(agent *Agent, args []string) bool {
	if len(args) == 0 {
		yellow.Println("Usage: /use <provider> [model]")
		yellow.Println("Available providers: deepseek, claude, openai, gemini, groq, ollama")
		yellow.Println("Example: /use gemini gemini-2.0-flash-exp")
		return true
	}

	providerName := args[0]

	// サポートされているプロバイダーかチェック
	validProviders := map[string]bool{
		"deepseek": true,
		"claude":   true,
		"openai":   true,
		"gemini":   true,
		"groq":     true,
		"ollama":   true,
	}

	if !validProviders[providerName] {
		red.Printf("Unknown provider: %s\n", providerName)
		yellow.Println("Available providers: deepseek, claude, openai, gemini, groq, ollama")
		return true
	}

	// 既に同じプロバイダーの場合でも、モデルが指定されていれば切り替え
	if agent.ProviderName == providerName && len(args) < 2 {
		yellow.Printf("Already using %s (model: %s)\n", providerName, agent.CurrentModel)
		yellow.Println("Hint: Use '/use <provider> <model>' to change model")
		return true
	}

	// プロバイダー切り替え実行
	if err := agent.SwitchProvider(providerName); err != nil {
		red.Printf("❌ %v\n", err)

		// API キー設定方法を表示
		switch providerName {
		case "deepseek":
			yellow.Println("\n設定方法:")
			yellow.Println("  export DEEPSEEK_API_KEY=your-api-key")
		case "openai":
			yellow.Println("\n設定方法:")
			yellow.Println("  export OPENAI_API_KEY=your-api-key")
		case "claude":
			yellow.Println("\n設定方法:")
			yellow.Println("  export ANTHROPIC_API_KEY=your-api-key")
		case "gemini":
			yellow.Println("\n設定方法:")
			yellow.Println("  export GEMINI_API_KEY=your-api-key")
		case "groq":
			yellow.Println("\n設定方法:")
			yellow.Println("  export GROQ_API_KEY=your-api-key")
		}
		return true
	}

	// モデル指定がある場合は追加でモデルを切り替え
	if len(args) >= 2 {
		newModel := args[1]
		oldModel := agent.CurrentModel
		agent.CurrentModel = newModel
		green.Printf("✅ Model: %s → %s\n", oldModel, newModel)
	}

	return true
}

// handleProvidersCommand は利用可能なプロバイダー一覧を表示
func handleProvidersCommand(agent *Agent) bool {
	providers := []string{"deepseek", "claude", "openai", "gemini", "groq", "ollama"}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("📡 利用可能なプロバイダー / Available Providers")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for _, provider := range providers {
		// 現在使用中かチェック
		isCurrent := agent.ProviderName == provider
		hasAPIKey := IsAPIKeyAvailable(provider)

		// アイコン
		icon := "  "
		if isCurrent {
			icon = "✓ "
		}

		// ステータス
		status := ""
		if provider == "ollama" {
			status = "(ローカル)"
		} else if hasAPIKey {
			status = "(API key設定済み)"
		} else {
			status = "(API key未設定)"
		}

		// 色付け
		if isCurrent {
			green.Printf("%s%-12s %s\n", icon, provider, status)
		} else if hasAPIKey {
			fmt.Printf("%s%-12s %s\n", icon, provider, status)
		} else {
			// API key未設定は薄く表示
			fmt.Printf("%s%-12s %s\n", icon, provider, status)
		}
	}

	fmt.Println()
	cyan.Println("使い方: /use <provider>")
	cyan.Println("例: /use claude")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}
