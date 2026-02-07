package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
		DefaultModel:    "deepseek-chat",
		General: GeneralConfig{
			Language: "ja", // デフォルト: 日本語
		},
		ProviderModels: map[string]ProviderModelConfig{
			"deepseek": {
				DefaultModel:    "deepseek-chat",
				MaxOutputTokens: 16384,
				ModelOverrides: map[string]ModelOverride{
					"deepseek-chat":     {MaxOutputTokens: 8192},
					"deepseek-reasoner": {MaxOutputTokens: 64000},
				},
			},
			"openai": {
				DefaultModel:    "gpt-5.2",
				MaxOutputTokens: 16384,
				ModelOverrides: map[string]ModelOverride{
					"gpt-5.2": {MaxOutputTokens: 16384},
				},
			},
			"gemini": {
				DefaultModel:    "gemini-2.5-flash",
				MaxOutputTokens: 65536,
				ModelOverrides: map[string]ModelOverride{
					"gemini-2.5-flash":       {MaxOutputTokens: 65536},
					"gemini-3-flash-preview": {MaxOutputTokens: 65536},
				},
			},
			"claude": {
				DefaultModel:    "claude-sonnet-4-5-20250514",
				MaxOutputTokens: 16384,
				ModelOverrides: map[string]ModelOverride{
					"claude-sonnet-4-5-20250514": {MaxOutputTokens: 16384},
				},
			},
			"ollama": {
				DefaultModel:    "qwen2.5-coder:7b",
				MaxOutputTokens: 4096,
			},
			"groq": {
				DefaultModel:    "meta-llama/llama-4-scout-17b-16e-instruct",
				MaxOutputTokens: 8192,
			},
			"openrouter": {
				DefaultModel:    "anthropic/claude-opus-4.5",
				MaxOutputTokens: 32768,
			},
			"bedrock": {
				DefaultModel:    "global.anthropic.claude-opus-4-5-20251101-v1:0",
				MaxOutputTokens: 32768,
			},
		},
		Compression: CompressionConfig{
			AutoCompress:     true, // デフォルトON - コスト削減のため
			ThresholdTokens:  0,    // 0 = 使用率ベース
			ThresholdPercent: 80,   // 80%で自動圧縮
			KeepRecent:       20,   // 履歴を多めに保持
			PreferCompactAPI: true, // OpenAI Compact API 優先
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
			Timeout:      3600, // xhigh thinking 対応（1時間）
		},
		Diff: DiffConfig{
			ContextLines: 10,
		},
		ToolConfirm: ToolConfirmConfig{
			AutoApproveSafe:   true,
			AutoApproveMedium: false,
		},
		CommandAliases: map[string]string{
			"c": "config",
			"u": "use",
		},
		PromptCache: PromptCacheConfig{
			Enabled:    true, // デフォルトON（Claude使用時のコスト削減）
			MaxEntries: 100,
			TTLSeconds: 300, // 5分
		},
		Paste: PasteConfig{
			BracketedPaste: true, // デフォルトON - 複数行ペースト対応
			MaxLines:       10000,
			MaxBytes:       1048576,
			TimeoutSeconds: 60,
		},
		Streaming: StreamingConfig{
			IdleTimeoutSeconds: 3600, // xhigh thinking 対応（1時間）
			ShowFileInfo:       true,
			ShowSearchProgress: true,
			StreamBashOutput:   true,
		},
		Bash: BashConfig{
			SafetyLevel:     "permissive", // 確認出るので安全、利便性向上
			SafeCommands:    []string{},
			AllowPipe:       true,
			AllowRedirect:   true, // 利便性向上
			AllowInlineEdit: true, // 利便性向上
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
			// 旧設定（後方互換性）
			MaxParallelSteps: 3,
			AutoRetry:        10,
			// 新設定（Phase 3）
			Parallel:        false,
			MaxWorkers:      3,
			SupervisorModel: "",
			LightModel:      "",
			HeavyModel:      "",
			MaxRetry:        10,
			StepTimeout:     600, // 10分
			ConfirmLevel:    "dangerous",
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
		OpenAI: OpenAIConfig{
			ResponsesAPIModels: []string{
				"gpt-5.2-codex",
				"gpt-5.1-codex",
				"gpt-5.1-codex-max",
				"gpt-5-codex",
				"gpt-5.2",
			},
		},
		RepoMap: RepoMapConfig{
			Enabled:   false, // デフォルトOFF
			MaxTokens: 0,     // 0 = 自動計算
		},
		Thinking: ThinkingConfig{
			Enabled: false,
			Level:   "medium",
		},
		Output: OutputConfig{
			MaxLines: 5, // デフォルト5行で折りたたみ
		},
		WebSearch: WebSearchConfig{
			CacheEnabled: true,
			CacheTTL:     1800, // 30分
			CacheSize:    100,
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

	// デフォルト値で初期化してからYAMLをマージ
	// これにより、YAMLで明示的に設定されたフィールドのみが上書きされる
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 追加のデフォルト値を適用（ネストされた構造体用）
	applyDefaults(cfg)

	return cfg, nil
}

// applyDefaults はデフォルト値を適用
func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "deepseek"
	}
	if cfg.ProviderModels == nil {
		cfg.ProviderModels = defaults.ProviderModels
	} else {
		// ProviderModels の MaxOutputTokens が 0（未設定）の場合、デフォルト値を適用
		// YAML で provider_models を設定すると map が上書きされ MaxOutputTokens が消えるため
		for name, pm := range cfg.ProviderModels {
			if pm.MaxOutputTokens == 0 {
				if defaultPM, ok := defaults.ProviderModels[name]; ok {
					pm.MaxOutputTokens = defaultPM.MaxOutputTokens
					cfg.ProviderModels[name] = pm
				}
			}
		}
	}

	// ネストされた構造体のデフォルト値を適用（YAMLで省略された場合）
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
	// Paste: 他のフィールドがすべてデフォルト値の場合、BracketedPaste もデフォルト適用
	// （既存の設定ファイルに bracketed_paste がない場合に true にするため）
	if cfg.Paste.MaxLines == 0 && cfg.Paste.MaxBytes == 0 && cfg.Paste.TimeoutSeconds == 0 {
		// Paste セクションが未設定 → 全てデフォルト適用
		cfg.Paste = defaults.Paste
	} else {
		// 個別フィールドのデフォルト適用
		if cfg.Paste.MaxLines == 0 {
			cfg.Paste.MaxLines = defaults.Paste.MaxLines
		}
		if cfg.Paste.MaxBytes == 0 {
			cfg.Paste.MaxBytes = defaults.Paste.MaxBytes
		}
		if cfg.Paste.TimeoutSeconds == 0 {
			cfg.Paste.TimeoutSeconds = defaults.Paste.TimeoutSeconds
		}
		// BracketedPaste: 明示的に false に設定されていない限り、デフォルト (true) を適用
		// 注: YAML で bracketed_paste: false を明示的に設定した場合のみ false になる
		// 既存の設定ファイル（フィールドがない）では true にする
	}
	if cfg.Streaming.IdleTimeoutSeconds == 0 {
		cfg.Streaming.IdleTimeoutSeconds = defaults.Streaming.IdleTimeoutSeconds
	}
	if cfg.PlanMode.MaxParallelSteps == 0 {
		cfg.PlanMode.MaxParallelSteps = defaults.PlanMode.MaxParallelSteps
	}
	// PlanMode マイグレーション（Phase 3）
	MigratePlanModeConfig(&cfg.PlanMode)
	// PlanMode 新フィールドのデフォルト適用
	if cfg.PlanMode.MaxWorkers == 0 {
		cfg.PlanMode.MaxWorkers = defaults.PlanMode.MaxWorkers
	}
	if cfg.PlanMode.MaxRetry == 0 {
		cfg.PlanMode.MaxRetry = defaults.PlanMode.MaxRetry
	}
	if cfg.PlanMode.StepTimeout == 0 {
		cfg.PlanMode.StepTimeout = defaults.PlanMode.StepTimeout
	}
	if cfg.PlanMode.ConfirmLevel == "" {
		cfg.PlanMode.ConfirmLevel = defaults.PlanMode.ConfirmLevel
	}
	// LSP設定のデフォルト適用
	// 注: cfg.LSP.Enabled が false の場合と、設定ファイルに LSP セクションがない場合を区別するため
	// Servers が nil の場合のみデフォルトを適用する
	if cfg.LSP.Servers == nil {
		cfg.LSP = defaults.LSP
	}
	// Note: Diff.ContextLines は0が有効値なので、デフォルト適用は行わない
	// Thinking: Level が空の場合はデフォルト適用
	if cfg.Thinking.Level == "" {
		cfg.Thinking.Level = defaults.Thinking.Level
	}
	// WebSearch: 全てゼロ値の場合のみデフォルト適用
	if !cfg.WebSearch.CacheEnabled && cfg.WebSearch.CacheTTL == 0 && cfg.WebSearch.CacheSize == 0 {
		cfg.WebSearch = defaults.WebSearch
	}
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

