package config

import (
	"encoding/json"
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

// CloneConfig は設定をディープコピーして返す。
func CloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return DefaultConfig()
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		clone := *cfg
		return &clone
	}

	var cloned Config
	if err := json.Unmarshal(data, &cloned); err != nil {
		clone := *cfg
		return &clone
	}
	return &cloned
}

// DefaultConfig はデフォルト設定
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "deepseek",
		DefaultModel:    "deepseek-chat",
		General: GeneralConfig{
			UILanguage:    "auto", // デフォルト: 自動判定（フォールバック: ja）
			ToolLoopLimit: 0,      // 内部既定: 0 = unlimited tool loop
		},
		ProviderModels: map[string]ProviderModelConfig{
			"deepseek": {
				DefaultModel:    "deepseek-chat",
				MaxOutputTokens: 16384,
			},
			"openai": {
				DefaultModel:    "gpt-5.4",
				MaxOutputTokens: 16384,
			},
			"gemini": {
				DefaultModel:    "gemini-3.1-pro-preview-customtools",
				MaxOutputTokens: 65536,
			},
			"claude": {
				DefaultModel:     "claude-sonnet-4-6",
				MaxOutputTokens:  64000,
				AnthropicVersion: "2023-06-01",
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
				DefaultModel:    "anthropic/claude-sonnet-4.6",
				MaxOutputTokens: 64000,
			},
			"bedrock": {
				DefaultModel:     "global.anthropic.claude-sonnet-4-6-v1",
				MaxOutputTokens:  64000,
				AnthropicVersion: "bedrock-2023-05-31",
			},
		},
		Compression: CompressionConfig{
			Enabled:              true, // デフォルトON - コスト削減のため
			TriggerPercent:       80,   // 80%で自動圧縮
			KeepRecent:           20,   // 履歴を多めに保持
			ThresholdTokens:      0,    // 内部: 0 = 使用率ベース
			TokenThreshold:       0,    // 内部: 0 = カスタム絶対閾値を無効化
			Model:                "",   // 内部: 空 = プロバイダー別デフォルト圧縮モデル
			PreferCompactAPI:     true, // 内部: OpenAI Compact API 優先
			ClaudeCompaction:     true, // 内部: Claude Compaction 優先
			CompactionTrigger:    150000,
			ClearToolUses:        true, // 内部: Claude系の tool_use/tool_result clearing
			ClearToolUsesTrigger: 80000,
			ClearToolInputs:      false,
			ProviderThresholds: map[string]int{
				"gemini":             180000,
				"claude":             150000,
				"bedrock":            150000,
				"deepseek":           80000, // 128K window に対して出力/推論 headroom を残す安全側の値
				"openai":             100000,
				"openai:gpt-5.4":     260000, // 272K pricing cliff 手前
				"openai:gpt-5.4-pro": 260000,
				"openrouter":         120000,
			},
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
		Execution: ExecutionConfig{
			Mode:              string(ExecutionBalanced),
			AlwaysConfirm:     []string{},
			SafeShellCommands: []string{},
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
			Enabled:  true,
			CacheTTL: "5m",
		},
		Paste: PasteConfig{
			BracketedPaste: true, // デフォルトON - 複数行ペースト対応
			MaxLines:       10000,
			MaxBytes:       1048576,
			TimeoutSeconds: 60,
		},
		Streaming: StreamingConfig{
			IdleTimeoutSeconds:     30,  // チャンク間隔タイムアウト（30秒）
			ThinkingTimeoutSeconds: 120, // thinking専用: text/FC が来なければタイムアウト（最大リトライ2回=360秒）
			ShowFileInfo:           true,
			ShowSearchProgress:     true,
			StreamBashOutput:       true,
		},
		Bash: BashConfig{
			SafetyLevel:     "permissive", // 確認出るので安全、利便性向上
			SafeCommands:    []string{},
			AllowRedirect:   true, // 利便性向上
			AllowInlineEdit: true, // 利便性向上
		},
		ListDir: ListDirConfig{
			AdditionalIgnoreDirs: []string{},
		},
		ProjectMap: ProjectMapConfig{
			Enabled:              true,
			ContextRatio:         ProjectMapContextRatioDefault,
			AdditionalIgnoreDirs: []string{},
		},

		GitStage: GitStageConfig{
			BatchConfirm: true,
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
			ResponsesAPIModels: []string{}, // プレフィックスマッチで自動判定（ユーザーカスタムモデル追加用）
		},
		Thinking: ThinkingConfig{
			Enabled: false,
			Level:   "medium",
		},
		Output: OutputConfig{
			MaxLines:         5,  // デフォルト5行で折りたたみ
			AssistantUpdates: "", // 空 = Normal Mode は phase / Plan Mode は verbose
		},
		WebSearch: WebSearchConfig{
			CacheEnabled: true,
			CacheTTL:     3600, // 1時間
			CacheSize:    50,
		},
		SubAgent: SubAgentConfig{
			Enabled:       true,
			DefaultModel:  "gpt-5.4-mini",
			DefaultEffort: "",
			MaxConcurrent: 1,
		},
		UtilityModel: UtilityModelConfig{
			Enabled:  false,
			Provider: "",
			Model:    "",
			Tasks:    []string{"web_search_compaction"},
		},
		MCP: MCPConfig{
			Enabled:  true,  // デフォルトON - MCP接続有効
			Headless: false, // デフォルトOFF - Headlessモードでは接続しない
		},
		Hooks: HooksConfig{
			OnCompletion: nil, // デフォルト: フックなし
			Timeout:      60,  // 60秒タイムアウト
			MaxRetry:     3,   // フック失敗時の最大リトライ回数
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

	// 旧キーからの migration（後方互換）
	migrateOldKeys(data, cfg)

	// 追加のデフォルト値を適用（ネストされた構造体用）
	applyDefaults(cfg)

	return cfg, nil
}

// migrateOldKeys は旧設定キーを新キーに読み替える。
// ルール: 新キーが YAML に明示されていれば新キーを優先。旧キーは新キーが存在しない場合の補完にのみ使う。
func migrateOldKeys(data []byte, cfg *Config) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}

	// general.language → general.ui_language
	if general, ok := raw["general"].(map[string]interface{}); ok {
		if lang, ok := general["language"].(string); ok && lang != "" {
			if _, hasNewKey := general["ui_language"]; !hasNewKey {
				cfg.General.UILanguage = lang
			}
		}
	}

	// tool_confirm + bash → execution の補完（execution が未設定の場合のみ）
	if _, hasExecution := raw["execution"]; !hasExecution {
		// execution セクションがなければ旧設定から推定
		if tc, ok := raw["tool_confirm"].(map[string]interface{}); ok {
			if autoMedium, ok := tc["auto_approve_medium"].(bool); ok && autoMedium {
				// auto_approve_medium: true → trusted 相当
				cfg.Execution.Mode = string(ExecutionTrusted)
			}
		}
	}

	// compression.auto_compress → compression.enabled
	// compression.threshold_percent → compression.trigger_percent
	if comp, ok := raw["compression"].(map[string]interface{}); ok {
		if autoCompress, ok := comp["auto_compress"].(bool); ok {
			if _, hasNewKey := comp["enabled"]; !hasNewKey {
				cfg.Compression.Enabled = autoCompress
			}
		}
		if thresholdPercent, ok := comp["threshold_percent"]; ok {
			if pct := toInt(thresholdPercent); pct > 0 {
				if _, hasNewKey := comp["trigger_percent"]; !hasNewKey {
					cfg.Compression.TriggerPercent = pct
				}
			}
		}
	}
}

