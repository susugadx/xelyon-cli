package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type configCommandMenu interface {
	Run() (*config.ConfigCategory, error)
	ShowFieldList(*config.ConfigCategory) (*config.ConfigField, error)
	EditField(*config.ConfigField) (interface{}, bool, error)
}

var (
	loadConfigForCommand          = config.LoadConfig
	saveConfigForCommand          = config.SaveConfig
	showConfigForCommand          = config.ShowConfig
	setFieldValueForCommand       = config.SetFieldValue
	buildConfigRegistryForCommand = config.BuildConfigRegistry
	newConfigMenuForCommand       = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
		return ui.NewConfigMenuWithRuntime(cfg, categories, runtime)
	}
)

// handleModelCommand はモデルの表示・切り替えを処理
func handleModelCommand(agent *Agent, args []string) bool {
	out := agent.output()

	// 引数なし → 現在のモデルとプロバイダーを表示
	if len(args) == 0 {
		state := agent.CurrentProviderModelState()
		_, _ = fmt.Fprintf(out, "🤖 Current model: %s\n", state.CurrentModel)
		_, _ = fmt.Fprintf(out, "🌐 Provider: %s\n", state.CurrentProvider)
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
	_ = switchModelForCurrentProviderWithOutput(agent, args[0])
	return true
}

func switchModelForCurrentProviderWithOutput(agent *Agent, newModel string) error {
	out := agent.output()
	outcome := agent.SwitchModelForCurrentProvider(newModel)
	if outcome.ValidationErr != nil {
		red.Fprintf(out, "Error: %v\n", outcome.ValidationErr)
		return outcome.ValidationErr
	}

	green.Fprintf(out, "✅ Model switched: %s → %s\n", outcome.OldModel, outcome.NewModel)
	if agent.CurrentProvider != nil {
		printContextSize(agent)
	}

	if outcome.LoadConfigErr != nil {
		yellow.Fprintf(out, "Warning: Failed to load config: %v\n", outcome.LoadConfigErr)
		return outcome.LoadConfigErr
	}

	if outcome.SaveConfigErr != nil {
		yellow.Fprintf(out, "Warning: Failed to save config: %v\n", outcome.SaveConfigErr)
		yellow.Fprintln(out, "Model switched for this session only")
		return outcome.SaveConfigErr
	}

	green.Fprintln(out, "💾 Default model saved to config")
	return nil
}

// handleConfigCommand は設定の表示・変更を処理
func handleConfigCommand(agent *Agent, args []string) bool {
	out := agent.output()

	cfg, err := loadConfigForCommand()
	if err != nil {
		red.Fprintf(out, "Failed to load config: %v\n", err)
		return true
	}

	// /config show → 全設定をデフォルトとの差分付きで表示
	if len(args) > 0 && args[0] == "show" {
		_, _ = fmt.Fprint(out, showConfigForCommand(cfg))
		return true
	}

	// /config model <model-name> → モデル変更
	if len(args) >= 2 && args[0] == "model" {
		newModel := args[1]
		if err := validateConfigModelChange(agent, cfg, newModel); err != nil {
			red.Fprintf(out, "Error: %v\n", err)
			return true
		}

		// 設定更新
		cfg.DefaultModel = newModel

		// プロバイダー別の設定がある場合はそちらも更新
		if agent != nil {
			agent.SyncDefaultModelToProvider(cfg)
		}

		if err := saveConfigForCommand(cfg); err != nil {
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

// isNonInteractiveConfigSubcommand は stdin を読まずに処理できる /config サブコマンドかを返す。
func isNonInteractiveConfigSubcommand(args []string) bool {
	return commandruntime.IsNonInteractiveConfigSubcommand(args)
}

// runInteractiveConfig は対話式設定メニューを実行
func runInteractiveConfig(agent *Agent, cfg *config.Config) {
	out := agent.output()
	categories := buildConfigRegistryForCommand(cfg)
	menu := newConfigMenuForCommand(cfg, categories, agent.ui())

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

			beforeStructMapEdit := (*config.Config)(nil)
			if selectedField.FieldType == config.FieldTypeStructMap {
				beforeStructMapEdit = config.CloneConfig(cfg)
			}

			// フィールド編集
			newValue, changed, err := menu.EditField(selectedField)
			if err != nil {
				restoreConfigSnapshot(cfg, beforeStructMapEdit)
				red.Fprintf(out, "Error: %v\n", err)
				continue
			}

			if !changed {
				restoreConfigSnapshot(cfg, beforeStructMapEdit)
				continue
			}

			if selectedField.Path == "default_model" {
				if _, ok := newValue.(string); !ok {
					red.Fprintf(out, "Error setting value: default_model must be a string\n")
					continue
				}
			}
			if err := validateInteractiveScalarConfigChange(agent, cfg, selectedField.Path, newValue); err != nil {
				red.Fprintf(out, "Error: %v\n", err)
				continue
			}

			// StructMap型は直接Configを編集するので、保存のみ
			if selectedField.FieldType == config.FieldTypeStructMap {
				if err := validateInteractiveStructMapConfigChange(cfg, selectedField.Path); err != nil {
					restoreConfigSnapshot(cfg, beforeStructMapEdit)
					red.Fprintf(out, "Error: %v\n", err)
					_, menu, selectedCategory = refreshInteractiveConfigMenu(agent, cfg, selectedCategory)
					continue
				}
				if err := saveConfigForCommand(cfg); err != nil {
					red.Fprintf(out, "Error saving: %v\n", err)
				} else {
					green.Fprintf(out, "✓ Saved: %s\n", selectedField.Path)
					if agent != nil {
						agent.setRuntimeConfig(cfg)
						agent.SyncWithRuntimeConfig()
					}
				}
				// カテゴリを再構築
				_, menu, selectedCategory = refreshInteractiveConfigMenu(agent, cfg, selectedCategory)
				continue
			}

			// 値を設定
			if err := setFieldValueForCommand(cfg, selectedField.Path, newValue); err != nil {
				red.Fprintf(out, "Error setting value: %v\n", err)
				continue
			}

			// default_model 変更時はプロバイダー別設定も同期
			if selectedField.Path == "default_model" && agent != nil {
				if strValue, ok := newValue.(string); ok {
					cfg.DefaultModel = strValue
					agent.SyncDefaultModelToProvider(cfg)
				}
			}

			// 保存
			if err := saveConfigForCommand(cfg); err != nil {
				red.Fprintf(out, "Error saving: %v\n", err)
				continue
			}

			green.Fprintf(out, "✓ Saved: %s = %v\n", selectedField.Path, newValue)

			if agent != nil {
				agent.setRuntimeConfig(cfg)
				agent.SyncWithRuntimeConfig()
			}

			// カテゴリを再構築して現在値を更新
			categories = buildConfigRegistryForCommand(cfg)
			menu = newConfigMenuForCommand(cfg, categories, agent.ui())
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

func validateInteractiveStructMapConfigChange(cfg *config.Config, path string) error {
	if path != "provider_models" {
		return nil
	}
	return validateGeminiFunctionCallingConfigForSave(cfg)
}

func validateInteractiveScalarConfigChange(agent *Agent, cfg *config.Config, path string, value interface{}) error {
	switch path {
	case "default_model":
		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("default_model must be a string")
		}
		return validateConfigModelChange(agent, cfg, strValue)
	case "default_provider":
		candidate := config.CloneConfig(cfg)
		if err := config.SetFieldValue(candidate, path, value); err != nil {
			return err
		}
		return validateGeminiFunctionCallingConfigForSave(candidate)
	default:
		return nil
	}
}

func restoreConfigSnapshot(cfg, snapshot *config.Config) {
	if cfg == nil || snapshot == nil {
		return
	}
	*cfg = *config.CloneConfig(snapshot)
}

func refreshInteractiveConfigMenu(agent *Agent, cfg *config.Config, selectedCategory *config.ConfigCategory) ([]config.ConfigCategory, configCommandMenu, *config.ConfigCategory) {
	categories := buildConfigRegistryForCommand(cfg)
	menu := newConfigMenuForCommand(cfg, categories, agent.ui())
	if selectedCategory == nil {
		return categories, menu, selectedCategory
	}
	for i := range categories {
		if categories[i].Name == selectedCategory.Name {
			selectedCategory = &categories[i]
			break
		}
	}
	return categories, menu, selectedCategory
}

// handleProviderCommand はプロバイダーを切り替える。
func handleProviderCommand(agent *Agent, args []string) bool {
	return handleProviderSwitchCommand(agent, args, "/provider")
}

// handleUseCommand は後方互換のプロバイダー切り替えコマンドを処理する。
func handleUseCommand(agent *Agent, args []string) bool {
	return handleProviderSwitchCommand(agent, args, "/use")
}

func handleProviderSwitchCommand(agent *Agent, args []string, commandName string) bool {
	providerList := strings.Join(api.ListProviders(), ", ")
	out := agent.output()

	if len(args) == 0 {
		yellow.Fprintf(out, "Usage: %s <provider> [model]\n", commandName)
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		yellow.Fprintf(out, "Example: %s gemini gemini-3.5-flash\n", commandName)
		return true
	}

	providerName := args[0]
	requestedModel := ""
	if len(args) >= 2 {
		requestedModel = args[1]
	}

	// レジストリで登録済みかチェック
	if !api.IsRegisteredProvider(providerName) {
		red.Fprintf(out, "Unknown provider: %s\n", providerName)
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		return true
	}

	// 既に同じプロバイダーの場合でも、モデルが指定されていれば切り替え
	state := agent.CurrentProviderModelState()
	requestedProviderConfigKey := config.ActiveProviderConfigKey(providerName)
	if len(args) < 2 && requestedProviderConfigKey != "" && requestedProviderConfigKey == state.ProviderConfigKey {
		yellow.Fprintf(out, "Already using %s (model: %s)\n", providerName, state.CurrentModel)
		yellow.Fprintf(out, "Hint: Use '%s <provider> <model>' to change model\n", commandName)
		return true
	}

	_ = switchProviderModelWithOutput(agent, providerName, requestedModel)
	return true
}

func switchProviderModelWithOutput(agent *Agent, providerName, requestedModel string) error {
	providerList := strings.Join(api.ListProviders(), ", ")
	out := agent.output()

	if !api.IsRegisteredProvider(providerName) {
		red.Fprintf(out, "Unknown provider: %s\n", providerName)
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	outcome, err := agent.SwitchProviderModel(providerName, requestedModel)
	if err != nil {
		red.Fprintf(out, "❌ %v\n", err)

		// API キー設定方法を表示
		if instructions := config.ProviderSetupInstructions(providerName); len(instructions) > 0 {
			yellow.Fprintln(out, "\n設定方法:")
			for _, line := range instructions {
				yellow.Fprintf(out, "  %s\n", line)
			}
		}
		return err
	}

	printProviderSwitchOutcome(agent, outcome)
	if agent.CurrentProvider != nil {
		printContextSize(agent)
	}

	return nil
}

// handleProvidersCommand は provider credential status を表示する。
func handleProvidersCommand(agent *Agent) bool {
	out := agent.output()
	providers := agent.ProviderCandidates()

	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Fprintln(out, "📡 Provider credential status")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)

	for _, provider := range providers {
		icon := "  "
		if provider.Current {
			icon = "✓ "
		}
		label := provider.Label
		if label == "" {
			label = provider.Key
		}
		if provider.Key != "" && provider.Key != label {
			label += " (" + provider.Key + ")"
		}
		status := providerCredentialStatusDisplay(provider.CredentialStatus)
		if provider.Current {
			green.Fprintf(out, "%s%-24s %s\n", icon, label, status)
		} else {
			_, _ = fmt.Fprintf(out, "%s%-24s %s\n", icon, label, status)
		}
	}

	_, _ = fmt.Fprintln(out)
	cyan.Fprintln(out, "Usage: /provider [provider] [model]")
	cyan.Fprintln(out, "TUI: /provider opens the provider/model picker")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}

func providerCredentialStatusDisplay(status ProviderCredentialStatus) string {
	switch status {
	case ProviderCredentialConfigured:
		return "(credential configured)"
	case ProviderCredentialLoggedIn:
		return "(logged in)"
	case ProviderCredentialLoginRequired:
		return "(login required)"
	case ProviderCredentialLocal:
		return "(local)"
	case ProviderCredentialAWSAuth:
		return "(AWS auth)"
	default:
		return "(credential missing)"
	}
}
