package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handleModelCommand はモデルの表示・切り替えを処理
func handleModelCommand(agent *Agent, args []string) bool {
	// 引数なし → 現在のモデルとプロバイダーを表示
	if len(args) == 0 {
		fmt.Printf("🤖 Current model: %s\n", agent.CurrentModel)
		fmt.Printf("🌐 Provider: %s\n", agent.ProviderName)
		yellow.Println("\nUsage: /model <model-name>")
		yellow.Println("Enter any model name supported by your provider.")

		// ModelLister対応プロバイダーの場合、インストール済みモデルを表示
		if modelLister, ok := agent.CurrentProvider.(api.ModelLister); ok {
			models, err := modelLister.ListModels()
			if err != nil {
				yellow.Printf("\nWarning: Could not list models: %v\n", err)
			} else if len(models) > 0 {
				yellow.Println("\nInstalled models:")
				for _, model := range models {
					fmt.Printf("  - %s\n", model)
				}
			}
		}
		return true
	}

	// /model <model-name> → モデル切り替え
	newModel := args[0]

	// 既存プロバイダーのキャッシュをクリア（サポートしている場合）
	if agent.CurrentProvider != nil {
		if cacheClearable, ok := agent.CurrentProvider.(api.CacheClearable); ok {
			cacheClearable.ClearCache()
		}
	}

	// モデルを切り替え
	oldModel := agent.CurrentModel
	agent.CurrentModel = newModel
	if agent.Stats != nil {
		agent.Stats.Model = newModel
	}

	green.Printf("✅ Model switched: %s → %s\n", oldModel, newModel)

	// 設定ファイルにも保存
	cfg, err := config.LoadConfig()
	if err != nil {
		yellow.Printf("Warning: Failed to load config: %v\n", err)
		return true
	}

	cfg.DefaultModel = newModel

	// プロバイダー別の設定がある場合はそちらも更新（優先されるため）
	if pm, ok := cfg.ProviderModels[agent.ProviderName]; ok {
		pm.DefaultModel = newModel
		cfg.ProviderModels[agent.ProviderName] = pm
	}

	if err := config.SaveConfig(cfg); err != nil {
		yellow.Printf("Warning: Failed to save config: %v\n", err)
		yellow.Println("Model switched for this session only")
		return true
	}

	green.Println("💾 Default model saved to config")
	return true
}

// handleConfigCommand は設定の表示・変更を処理
func handleConfigCommand(agent *Agent, args []string) bool {
	cfg, err := config.LoadConfig()
	if err != nil {
		red.Printf("Failed to load config: %v\n", err)
		return true
	}

	// /config show → 全設定をデフォルトとの差分付きで表示
	if len(args) > 0 && args[0] == "show" {
		fmt.Print(config.ShowConfig(cfg))
		return true
	}

	// /config model <model-name> → モデル変更
	if len(args) >= 2 && args[0] == "model" {
		newModel := args[1]

		// 設定更新（バリデーションなし、任意のモデル名を受け付ける）
		cfg.DefaultModel = newModel

		// プロバイダー別の設定がある場合はそちらも更新
		if agent != nil {
			if pm, ok := cfg.ProviderModels[agent.ProviderName]; ok {
				pm.DefaultModel = newModel
				cfg.ProviderModels[agent.ProviderName] = pm
			}
		}

		if err := config.SaveConfig(cfg); err != nil {
			red.Printf("Failed to save config: %v\n", err)
			return true
		}

		// グローバル設定を更新（このプロセス内で即反映させる）
		config.SetGlobalConfig(cfg)
		// Agent にも同期（フッター/次回API呼び出し）
		if agent != nil {
			agent.SyncWithGlobalConfig()
		}

		green.Printf("✅ Default model updated to: %s\n", newModel)
		return true
	}

	// 引数なし → 対話式メニュー
	runInteractiveConfig(agent, cfg)
	return true
}

// runInteractiveConfig は対話式設定メニューを実行
func runInteractiveConfig(agent *Agent, cfg *config.Config) {
	categories := config.BuildConfigRegistry(cfg)
	menu := ui.NewConfigMenu(cfg, categories)

	for {
		// カテゴリ選択
		selectedCategory, err := menu.Run()
		if err != nil || selectedCategory == nil {
			return // 'q' でキャンセル
		}

		// フィールド選択ループ
		for {
			selectedField, err := menu.ShowFieldList(selectedCategory)
			if err != nil {
				break // 'b' で戻る
			}

			// フィールド編集
			newValue, changed, err := menu.EditField(selectedField)
			if err != nil {
				red.Printf("Error: %v\n", err)
				continue
			}

			if !changed {
				continue
			}

			// StructMap型は直接Configを編集するので、保存のみ
			if selectedField.FieldType == config.FieldTypeStructMap {
				if err := config.SaveConfig(cfg); err != nil {
					red.Printf("Error saving: %v\n", err)
				} else {
					green.Printf("✓ Saved: %s\n", selectedField.Path)
					// グローバル設定/Agent を同期（即反映）
					config.SetGlobalConfig(cfg)
					if agent != nil {
						agent.SyncWithGlobalConfig()
					}
				}
				// カテゴリを再構築
				categories = config.BuildConfigRegistry(cfg)
				menu = ui.NewConfigMenu(cfg, categories)
				// 現在のカテゴリを更新
				for i := range categories {
					if categories[i].Name == selectedCategory.Name {
						selectedCategory = &categories[i]
						break
					}
				}
				continue
			}

			// 値を設定
			if err := config.SetFieldValue(cfg, selectedField.Path, newValue); err != nil {
				red.Printf("Error setting value: %v\n", err)
				continue
			}

			// default_model 変更時はプロバイダー別設定も同期
			if selectedField.Path == "default_model" && agent != nil {
				if strValue, ok := newValue.(string); ok {
					if pm, ok := cfg.ProviderModels[agent.ProviderName]; ok {
						pm.DefaultModel = strValue
						cfg.ProviderModels[agent.ProviderName] = pm
					}
				}
			}

			// 保存
			if err := config.SaveConfig(cfg); err != nil {
				red.Printf("Error saving: %v\n", err)
				continue
			}

			green.Printf("✓ Saved: %s = %v\n", selectedField.Path, newValue)

			// グローバル設定/Agent を同期（即反映）
			config.SetGlobalConfig(cfg)
			if agent != nil {
				agent.SyncWithGlobalConfig()
			}

			// カテゴリを再構築して現在値を更新
			categories = config.BuildConfigRegistry(cfg)
			menu = ui.NewConfigMenu(cfg, categories)
			// 現在のカテゴリを更新
			for i := range categories {
				if categories[i].Name == selectedCategory.Name {
					selectedCategory = &categories[i]
					break
				}
			}
		}
	}
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
