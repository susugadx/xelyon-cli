package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".xelyon"
	configFile = "config.yaml"
)

var globalConfig *Config

// SetGlobalConfig はグローバル設定を保存
func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
}

// GetGlobalConfig はグローバル設定を取得
func GetGlobalConfig() *Config {
	if globalConfig == nil {
		globalConfig = DefaultConfig()
	}
	return globalConfig
}

// Config はXELYON CLIの設定
type Config struct {
	DefaultProvider string                         `yaml:"default_provider"`
	DefaultModel    string                         `yaml:"default_model"`
	ProviderModels  map[string]ProviderModelConfig `yaml:"provider_models"`
	Compression     CompressionConfig              `yaml:"compression,omitempty"`
	Backup          BackupConfig                   `yaml:"backup,omitempty"`
	LoopDetection   LoopDetectionConfig            `yaml:"loop_detection,omitempty"`
	APIRetry        APIRetryConfig                 `yaml:"api_retry,omitempty"`
	Diff            DiffConfig                     `yaml:"diff,omitempty"`
	ToolConfirm     ToolConfirmConfig              `yaml:"tool_confirm,omitempty"`
	CommandAliases  map[string]string              `yaml:"command_aliases,omitempty"` // コマンドエイリアス
	PromptCache     PromptCacheConfig              `yaml:"prompt_cache,omitempty"`
	Paste           PasteConfig                    `yaml:"paste,omitempty"`
	Streaming       StreamingConfig                `yaml:"streaming,omitempty"`
	Bash            BashConfig                     `yaml:"bash,omitempty"`
	CodeHealth      CodeHealthConfig               `yaml:"code_health,omitempty"`
	GitStage        GitStageConfig                 `yaml:"git_stage,omitempty"`
	// 将来の拡張用
	// Cloud CloudConfig `yaml:"cloud,omitempty"`
}

// CompressionConfig は会話履歴圧縮の設定
type CompressionConfig struct {
	AutoCompress    bool `yaml:"auto_compress"`    // 自動圧縮を有効化
	ThresholdTokens int  `yaml:"threshold_tokens"` // 自動圧縮のトークン閾値
	KeepRecent      int  `yaml:"keep_recent"`      // 保持する最新メッセージ数
}

// BackupConfig はバックアップファイルの設定
type BackupConfig struct {
	MaxGenerations int `yaml:"max_generations"` // 保持する世代数（デフォルト5）
}

// LoopDetectionConfig はループ検知の設定
type LoopDetectionConfig struct {
	Threshold int `yaml:"threshold"` // ループ検知回数（デフォルト3）
}

// APIRetryConfig はAPIリトライの設定
type APIRetryConfig struct {
	Count        int `yaml:"count"`         // リトライ回数（デフォルト3）
	InitialDelay int `yaml:"initial_delay"` // 初回待機秒数（デフォルト1）
	MaxDelay     int `yaml:"max_delay"`     // 最大待機秒数（デフォルト30）
	Timeout      int `yaml:"timeout"`       // APIタイムアウト秒数（デフォルト300=5分）
}

// DiffConfig は差分表示の設定
type DiffConfig struct {
	ContextLines int `yaml:"context_lines"` // 差分表示行数（デフォルト10、0で省略なし）
}

// ToolConfirmConfig はツール実行確認の設定
type ToolConfirmConfig struct {
	AutoApproveSafe bool `yaml:"auto_approve_safe"` // SafetyHigh（read_file等）を確認なしで実行（デフォルトtrue）
}

// PromptCacheConfig はプロンプトキャッシュの設定
//
// 目的: system prompt / repo map 等の生成コストを削減するためのキャッシュ。
// 現時点では in-memory キャッシュを想定（永続化なし）。
type PromptCacheConfig struct {
	Enabled    bool `yaml:"enabled"`
	MaxEntries int  `yaml:"max_entries"` // 0以下の場合はデフォルト適用
	TTLSeconds int  `yaml:"ttl_seconds"` // 0の場合はデフォルト適用（デフォルトTTL）
}

