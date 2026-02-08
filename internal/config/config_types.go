package config

// Config はXELYON CLIの設定
type Config struct {
	DefaultProvider string                         `yaml:"default_provider"`
	DefaultModel    string                         `yaml:"default_model"`
	ProviderModels  map[string]ProviderModelConfig `yaml:"provider_models"`
	General         GeneralConfig                  `yaml:"general"`
	Compression     CompressionConfig              `yaml:"compression"`
	Backup          BackupConfig                   `yaml:"backup"`
	LoopDetection   LoopDetectionConfig            `yaml:"loop_detection"`
	APIRetry        APIRetryConfig                 `yaml:"api_retry"`
	Diff            DiffConfig                     `yaml:"diff"`
	ToolConfirm     ToolConfirmConfig              `yaml:"tool_confirm"`
	CommandAliases  map[string]string              `yaml:"command_aliases,omitempty"` // コマンドエイリアス
	PromptCache     PromptCacheConfig              `yaml:"prompt_cache"`
	Paste           PasteConfig                    `yaml:"paste"`
	Streaming       StreamingConfig                `yaml:"streaming"`
	Bash            BashConfig                     `yaml:"bash"`
	CodeHealth      CodeHealthConfig               `yaml:"code_health"`
	GitStage        GitStageConfig                 `yaml:"git_stage"`
	PlanMode        PlanModeConfig                 `yaml:"plan_mode"`
	LSP             LSPConfig                      `yaml:"lsp"`
	OpenAI          OpenAIConfig                   `yaml:"openai"`
	Thinking        ThinkingConfig                 `yaml:"thinking"`
	RepoMap         RepoMapConfig                  `yaml:"repomap"`
	Output          OutputConfig                   `yaml:"output"`
	WebSearch       WebSearchConfig                `yaml:"web_search"`
	// 将来の拡張用
	// Cloud CloudConfig `yaml:"cloud,omitempty"`
}

// OutputConfig はツール出力表示の設定
type OutputConfig struct {
	MaxLines int `yaml:"max_lines"` // 折りたたみ前の最大表示行数（デフォルト: 5）
}

// GeneralConfig は一般設定
type GeneralConfig struct {
	Language      string `yaml:"language"`        // 表示言語（ja, en）デフォルト: ja
	ToolLoopLimit int    `yaml:"tool_loop_limit"` // ツールループ最大回数（デフォルト: 80）
}

// CompressionConfig は会話履歴圧縮の設定
type CompressionConfig struct {
	AutoCompress     bool `yaml:"auto_compress"`      // 自動圧縮を有効化（デフォルト: true）
	ThresholdTokens  int  `yaml:"threshold_tokens"`   // 自動圧縮のトークン閾値（0 = 使用率ベース）
	ThresholdPercent int  `yaml:"threshold_percent"`  // 自動圧縮の使用率閾値（デフォルト: 80%）
	KeepRecent       int  `yaml:"keep_recent"`        // 保持する最新メッセージ数
	PreferCompactAPI bool `yaml:"prefer_compact_api"` // OpenAI Compact API を優先（デフォルト: true）
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
	AutoApproveSafe   bool `yaml:"auto_approve_safe"`   // SafetyHigh（read_file等）を確認なしで実行（デフォルトtrue）
	AutoApproveMedium bool `yaml:"auto_approve_medium"` // SafetyMedium（str_replace等）を確認なしで実行（デフォルトfalse）
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
	BracketedPaste bool `yaml:"bracketed_paste"` // Bracketed Paste Mode を有効化（デフォルト: true）
	MaxLines       int  `yaml:"max_lines"`       // 最大行数（デフォルト10000）
	MaxBytes       int  `yaml:"max_bytes"`       // 最大バイト数（デフォルト1MB）
	TimeoutSeconds int  `yaml:"timeout_seconds"` // タイムアウト秒（デフォルト60）
}

// StreamingConfig はストリーミングレスポンスの設定
type StreamingConfig struct {
	IdleTimeoutSeconds int  `yaml:"idle_timeout_seconds"` // アイドルタイムアウト秒（デフォルト30）
	ShowFileInfo       bool `yaml:"show_file_info"`       // ファイル読み込み時にサイズ表示（デフォルト: true）
	ShowSearchProgress bool `yaml:"show_search_progress"` // 検索中に進捗表示（デフォルト: true）
	StreamBashOutput   bool `yaml:"stream_bash_output"`   // bashコマンド出力をストリーミング（デフォルト: true）
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

// PlanModeConfig は Plan Mode の設定
type PlanModeConfig struct {
	// 旧設定（後方互換性のため残す）
	MaxParallelSteps int `yaml:"max_parallel_steps,omitempty"` // 非推奨: max_workers を使用
	AutoRetry        int `yaml:"auto_retry,omitempty"`         // 非推奨: max_retry を使用

	// 並列実行設定（Phase 3）
	Parallel   bool `yaml:"parallel"`    // 並列モード有効化（デフォルト: false）
	MaxWorkers int  `yaml:"max_workers"` // 並列ワーカー数（デフォルト: 3）

	// モデル設定（Phase 3）
	SupervisorModel string `yaml:"supervisor_model"` // Supervisor用モデル（空=メインモデル）
	LightModel      string `yaml:"light_model"`      // Worker用軽量モデル（空=メインモデル）
	HeavyModel      string `yaml:"heavy_model"`      // エスカレーション用モデル（空=無効）

	// リトライ・タイムアウト（Phase 3）
	MaxRetry    int `yaml:"max_retry"`    // 自動リトライ回数（デフォルト: 10）
	StepTimeout int `yaml:"step_timeout"` // ステップタイムアウト秒（デフォルト: 600）

	// 確認レベル（Phase 3）
	ConfirmLevel string `yaml:"confirm_level"` // all / dangerous / none（デフォルト: dangerous）
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

// RepoMapConfig は RepoMap の設定
type RepoMapConfig struct {
	Enabled   bool `yaml:"enabled"`              // RepoMapを有効化（デフォルト: true）
	MaxTokens int  `yaml:"max_tokens,omitempty"` // 0 = 自動計算（ファイル数ベース）
}

// WebSearchConfig はWeb検索キャッシュの設定
type WebSearchConfig struct {
	CacheEnabled bool `yaml:"cache_enabled"` // キャッシュを有効化（デフォルト: true）
	CacheTTL     int  `yaml:"cache_ttl"`     // キャッシュTTL秒数（デフォルト: 1800 = 30分）
	CacheSize    int  `yaml:"cache_size"`    // 最大キャッシュ数（デフォルト: 100）
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
func (c *Config) IsResponsesAPIModel(model string) bool {
	for _, m := range c.OpenAI.ResponsesAPIModels {
		if m == model {
			return true
		}
	}
	return false
}
