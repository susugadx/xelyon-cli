package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SaveConfig は設定ファイルを保存する。
func saveConfig(cfg *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// ディレクトリ作成
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// YAML形式で保存
	data, err := marshalConfigYAML(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// ヘッダーコメント追加
	header := fmt.Sprintf("# XELYON CLI 設定\n# Providers: %s\n# 各プロバイダーのモデル設定は provider_models で管理されます\n\n", strings.Join(GetDisplayProviders(), ", "))
	fullData := []byte(header + string(data))

	if err := os.WriteFile(configPath, fullData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func marshalConfigYAML(cfg *Config) ([]byte, error) {
	saveCfg := CloneConfig(cfg)
	if saveCfg != nil {
		saveCfg.ProviderModels = saveCfg.ProviderModelsForSave()
	}

	data, err := yaml.Marshal(saveCfg)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return data, nil
	}

	if saveCfg != nil && saveCfg.LSP.Servers == nil {
		_ = setNestedYAMLValueToNull(&doc, "lsp", "servers")
	}
	if saveCfg == nil || saveCfg.ProviderModels == nil {
		removeTopLevelYAMLKey(&doc, "provider_models")
	}

	patched, err := yaml.Marshal(&doc)
	if err != nil {
		return data, nil
	}
	return patched, nil
}

func setNestedYAMLValueToNull(doc *yaml.Node, parentKey, childKey string) bool {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return false
	}

	parent := findYAMLMappingValue(doc.Content[0], parentKey)
	if parent == nil || parent.Kind != yaml.MappingNode {
		return false
	}

	child := findYAMLMappingValue(parent, childKey)
	if child == nil {
		return false
	}

	*child = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	return true
}

func removeTopLevelYAMLKey(doc *yaml.Node, key string) bool {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return false
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content = append(root.Content[:i], root.Content[i+2:]...)
			return true
		}
	}

	return false
}

func findYAMLMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
