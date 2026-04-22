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

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return bootstrapMissingConfig()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return loadConfigFromData(data)
}

func bootstrapMissingConfig() (*Config, error) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.refreshEffectiveProviderModels()
	if err := SaveConfig(cfg); err != nil {
		return cfg, nil
	}
	return cfg, nil
}

func loadConfigFromData(data []byte) (*Config, error) {
	raw := parseYAMLRootMap(data)
	sections := detectLoaderSectionsFromRoot(raw)
	cfg := defaultConfigForLoad(sections)

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	applyLegacyLoadCompatibility(data, raw, cfg)
	applyDefaults(cfg, sections.defaultApplyOptions())

	return cfg, nil
}