// toInt は interface{} から int を取得する（YAML パーサーが返す型に対応）。
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// applyDefaults はデフォルト値と内部正規化を適用する。
// 内部設定の補正（tool_loop_limit 負値→0 など）もこのフェーズで行う。
func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	// --- 内部設定の正規化 ---
	if cfg.General.ToolLoopLimit < 0 {
		cfg.General.ToolLoopLimit = 0 // 負値は 0（無制限）にサイレント補正
	}
	// Execution: mode が空なら balanced をデフォルト適用
	if cfg.Execution.Mode == "" {
		cfg.Execution.Mode = defaults.Execution.Mode
	}

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
	// Compression: ゼロ値のフィールドのみデフォルト適用（構造体全体の上書きはしない）
	if cfg.Compression.TriggerPercent == 0 {
		cfg.Compression.TriggerPercent = defaults.Compression.TriggerPercent
	}
	if cfg.Compression.KeepRecent == 0 {
		cfg.Compression.KeepRecent = defaults.Compression.KeepRecent
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
	// Hooks: Timeout が 0 の場合はデフォルト適用
	if cfg.Hooks.Timeout == 0 {
		cfg.Hooks.Timeout = defaults.Hooks.Timeout
	}
	if cfg.Output.AssistantUpdates != "" {
		switch strings.ToLower(cfg.Output.AssistantUpdates) {
		case "verbose", "phase", "off":
			cfg.Output.AssistantUpdates = strings.ToLower(cfg.Output.AssistantUpdates)
		default:
			cfg.Output.AssistantUpdates = defaults.Output.AssistantUpdates
		}
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

// SaveConfig は設定ファイルを保存する。
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
	// 内部設定の環境変数オーバーライド（後方互換）
	applyEnvInt("XELYON_LOOP_THRESHOLD", func(n int) bool { return n > 0 },
		func(n int) { c.LoopDetection.Threshold = n }, "positive integer")
	applyEnvInt("XELYON_DIFF_CONTEXT_LINES", func(n int) bool { return n >= 0 },
		func(n int) { c.Diff.ContextLines = n }, "non-negative integer")
}

// applyEnvInt は環境変数を整数として読み込み、バリデーション後に適用する。
// 不正な値の場合は stderr に警告を出力する。
func applyEnvInt(envKey string, valid func(int) bool, apply func(int), expect string) {
	val := os.Getenv(envKey)
	if val == "" {
		return
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: invalid %s=%q (expected %s)\n", envKey, val, expect)
		return
	}
	if !valid(n) {
		fmt.Fprintf(os.Stderr, "Warning: invalid %s=%q (expected %s)\n", envKey, val, expect)
		return
	}
	apply(n)
}

// ApplyFlagOverrides はCLIフラグで設定を上書き（内部設定）
func (c *Config) ApplyFlagOverrides(loopThreshold, diffLines *int) {
	if loopThreshold != nil && *loopThreshold > 0 {
		c.LoopDetection.Threshold = *loopThreshold
	}
	if diffLines != nil && *diffLines >= 0 {
		c.Diff.ContextLines = *diffLines
	}
}