// SaveConfig は設定ファイルを保存し、グローバル設定を更新
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
	header := fmt.Sprintf("# XELYON CLI 設定\n# Providers: %s\n# 各プロバイダーのモデル設定は provider_models で管理されます\n\n", strings.Join(GetDisplayProviders(), ", "))
	fullData := []byte(header + string(data))

	if err := os.WriteFile(configPath, fullData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// グローバル設定を更新
	SetGlobalConfig(cfg)

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
	// Bracketed Paste Mode の制御（XELYON_BRACKETED_PASTE=0 で無効化）
	if val := os.Getenv("XELYON_BRACKETED_PASTE"); val == "0" || val == "false" {
		c.Paste.BracketedPaste = false
	}
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

// MigratePlanModeConfig は旧設定を新設定にマイグレーション（Phase 3）
// - max_parallel_steps → max_workers
// - auto_retry → max_retry
// また、デフォルト値も設定する
func MigratePlanModeConfig(cfg *PlanModeConfig) {
	// マイグレーション: max_parallel_steps → max_workers
	if cfg.MaxParallelSteps > 0 && cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = cfg.MaxParallelSteps
	}
	// マイグレーション: auto_retry → max_retry
	if cfg.AutoRetry > 0 && cfg.MaxRetry == 0 {
		cfg.MaxRetry = cfg.AutoRetry
	}

	// デフォルト値の設定
	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = 3
	}
	if cfg.MaxRetry == 0 {
		cfg.MaxRetry = 10
	}
	if cfg.StepTimeout == 0 {
		cfg.StepTimeout = 600 // 10分
	}
	if cfg.ConfirmLevel == "" {
		cfg.ConfirmLevel = "dangerous"
	}
}
