package config

import "strings"

// Config はXELYON CLIの設定
type Config struct {
	DefaultProvider string                         `yaml:"default_provider"`
	DefaultModel    string                         `yaml:"default_model"`
	ProviderModels  map[string]ProviderModelConfig `yaml:"provider_models"`
	General         GeneralConfig                  `yaml:"general"`
	Compression     CompressionConfig              `yaml:"compression"`
	LoopDetection   LoopDetectionConfig            `yaml:"loop_detection"`
	APIRetry        APIRetryConfig                 `yaml:"api_retry"`
	Diff            DiffConfig                     `yaml:"diff"`
	Execution       ExecutionConfig                `yaml:"execution"`
	ToolConfirm     ToolConfirmConfig              `yaml:"tool_confirm"`
	CommandAliases  map[string]string              `yaml:"command_aliases,omitempty"` // コマンドエイリアス
	PromptCache     PromptCacheConfig              `yaml:"prompt_cache"`
	Paste           PasteConfig                    `yaml:"paste"`
	Streaming       StreamingConfig                `yaml:"streaming"`
	Bash            BashConfig                     `yaml:"bash"`
	ListDir         ListDirConfig                  `yaml:"list_dir"`
	ProjectMap      ProjectMapConfig               `yaml:"project_map"`

	GitStage       GitStageConfig     `yaml:"git_stage"`
	PlanMode       PlanModeConfig     `yaml:"plan_mode"`
	LSP            LSPConfig          `yaml:"lsp"`
	OpenAI         OpenAIConfig       `yaml:"openai"`
	Thinking       ThinkingConfig     `yaml:"thinking"`
	Output         OutputConfig       `yaml:"output"`
	WebSearch      WebSearchConfig    `yaml:"web_search"`
	SubAgent       SubAgentConfig     `yaml:"sub_agent"`
	UtilityModel   UtilityModelConfig `yaml:"utility_model"`
	MCP            MCPConfig          `yaml:"mcp"`
	Hooks          HooksConfig        `yaml:"hooks"`
	SubAgentPrompt string             `yaml:"-"`
	// 将来の拡張用
	// Cloud CloudConfig `yaml:"cloud,omitempty"`
}

// OutputConfig はツール出力表示の設定
type OutputConfig struct {
	MaxLines         int    `yaml:"max_lines"`         // 折りたたみ前の最大表示行数（デフォルト: 5）
	AssistantUpdates string `yaml:"assistant_updates"` // assistant prose の中間表示制御（verbose/phase/off、空=モード別デフォルト）
}

// GeneralConfig は一般設定
type GeneralConfig struct {
	UILanguage    string `yaml:"ui_language"`     // 表示言語（auto, ja, en）デフォルト: auto
	ToolLoopLimit int    `yaml:"tool_loop_limit"` // Max tool loop iterations (0 = unlimited, default) — 内部既定値
}

// CompressionConfig は会話履歴圧縮の設定
//
// user-facing 設定: Enabled, TriggerPercent, KeepRecent
// 以下は内部既定値として保持（user-facing config には非表示）
type CompressionConfig struct {
	Enabled              bool           `yaml:"enabled"`                                // 自動圧縮を有効化（デフォルト: true）
	TriggerPercent       int            `yaml:"trigger_percent"`                        // 自動圧縮の使用率閾値（デフォルト: 80%）
	KeepRecent           int            `yaml:"keep_recent"`                            // 保持する最新メッセージ数
	ThresholdTokens      int            `yaml:"threshold_tokens"`                       // 内部: トークン閾値（0 = 使用率ベース）
	TokenThreshold       int            `yaml:"token_threshold" json:"token_threshold"` // 内部: カスタム絶対閾値（デフォルト: 0 = 無効）
	Model                string         `yaml:"model" json:"model"`                     // 内部: 圧縮用モデル名
	PreferCompactAPI     bool           `yaml:"prefer_compact_api"`                     // 内部: OpenAI Compact API 優先
	ClaudeCompaction     bool           `yaml:"claude_compaction"`                      // 内部: Claude compact_20260112
	CompactionTrigger    int            `yaml:"compaction_trigger"`                     // 内部: compact トリガー閾値
	ClearToolUses        bool           `yaml:"clear_tool_uses"`                        // 内部: server-side tool clearing
	ClearToolUsesTrigger int            `yaml:"clear_tool_uses_trigger"`                // 内部: clear_tool_uses トリガー閾値
	ClearToolInputs      bool           `yaml:"clear_tool_inputs"`                      // 内部: tool_use 入力クリア
	ProviderThresholds   map[string]int `yaml:"provider_thresholds,omitempty"`          // 内部: provider/model 別閾値
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
	ContextLines  int `yaml:"context_lines"`   // 差分表示行数（デフォルト10、0で省略なし）
	MaxTotalLines int `yaml:"max_total_lines"` // 差分表示の最大行数（0で無制限）
}

