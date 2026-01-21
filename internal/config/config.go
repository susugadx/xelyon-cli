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
			AutoCompress:     true, // デフォルトON - コスト削減のため
			ThresholdTokens:  0,    // 0 = 使用率ベース
			ThresholdPercent: 80,   // 80%で自動圧縮
			KeepRecent:       10,
		},
		Backup: BackupConfig{
			MaxGenerations: 5,
		},
		LoopDetection: LoopDetectionConfig{
			Threshold: 3,
		},
		APIRetry: APIRetryConfig{
			Count:        3,
			InitialDelay: 1,
			MaxDelay:     30,
			Timeout:      300,
		},
		Diff: DiffConfig{
			ContextLines: 10,
		},
		ToolConfirm: ToolConfirmConfig{
			AutoApproveSafe:   true,
			AutoApproveMedium: false,
		},
		Paste: PasteConfig{
			MaxLines:       10000,
			MaxBytes:       1048576,
			TimeoutSeconds: 60,
		},
		Streaming: StreamingConfig{
			IdleTimeoutSeconds: 30,
		},
		Bash: BashConfig{
			SafetyLevel:     "moderate",
			SafeCommands:    []string{},
			AllowPipe:       true,
			AllowRedirect:   false,
			AllowInlineEdit: false,
		},
		CodeHealth: CodeHealthConfig{
			Enabled:          true,
			MaxFileLines:     300,
			MaxFunctionLines: 50,
			AutoSuggest:      true,
			OnChange:         []string{"check_file_size", "check_function_size"},
		},
		GitStage: GitStageConfig{
			BatchConfirm: true,
		},
		PlanMode: PlanModeConfig{
			MaxParallelSteps: 3,
		},
		LSP: LSPConfig{
			Enabled: true,
			Servers: map[string]LSPServerConfig{
				// ===== Existing (4 languages) =====
				"go": {
					Command: "gopls",
					Args:    []string{},
				},
				"typescript": {
					Command: "vtsls",
					Args:    []string{"--stdio"},
				},
				"python": {
					Command: "pyright-langserver",
					Args:    []string{"--stdio"},
				},
				"rust": {
					Command: "rust-analyzer",
					Args:    []string{},
				},
				// ===== Tier 1: Backend languages (11 languages) =====
				"java": {
					Command: "jdtls",
					Args:    []string{},
				},
				"c": {
					Command: "clangd",
					Args:    []string{},
				},
				"cpp": {
					Command: "clangd",
					Args:    []string{},
				},
				"ruby": {
					Command: "solargraph",
					Args:    []string{"stdio"},
				},
				"kotlin": {
					Command: "kotlin-language-server",
					Args:    []string{},
				},
				"swift": {
					Command: "sourcekit-lsp",
					Args:    []string{},
				},
				"csharp": {
					Command: "csharp-ls",
					Args:    []string{},
				},
				"scala": {
					Command: "metals",
					Args:    []string{},
				},
				"php": {
					Command: "intelephense",
					Args:    []string{"--stdio"},
				},
				"elixir": {
					Command: "elixir-ls",
					Args:    []string{},
				},
				"lua": {
					Command: "lua-language-server",
					Args:    []string{},
				},
				// ===== Tier 2: Frontend languages (4 languages) =====
				"css": {
					Command: "vscode-css-language-server",
					Args:    []string{"--stdio"},
				},
				"html": {
					Command: "vscode-html-language-server",
					Args:    []string{"--stdio"},
				},
				"vue": {
					Command: "vue-language-server",
					Args:    []string{"--stdio"},
				},
				"svelte": {
					Command: "svelteserver",
					Args:    []string{"--stdio"},
				},
				// ===== Tier 3: Config/Script languages (5 languages) =====
				"yaml": {
					Command: "yaml-language-server",
					Args:    []string{"--stdio"},
				},
				"toml": {
					Command: "taplo",
					Args:    []string{"lsp", "stdio"},
				},
				"sql": {
					Command: "sqls",
					Args:    []string{},
				},
				"bash": {
					Command: "bash-language-server",
					Args:    []string{"start"},
				},
				"markdown": {
					Command: "marksman",
					Args:    []string{"server"},
				},
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
	applyDefaults(&cfg)

	return &cfg, nil
}

// applyDefaults はデフォルト値を適用
func applyDefaults(cfg *Config) {
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
	if cfg.APIRetry.Timeout == 0 {
		cfg.APIRetry.Timeout = defaults.APIRetry.Timeout
	}
	if cfg.Backup.MaxGenerations == 0 {
		cfg.Backup = defaults.Backup
	}
	// Compression: ThresholdTokens=0 かつ ThresholdPercent=0 の場合のみデフォルト適用
	// （ThresholdTokens=0 は「使用率ベース」を意味するため）
	if cfg.Compression.ThresholdTokens == 0 && cfg.Compression.ThresholdPercent == 0 {
		cfg.Compression = defaults.Compression
	}
	if cfg.Paste.MaxLines == 0 {
		cfg.Paste.MaxLines = defaults.Paste.MaxLines
	}
	if cfg.Paste.MaxBytes == 0 {
		cfg.Paste.MaxBytes = defaults.Paste.MaxBytes
	}
	if cfg.Paste.TimeoutSeconds == 0 {
		cfg.Paste.TimeoutSeconds = defaults.Paste.TimeoutSeconds
	}
	if cfg.Streaming.IdleTimeoutSeconds == 0 {
		cfg.Streaming.IdleTimeoutSeconds = defaults.Streaming.IdleTimeoutSeconds
	}
	if cfg.PlanMode.MaxParallelSteps == 0 {
		cfg.PlanMode.MaxParallelSteps = defaults.PlanMode.MaxParallelSteps
	}
	// LSP設定のデフォルト適用
	// 注: cfg.LSP.Enabled が false の場合と、設定ファイルに LSP セクションがない場合を区別するため
	// Servers が nil の場合のみデフォルトを適用する
	if cfg.LSP.Servers == nil {
		cfg.LSP = defaults.LSP
	}
	// Note: Diff.ContextLines は0が有効値なので、デフォルト適用は行わない
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
	return true
}

// ApplyEnvironmentOverrides は環境変数で設定を上書き
func (c *Config) ApplyEnvironmentOverrides() {
	if val := os.Getenv("XELYON_LOOP_THRESHOLD"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.LoopDetection.Threshold = n
		}
	}
	if val := os.Getenv("XELYON_API_RETRY_COUNT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.APIRetry.Count = n
		}
	}
	if val := os.Getenv("XELYON_API_RETRY_INITIAL_DELAY"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.APIRetry.InitialDelay = n
		}
	}
	if val := os.Getenv("XELYON_API_RETRY_MAX_DELAY"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.APIRetry.MaxDelay = n
		}
	}
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
