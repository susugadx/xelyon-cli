package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".xelyon"
	configFile = "config.yaml"
)

// Config はXELYON CLIの設定
type Config struct {
	DefaultProvider string                         `yaml:"default_provider"`
	DefaultModel    string                         `yaml:"default_model"`
	ProviderModels  map[string]ProviderModelConfig `yaml:"provider_models"`
	// 将来の拡張用
	// Cloud CloudConfig `yaml:"cloud,omitempty"`
}

// ProviderModelConfig はプロバイダーごとのモデル設定
type ProviderModelConfig struct {
	DefaultModel    string   `yaml:"default_model"`
	AvailableModels []string `yaml:"available_models,omitempty"`
}

// CloudConfig はXELYON Cloud連携設定（将来用）
// type CloudConfig struct {
// 	Enabled bool   `yaml:"enabled"`
// 	UserID  string `yaml:"user_id"`
// 	Token   string `yaml:"token"`
// }

// DefaultConfig はデフォルト設定
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "deepseek",
		DefaultModel:    "deepseek-coder",
		ProviderModels: map[string]ProviderModelConfig{
			"deepseek": {
				DefaultModel:    "deepseek-coder",
				AvailableModels: []string{"deepseek-chat", "deepseek-coder", "deepseek-reasoner"},
			},
			"openai": {
				DefaultModel:    "gpt-4o",
				AvailableModels: []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo"},
			},
			"gemini": {
				DefaultModel:    "gemini-2.0-flash-exp",
				AvailableModels: []string{"gemini-2.0-flash-exp", "gemini-1.5-pro", "gemini-1.5-flash"},
			},
			"claude": {
				DefaultModel:    "claude-sonnet-4-20250514",
				AvailableModels: []string{"claude-sonnet-4-20250514", "claude-opus-4", "claude-haiku-3-5-20241022"},
			},
			"ollama": {
				DefaultModel:    "llama3",
				AvailableModels: []string{}, // 自動検出されるため空
			},
			"groq": {
				DefaultModel:    "llama3-70b-8192",
				AvailableModels: []string{"llama3-70b-8192", "llama3-8b-8192", "mixtral-8x7b-32768"},
			},
		},
	}
}

// LoadConfig は設定ファイルを読み込む
func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// 設定ファイルが存在しない場合はデフォルトを作成
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
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

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// デフォルト値を適用
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "deepseek"
	}
	if cfg.ProviderModels == nil {
		cfg.ProviderModels = DefaultConfig().ProviderModels
	}

	return &cfg, nil
}

// SaveConfig は設定ファイルを保存
func SaveConfig(cfg *Config) error {
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
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// ヘッダーコメント追加
	header := "# XELYON CLI 設定\n# Providers: deepseek, openai, gemini, claude, ollama, groq\n# 各プロバイダーのモデル設定は provider_models で管理されます\n\n"
	fullData := []byte(header + string(data))

	if err := os.WriteFile(configPath, fullData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigPath は設定ファイルのパスを返す
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, configDir, configFile), nil
}

// ValidateModel は任意のモデル名を受け付ける（後方互換のため残す）
// 注: v0.16.0以降、モデル名の検証は行わない
func ValidateModel(model string) bool {
	// 任意のモデル名を許可
	return true
}

// GetModelForProvider はプロバイダーに対応するデフォルトモデルを取得
func (c *Config) GetModelForProvider(provider string) string {
	if providerConfig, ok := c.ProviderModels[provider]; ok {
		return providerConfig.DefaultModel
	}
	return "" // フォールバック
}

// ValidateModelForProvider はモデル名がプロバイダーに対応しているか検証
func (c *Config) ValidateModelForProvider(provider, model string) bool {
	providerConfig, ok := c.ProviderModels[provider]
	if !ok {
		return false // プロバイダー不明
	}

	// available_modelsが空ならバリデーションスキップ（Ollama等）
	if len(providerConfig.AvailableModels) == 0 {
		return true
	}

	// モデル名がリストに含まれているか確認
	for _, m := range providerConfig.AvailableModels {
		if m == model {
			return true
		}
	}
	return false
}