// ToolConfirmConfig はツール実行確認の設定
type ToolConfirmConfig struct {
	AutoApproveSafe   bool `yaml:"auto_approve_safe"`   // SafetyHigh（read_file等）を確認なしで実行（デフォルトtrue）
	AutoApproveMedium bool `yaml:"auto_approve_medium"` // SafetyMedium（str_replace等）を確認なしで実行（デフォルトfalse）
}

// PromptCacheConfig はプロンプトキャッシュの設定（Anthropic API cache_control）
type PromptCacheConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CacheTTL string `yaml:"cache_ttl"` // キャッシュTTL（デフォルト: "5m"、"1h" で延長キャッシュ）
}

// PasteConfig はペーストモードの設定
type PasteConfig struct {
	BracketedPaste bool `yaml:"bracketed_paste"` // Bracketed Paste Mode を有効化（デフォルト: true）
	MaxLines       int  `yaml:"max_lines"`       // 最大行数（デフォルト10000）
	MaxBytes       int  `yaml:"max_bytes"`       // 最大バイト数（デフォルト1MB）
	TimeoutSeconds int  `yaml:"timeout_seconds"` // タイムアウト秒（デフォルト60）
}

// StreamingConfig はストリーミングレスポンスの設定
type StreamingConfig struct {
	IdleTimeoutSeconds     int  `yaml:"idle_timeout_seconds"`     // アイドルタイムアウト秒（デフォルト30）
	ThinkingTimeoutSeconds int  `yaml:"thinking_timeout_seconds"` // thinking専用タイムアウト秒（text/FC未受信時、デフォルト300）
	ShowFileInfo           bool `yaml:"show_file_info"`           // ファイル読み込み時にサイズ表示（デフォルト: true）
	ShowSearchProgress     bool `yaml:"show_search_progress"`     // 検索中に進捗表示（デフォルト: true）
	StreamBashOutput       bool `yaml:"stream_bash_output"`       // bashコマンド出力をストリーミング（デフォルト: true）
}

// BashConfig はbashツールの設定
type BashConfig struct {
	SafetyLevel     string   `yaml:"safety_level"`      // strict, moderate, permissive（デフォルト: moderate）
	SafeCommands    []string `yaml:"safe_commands"`     // 追加の安全コマンド
	AllowRedirect   bool     `yaml:"allow_redirect"`    // リダイレクトを許可（デフォルト: true）
	AllowInlineEdit bool     `yaml:"allow_inline_edit"` // sed -i等を許可（デフォルト: false）
}

// ListDirConfig は list_dir ツールの設定
type ListDirConfig struct {
	AdditionalIgnoreDirs []string `yaml:"additional_ignore_dirs"` // デフォルト除外に追加するディレクトリ名
}

// ProjectMapConfig はプロジェクト構造マップの設定
type ProjectMapConfig struct {
	Enabled              bool     `yaml:"enabled"`                // 起動時に Project Map を生成・注入
	ContextRatio         float64  `yaml:"context_ratio"`          // ProjectMap のベース比率（大規模 repo では自動的に引き上げ）
	AdditionalIgnoreDirs []string `yaml:"additional_ignore_dirs"` // デフォルト除外に追加するディレクトリ名
}

// GitStageConfig はgit_addツールの設定
type GitStageConfig struct {
	BatchConfirm bool `yaml:"batch_confirm"` // 複数ファイルのバッチ確認UI（デフォルト: true）
}

// PlanModeConfig は Plan Mode の設定
type PlanModeConfig struct {
	MaxRetry               int  `yaml:"max_retry"`                 // 自動リトライ回数（デフォルト: 10）
	StepTimeout            int  `yaml:"step_timeout"`              // ステップタイムアウト秒（デフォルト: 600）
	ClearContextOnApproval bool `yaml:"clear_context_on_approval"` // Clear investigation context after plan approval
}

// OpenAIConfig は OpenAI プロバイダーの設定
type OpenAIConfig struct {
	ResponsesAPIModels []string `yaml:"responses_api_models"` // Responses API を使用するモデル
}

// ThinkingConfig は Extended Thinking の設定
type ThinkingConfig struct {
	Enabled bool   `yaml:"enabled"` // デフォルト: false
	Level   string `yaml:"level"`   // low/medium/high/xhigh（デフォルト: medium）
}

// WebSearchConfig は Web 検索プロバイダーとキャッシュの設定
type WebSearchConfig struct {
	Provider     string `yaml:"provider,omitempty"` // 検索プロバイダー（openai/gemini/claude、未設定時はメインプロバイダーを使用）
	CacheEnabled bool   `yaml:"cache_enabled"`      // キャッシュを有効化（デフォルト: true）
	CacheTTL     int    `yaml:"cache_ttl"`          // キャッシュTTL秒数（デフォルト: 3600 = 1時間）
	CacheSize    int    `yaml:"cache_size"`         // 最大キャッシュ数（デフォルト: 50）
}