// PasteConfig はペーストモードの設定
type PasteConfig struct {
	MaxLines       int `yaml:"max_lines"`       // 最大行数（デフォルト10000）
	MaxBytes       int `yaml:"max_bytes"`       // 最大バイト数（デフォルト1MB）
	TimeoutSeconds int `yaml:"timeout_seconds"` // タイムアウト秒（デフォルト60）
}

// StreamingConfig はストリーミングレスポンスの設定
type StreamingConfig struct {
	IdleTimeoutSeconds int `yaml:"idle_timeout_seconds"` // アイドルタイムアウト秒（デフォルト30）
}

// BashConfig はbashツールの設定
type BashConfig struct {
	SafetyLevel     string   `yaml:"safety_level"`      // strict, moderate, permissive（デフォルト: moderate）
	SafeCommands    []string `yaml:"safe_commands"`     // 追加の安全コマンド
	AllowPipe       bool     `yaml:"allow_pipe"`        // パイプを許可（デフォルト: true - moderateで有効）
	AllowRedirect   bool     `yaml:"allow_redirect"`    // リダイレクトを許可（デフォルト: false）
	AllowInlineEdit bool     `yaml:"allow_inline_edit"` // sed -i等を許可（デフォルト: false）
}

// CodeHealthConfig はコード健全性チェックの設定
type CodeHealthConfig struct {
	Enabled          bool     `yaml:"enabled"`            // コード健全性チェックを有効化（デフォルト: true）
	MaxFileLines     int      `yaml:"max_file_lines"`     // ファイル行数上限（デフォルト: 300）
	MaxFunctionLines int      `yaml:"max_function_lines"` // 関数行数上限（デフォルト: 50）
	AutoSuggest      bool     `yaml:"auto_suggest"`       // 閾値超過時に自動で提案（デフォルト: true）
	OnChange         []string `yaml:"on_change"`          // 変更時チェック項目（check_file_size, check_function_size, check_duplication）
}

// GitStageConfig はgit_addツールの設定
type GitStageConfig struct {
	BatchConfirm bool `yaml:"batch_confirm"` // 複数ファイルのバッチ確認UI（デフォルト: true）
}

