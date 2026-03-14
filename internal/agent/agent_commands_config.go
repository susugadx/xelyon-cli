package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handleModelCommand はモデルの表示・切り替えを処理
func handleModelCommand(agent *Agent, args []string) bool {
	out := agent.output()

	// 引数なし → 現在のモデルとプロバイダーを表示
	if len(args) == 0 {
		_, _ = fmt.Fprintf(out, "🤖 Current model: %s\n", agent.CurrentModel)
		_, _ = fmt.Fprintf(out, "🌐 Provider: %s\n", agent.ProviderName)
		yellow.Fprintln(out, "\nUsage: /model <model-name>")
		yellow.Fprintln(out, "Enter any model name supported by your provider.")

		// ModelLister対応プロバイダーの場合、インストール済みモデルを表示
		if modelLister, ok := agent.CurrentProvider.(api.ModelLister); ok {
			models, err := modelLister.ListModels()
			if err != nil {
				yellow.Fprintf(out, "\nWarning: Could not list models: %v\n", err)
			} else if len(models) > 0 {
				yellow.Fprintln(out, "\nInstalled models:")
				for _, model := range models {
					_, _ = fmt.Fprintf(out, "  - %s\n", model)
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
	agent.syncSessionModel()
	if agent.Stats != nil {
		agent.Stats.Model = newModel
	}
	agent.rebuildSystemPromptForCurrentProvider()

	green.Fprintf(out, "✅ Model switched: %s → %s\n", oldModel, newModel)
	if agent.CurrentProvider != nil {
		printContextSize(agent)
	}

	// 設定ファイルにも保存
	cfg, err := config.LoadConfig()
	if err != nil {
		yellow.Fprintf(out, "Warning: Failed to load config: %v\n", err)
		return true
	}

	cfg.DefaultModel = newModel

	// プロバイダー別の設定がある場合はそちらも更新（優先されるため）
	if pm, ok := cfg.ProviderModels[agent.ProviderName]; ok {
		pm.DefaultModel = newModel
		cfg.ProviderModels[agent.ProviderName] = pm
	}

	if err := config.SaveConfig(cfg); err != nil {
		yellow.Fprintf(out, "Warning: Failed to save config: %v\n", err)
		yellow.Fprintln(out, "Model switched for this session only")
		return true
	}

	agent.setRuntimeConfig(cfg)
	green.Fprintln(out, "💾 Default model saved to config")
	return true
}

// handleConfigCommand は設定の表示・変更を処理
func handleConfigCommand(agent *Agent, args []string) bool {
	out := agent.output()

	cfg, err := config.LoadConfig()
	if err != nil {
		red.Fprintf(out, "Failed to load config: %v\n", err)
		return true
	}

	// /config show → 全設定をデフォルトとの差分付きで表示
	if len(args) > 0 && args[0] == "show" {
		_, _ = fmt.Fprint(out, config.ShowConfig(cfg))
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
			red.Fprintf(out, "Failed to save config: %v\n", err)
			return true
		}

		if agent != nil {
			agent.setRuntimeConfig(cfg)
			agent.SyncWithRuntimeConfig()
		}

		green.Fprintf(out, "✅ Default model updated to: %s\n", newModel)
		return true
	}

	// 引数なし → 対話式メニュー
	runInteractiveConfig(agent, cfg)
	return true
}

// runInteractiveConfig は対話式設定メニューを実行
func runInteractiveConfig(agent *Agent, cfg *config.Config) {
	out := agent.output()
	categories := config.BuildConfigRegistry(cfg)
	menu := ui.NewConfigMenuWithRuntime(cfg, categories, agent.ui())

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
				red.Fprintf(out, "Error: %v\n", err)
				continue
			}

			if !changed {
				continue
			}

			// StructMap型は直接Configを編集するので、保存のみ
			if selectedField.FieldType == config.FieldTypeStructMap {
				if err := config.SaveConfig(cfg); err != nil {
					red.Fprintf(out, "Error saving: %v\n", err)
				} else {
					green.Fprintf(out, "✓ Saved: %s\n", selectedField.Path)
					if agent != nil {
						agent.setRuntimeConfig(cfg)
						agent.SyncWithRuntimeConfig()
					}
				}
				// カテゴリを再構築
				categories = config.BuildConfigRegistry(cfg)
				menu = ui.NewConfigMenuWithRuntime(cfg, categories, agent.ui())
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
				red.Fprintf(out, "Error setting value: %v\n", err)
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
				red.Fprintf(out, "Error saving: %v\n", err)
				continue
			}

			green.Fprintf(out, "✓ Saved: %s = %v\n", selectedField.Path, newValue)

			if agent != nil {
				agent.setRuntimeConfig(cfg)
				agent.SyncWithRuntimeConfig()
			}

			// カテゴリを再構築して現在値を更新
			categories = config.BuildConfigRegistry(cfg)
			menu = ui.NewConfigMenuWithRuntime(cfg, categories, agent.ui())
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
	providerList := strings.Join(api.ListProviders(), ", ")
	out := agent.output()

	if len(args) == 0 {
		yellow.Fprintln(out, "Usage: /use <provider> [model]")
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		yellow.Fprintln(out, "Example: /use gemini gemini-2.0-flash-exp")
		return true
	}

	providerName := args[0]

	// レジストリで登録済みかチェック
	if !api.IsRegisteredProvider(providerName) {
		red.Fprintf(out, "Unknown provider: %s\n", providerName)
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		return true
	}

	// 既に同じプロバイダーの場合でも、モデルが指定されていれば切り替え
	if agent.ProviderName == providerName && len(args) < 2 {
		yellow.Fprintf(out, "Already using %s (model: %s)\n", providerName, agent.CurrentModel)
		yellow.Fprintln(out, "Hint: Use '/use <provider> <model>' to change model")
		return true
	}

	// プロバイダー切り替え実行
	if err := agent.SwitchProvider(providerName); err != nil {
		red.Fprintf(out, "❌ %v\n", err)

		// API キー設定方法を表示
		switch providerName {
		case "deepseek":
			yellow.Fprintln(out, "\n設定方法:")
			yellow.Fprintln(out, "  export DEEPSEEK_API_KEY=your-api-key")
		case "openai":
			yellow.Fprintln(out, "\n設定方法:")
			yellow.Fprintln(out, "  export OPENAI_API_KEY=your-api-key")
		case "claude":
			yellow.Fprintln(out, "\n設定方法:")
			yellow.Fprintln(out, "  export ANTHROPIC_API_KEY=your-api-key")
		case "gemini":
			yellow.Fprintln(out, "\n設定方法:")
			yellow.Fprintln(out, "  export GEMINI_API_KEY=your-api-key")
		case "groq":
			yellow.Fprintln(out, "\n設定方法:")
			yellow.Fprintln(out, "  export GROQ_API_KEY=your-api-key")
		case "openrouter":
			yellow.Fprintln(out, "\n設定方法:")
			yellow.Fprintln(out, "  export OPENROUTER_API_KEY=your-api-key")
		case "bedrock":
			yellow.Fprintln(out, "\n設定方法:")
			yellow.Fprintln(out, "  AWS認証チェーン（IAMロール、環境変数、~/.aws/credentials等）を設定")
		}
		return true
	}

	// モデル指定がある場合は追加でモデルを切り替え
	if len(args) >= 2 {
		newModel := args[1]
		oldModel := agent.CurrentModel
		agent.CurrentModel = newModel
		agent.syncSessionModel()
		if agent.Stats != nil {
			agent.Stats.Model = newModel
		}
		agent.rebuildSystemPromptForCurrentProvider()
		green.Fprintf(out, "✅ Model: %s → %s\n", oldModel, newModel)
	}

	if agent.CurrentProvider != nil {
		printContextSize(agent)
	}

	return true
}

// handleProvidersCommand は利用可能なプロバイダー一覧を表示
func handleProvidersCommand(agent *Agent) bool {
	providers := api.ListProviders()
	out := agent.output()

	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Fprintln(out, "📡 利用可能なプロバイダー / Available Providers")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)

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
		} else if provider == "bedrock" {
			status = "(AWS認証)"
		} else if hasAPIKey {
			status = "(API key設定済み)"
		} else {
			status = "(API key未設定)"
		}

		// 色付け
		if isCurrent {
			green.Fprintf(out, "%s%-12s %s\n", icon, provider, status)
		} else if hasAPIKey {
			_, _ = fmt.Fprintf(out, "%s%-12s %s\n", icon, provider, status)
		} else {
			// API key未設定は薄く表示
			_, _ = fmt.Fprintf(out, "%s%-12s %s\n", icon, provider, status)
		}
	}

	_, _ = fmt.Fprintln(out)
	cyan.Fprintln(out, "使い方: /use <provider>")
	cyan.Fprintln(out, "例: /use claude")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}