// SubAgentConfig はサブエージェント設定
type SubAgentConfig struct {
	Enabled       bool   `yaml:"enabled"`        // サブエージェント機能を有効化（デフォルト: true）
	DefaultModel  string `yaml:"default_model"`  // サブエージェント既定モデル（空でメイン provider の最安モデルを自動選択）
	DefaultEffort string `yaml:"default_effort"` // サブエージェント既定推論強度（空または off で無効）
	MaxConcurrent int    `yaml:"max_concurrent"` // 同時実行上限
}

// UtilityModelConfig は軽量な補助タスク専用モデルの設定
type UtilityModelConfig struct {
	Enabled  bool     `yaml:"enabled"`            // utility model を有効化
	Provider string   `yaml:"provider,omitempty"` // 使用プロバイダー（openai/gemini/claude 等）
	Model    string   `yaml:"model,omitempty"`    // 使用モデル（空 = provider_models の default_model）
	Tasks    []string `yaml:"tasks,omitempty"`    // 許可する軽量タスク
}

// MCPConfig は MCP (Model Context Protocol) サーバー接続の設定
type MCPConfig struct {
	Enabled  bool `yaml:"enabled"`  // MCP接続を有効化（デフォルト: true）
	Headless bool `yaml:"headless"` // Headlessモードでも接続（デフォルト: false）
}

// HooksConfig はフック設定
type HooksConfig struct {
	OnCompletion   []string `yaml:"on_completion"`    // 完了時に実行するコマンド
	OnStepComplete []string `yaml:"on_step_complete"` // ステップ完了時に実行するコマンド
	Timeout        int      `yaml:"timeout"`          // コマンドタイムアウト秒（デフォルト: 60）
	MaxRetry       int      `yaml:"max_retry"`        // フック失敗時の最大リトライ回数（デフォルト: 3）
}

// LSPConfig は LSP (Language Server Protocol) 連携の設定
type LSPConfig struct {
	Enabled           bool                       `yaml:"enabled"`                       // LSP機能を有効化（デフォルト: true）
	SkipInstallPrompt bool                       `yaml:"skip_install_prompt,omitempty"` // インストール提案をスキップ
	Servers           map[string]LSPServerConfig `yaml:"servers,omitempty"`
}

// LSPServerConfig は個別のLSPサーバー設定
type LSPServerConfig struct {
	Command  string   `yaml:"command"`            // サーバーコマンド（例: gopls, vtsls）
	Args     []string `yaml:"args,omitempty"`     // コマンド引数
	Disabled bool     `yaml:"disabled,omitempty"` // このサーバーを無効化
}

// ModelOverride はモデルごとの個別設定
type ModelOverride struct {
	MaxOutputTokens int `yaml:"max_output_tokens,omitempty"` // このモデル固有の最大出力トークン数
}

// ProviderModelConfig はプロバイダーごとのモデル設定
type ProviderModelConfig struct {
	DefaultModel     string                   `yaml:"default_model"`
	MaxOutputTokens  int                      `yaml:"max_output_tokens,omitempty"` // プロバイダー全体のデフォルト最大出力トークン数
	AnthropicVersion string                   `yaml:"anthropic_version,omitempty"` // Anthropic API バージョン
	AnthropicBeta    []string                 `yaml:"anthropic_beta,omitempty"`    // Anthropic Beta ヘッダー
	ModelOverrides   map[string]ModelOverride `yaml:"model_overrides,omitempty"`   // モデルごとの個別オーバーライド
}

// CloudConfig はXELYON Cloud連携設定（将来用）
// type CloudConfig struct {
// 	Enabled bool   `yaml:"enabled"`
// 	UserID  string `yaml:"user_id"`
// 	Token   string `yaml:"token"`
// }

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

// IsResponsesAPIModel はモデルが OpenAI Responses API を使用するか判定
// 対応モデルは prefix マッチで自動判定し、設定リストをフォールバックとして使用
func (c *Config) IsResponsesAPIModel(model string) bool {
	// GPT-5 シリーズ
	if strings.HasPrefix(model, "gpt-5") {
		return true
	}
	// GPT-4o シリーズ（gpt-4o, gpt-4o-mini 等）
	// 注意: gpt-4-turbo, gpt-4 は非対応なので gpt-4o のみ
	if strings.HasPrefix(model, "gpt-4o") {
		return true
	}
	// o-series reasoning モデル（o1, o3, o4 等）
	if strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4") {
		return true
	}
	// 設定リストによるフォールバック（ユーザーカスタムモデル用）
	for _, m := range c.OpenAI.ResponsesAPIModels {
		if m == model {
			return true
		}
	}
	return false
}
