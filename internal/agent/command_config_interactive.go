package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type configCommandMenu interface {
	Run() (*config.ConfigCategory, error)
	ShowFieldList(*config.ConfigCategory) (*config.ConfigField, error)
	EditField(*config.ConfigField) (interface{}, bool, error)
}

var (
	buildConfigRegistryForCommand = config.BuildConfigRegistry
	newConfigMenuForCommand       = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
		return ui.NewConfigMenuWithRuntime(cfg, categories, runtime)
	}
)

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
