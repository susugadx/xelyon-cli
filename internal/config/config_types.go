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
	ToolConfirm     ToolConfirmConfig              `yaml:"tool_confirm"`
	CommandAliases  map[string]string              `yaml:"command_aliases,omitempty"` // コマンドエイリアス
	PromptCache     PromptCacheConfig              `yaml:"prompt_cache"`
	Paste           PasteConfig                    `yaml:"paste"`
	Streaming       StreamingConfig                `yaml:"streaming"`
	Bash            BashConfig                     `yaml:"bash"`
	ListDir         ListDirConfig                  `yaml:"list_dir"`
	ProjectMap      ProjectMapConfig               `yaml:"project_map"`

	GitStage  GitStageConfig  `yaml:"git_stage"`
	PlanMode  PlanModeConfig  `yaml:"plan_mode"`
	LSP       LSPConfig       `yaml:"lsp"`
	OpenAI    OpenAIConfig    `yaml:"openai"`
	Thinking  ThinkingConfig  `yaml:"thinking"`
	Output    OutputConfig    `yaml:"output"`
	WebSearch WebSearchConfig `yaml:"web_search"`
	MCP       MCPConfig       `yaml:"mcp"`
	Hooks     HooksConfig     `yaml:"hooks"`
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
	AutoCompress      bool   `yaml:"auto_compress"`                          // 自動圧縮を有効化（デフォルト: true）
	ThresholdTokens   int    `yaml:"threshold_tokens"`                       // 自動圧縮のトークン閾値（0 = 使用率ベース）
	ThresholdPercent  int    `yaml:"threshold_percent"`                      // 自動圧縮の使用率閾値（デフォルト: 80%）
	TokenThreshold    int    `yaml:"token_threshold" json:"token_threshold"` // Context トークン数の絶対閾値（デフォルト: 100000）
	Model             string `yaml:"model" json:"model"`                     // 圧縮用モデル名（空 = プロバイダー別デフォルト、main = メインモデル）
	KeepRecent        int    `yaml:"keep_recent"`                            // 保持する最新メッセージ数
	PreferCompactAPI  bool   `yaml:"prefer_compact_api"`                     // OpenAI Compact API を優先（デフォルト: true）
	ClaudeCompaction  bool   `yaml:"claude_compaction"`                      // Claude Compaction API 有効化
	CompactionTrigger int    `yaml:"compaction_trigger"`                     // トリガー閾値（デフォルト 150000）
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
	AllowPipe       bool     `yaml:"allow_pipe"`        // パイプを許可（デフォルト: true - moderateで有効）
	AllowRedirect   bool     `yaml:"allow_redirect"`    // リダイレクトを許可（デフォルト: false）
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
	MaxRetry    int `yaml:"max_retry"`    // 自動リトライ回数（デフォルト: 10）
	StepTimeout int `yaml:"step_timeout"` // ステップタイムアウト秒（デフォルト: 600）
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
