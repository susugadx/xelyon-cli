package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig は設定ファイルを読み込む
func loadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// 設定ファイルが存在しない場合はデフォルトを作成
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
		cfg.refreshEffectiveProviderModels()
		if err := SaveConfig(cfg); err != nil {
			// 保存失敗してもデフォルト設定を返す
			return cfg, nil
		}
		return cfg, nil
	}

	// 設定ファイルを読み込む
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// デフォルト値で初期化してからYAMLをマージ
	// これにより、YAMLで明示的に設定されたフィールドのみが上書きされる
	cfg := DefaultConfig()
	lspSectionExists := yamlHasKey(data, "lsp")
	lspServersExists := yamlHasNestedKey(data, "lsp", "servers")
	if lspServersExists {
		// lsp.servers は nil と empty map を区別したいので、
		// YAML に存在する場合だけ defaults 側の既定 map を事前に外す。
		cfg.LSP.Servers = nil
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 旧キーからの migration（後方互換）
	migrateOldKeys(data, cfg)
	cfg.providerModelsStore = providerModelStoreFromYAML(data)

	// 追加のデフォルト値を適用（ネストされた構造体用）
	applyDefaults(cfg, defaultApplyOptions{
		lspSectionExists: lspSectionExists,
		lspServersExists: lspServersExists,
	})

	return cfg, nil
}

func yamlHasNestedKey(data []byte, parentKey, childKey string) bool {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}

	parent, ok := raw[parentKey].(map[string]interface{})
	if !ok {
		return false
	}

	_, exists := parent[childKey]
	return exists
}

func yamlHasKey(data []byte, key string) bool {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	_, exists := raw[key]
	return exists
}