// ProviderModelConfig はプロバイダーごとのモデル設定
type ProviderModelConfig struct {
	DefaultModel string `yaml:"default_model"`
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
				DefaultModel: "deepseek-coder",
			},
			"openai": {
				DefaultModel: "gpt-5.2",
			},
			"gemini": {
				DefaultModel: "gemini-2.5-flash",
			},
			"claude": {
				DefaultModel: "claude-sonnet-4-5-20250514",
			},
			"ollama": {
				DefaultModel: "qwen2.5-coder:7b",
			},
			"groq": {
				DefaultModel: "meta-llama/llama-4-scout-17b-16e-instruct",
			},
		},
		Compression: CompressionConfig{
			AutoCompress:    false, // デフォルトは手動圧縮のみ
			ThresholdTokens: 40000,
			KeepRecent:      10,
		},
		Backup: BackupConfig{
			MaxGenerations: 5, // デフォルトは5世代保持
		},
		LoopDetection: LoopDetectionConfig{
			Threshold: 3, // デフォルトは3回
		},
		APIRetry: APIRetryConfig{
			Count:        3,   // デフォルトは3回
			InitialDelay: 1,   // デフォルトは1秒
			MaxDelay:     30,  // デフォルトは30秒
			Timeout:      300, // デフォルトは5分（300秒）
		},
		Diff: DiffConfig{
			ContextLines: 10, // デフォルトは10行
		},
		ToolConfirm: ToolConfirmConfig{
			AutoApproveSafe: true, // SafetyHigh（read_file等）は確認なしで実行
		},
		Paste: PasteConfig{
			MaxLines:       10000,   // デフォルト10000行
			MaxBytes:       1048576, // デフォルト1MB
			TimeoutSeconds: 60,      // デフォルト60秒
		},
		Streaming: StreamingConfig{
			IdleTimeoutSeconds: 30, // デフォルト30秒
		},
		Bash: BashConfig{
			SafetyLevel:     "moderate", // デフォルトはmoderate（パイプOK）
			SafeCommands:    []string{}, // 追加の安全コマンドなし
			AllowPipe:       true,       // パイプを許可
			AllowRedirect:   false,      // リダイレクトは不許可
			AllowInlineEdit: false,      // sed -i等は不許可
		},
		CodeHealth: CodeHealthConfig{
			Enabled:          true, // デフォルトは有効
			MaxFileLines:     300,  // デフォルトは300行
			MaxFunctionLines: 50,   // デフォルトは50行
			AutoSuggest:      true, // デフォルトは自動提案有効
			OnChange:         []string{"check_file_size", "check_function_size"},
		},
		GitStage: GitStageConfig{
			BatchConfirm: true, // デフォルトは有効
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

	// ネストされた構造体のデフォルト値を適用（YAMLで省略された場合）
	defaults := DefaultConfig()
	if cfg.LoopDetection.Threshold == 0 {
		cfg.LoopDetection = defaults.LoopDetection
	}
	if cfg.APIRetry.Count == 0 {
		cfg.APIRetry = defaults.APIRetry
	}
	// Timeoutだけ0の場合もデフォルト適用
	if cfg.APIRetry.Timeout == 0 {
		cfg.APIRetry.Timeout = defaults.APIRetry.Timeout
	}
	if cfg.Backup.MaxGenerations == 0 {
		cfg.Backup = defaults.Backup
	}
	if cfg.Compression.ThresholdTokens == 0 {
		cfg.Compression = defaults.Compression
	}
	// Paste設定のデフォルト適用
	if cfg.Paste.MaxLines == 0 {
		cfg.Paste.MaxLines = defaults.Paste.MaxLines
	}
	if cfg.Paste.MaxBytes == 0 {
		cfg.Paste.MaxBytes = defaults.Paste.MaxBytes
	}
	if cfg.Paste.TimeoutSeconds == 0 {
		cfg.Paste.TimeoutSeconds = defaults.Paste.TimeoutSeconds
	}
	// Streaming設定のデフォルト適用
	if cfg.Streaming.IdleTimeoutSeconds == 0 {
		cfg.Streaming.IdleTimeoutSeconds = defaults.Streaming.IdleTimeoutSeconds
	}
	// Note: Diff.ContextLines は0が有効値なので、デフォルト適用は行わない

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

	if err := os.WriteFile(configPath, fullData, 0600); err != nil {
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

// ValidateModelForProvider は任意のモデル名を受け付ける（後方互換のため残す）
// 注: v0.16.0以降、モデル名の検証は行わない
func (c *Config) ValidateModelForProvider(provider, model string) bool {
	// プロバイダーが存在するかのみチェック
	_, ok := c.ProviderModels[provider]
	return ok
}

// ApplyEnvironmentOverrides は環境変数で設定を上書き
func (c *Config) ApplyEnvironmentOverrides() {
	// ループ検知回数
	if val := os.Getenv("XELYON_LOOP_THRESHOLD"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.LoopDetection.Threshold = n
		}
	}

	// APIリトライ回数
	if val := os.Getenv("XELYON_API_RETRY_COUNT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.APIRetry.Count = n
		}
	}

	// API初回待機秒数
	if val := os.Getenv("XELYON_API_RETRY_INITIAL_DELAY"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.APIRetry.InitialDelay = n
		}
	}

	// API最大待機秒数
	if val := os.Getenv("XELYON_API_RETRY_MAX_DELAY"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.APIRetry.MaxDelay = n
		}
	}

	// 差分表示行数
	if val := os.Getenv("XELYON_DIFF_CONTEXT_LINES"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n >= 0 {
			c.Diff.ContextLines = n
		}
	}
}

// ApplyFlagOverrides はCLIフラグで設定を上書き
func (c *Config) ApplyFlagOverrides(loopThreshold, apiRetry, apiRetryDelay, diffLines *int) {
	if loopThreshold != nil && *loopThreshold > 0 {
		c.LoopDetection.Threshold = *loopThreshold
	}

	if apiRetry != nil && *apiRetry > 0 {
		c.APIRetry.Count = *apiRetry
	}

	if apiRetryDelay != nil && *apiRetryDelay > 0 {
		c.APIRetry.InitialDelay = *apiRetryDelay
	}

	if diffLines != nil && *diffLines >= 0 {
		c.Diff.ContextLines = *diffLines
	}
}
