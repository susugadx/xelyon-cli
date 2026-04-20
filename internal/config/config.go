package config

import "encoding/json"

const (
	configDir  = ".xelyon"
	configFile = "config.yaml"
)

// CloneConfig は設定をディープコピーして返す。
func CloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return DefaultConfig()
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		clone := *cfg
		clone.providerModelsStore = cfg.providerModelsStore.clone()
		return &clone
	}

	var cloned Config
	if err := json.Unmarshal(data, &cloned); err != nil {
		clone := *cfg
		clone.providerModelsStore = cfg.providerModelsStore.clone()
		return &clone
	}
	cloned.providerModelsStore = cfg.providerModelsStore.clone()
	return &cloned
}

// LoadConfig は設定ファイルを読み込む
func LoadConfig() (*Config, error) {
	return loadConfig()
}

// LoadConfigWithValidation は設定ファイルを読み込み、バリデーションを実行
// バリデーションエラーがあっても設定は返す（警告のみ）
func LoadConfigWithValidation() (*Config, ValidationResult, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, ValidationResult{}, err
	}

	result := ValidateConfig(cfg)
	return cfg, result, nil
}

// SaveConfig は設定ファイルを保存する。
func SaveConfig(cfg *Config) error {
	return saveConfig(cfg)
}

// ValidateModel は任意のモデル名を受け付ける（後方互換のため残す）
// 注: v0.16.0以降、モデル名の検証は行わない
func ValidateModel(model string) bool {
	return true
}
