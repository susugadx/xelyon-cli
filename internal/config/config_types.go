package config

import "github.com/susugadx/xelyon-cli/internal/llmcatalog"

type providerModelSectionState int

const (
	providerModelSectionStateAbsent providerModelSectionState = iota
	providerModelSectionStateExplicitEmpty
	providerModelSectionStateExplicitEntries
	providerModelSectionStateExplicitEntriesPreserveEmpty
	providerModelSectionStateImplicitEntries
	providerModelSectionStateInMemoryEffectiveOnly
)

type providerModelStore struct {
	state providerModelSectionState
	raw   map[string]ProviderModelConfig
}

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

	GitStage            GitStageConfig     `yaml:"git_stage"`
	LSP                 LSPConfig          `yaml:"lsp"`
	OpenAI              OpenAIConfig       `yaml:"openai"`
	Thinking            ThinkingConfig     `yaml:"thinking"`
	Output              OutputConfig       `yaml:"output"`
	WebSearch           WebSearchConfig    `yaml:"web_search"`
	SubAgent            SubAgentConfig     `yaml:"sub_agent"`
	MCP                 MCPConfig          `yaml:"mcp"`
	FinalChecks         FinalChecksConfig  `yaml:"final_checks"`
	SubAgentPrompt      string             `yaml:"-"`
	providerModelsStore providerModelStore `yaml:"-"`
	// 将来の拡張用
	// Cloud CloudConfig `yaml:"cloud,omitempty"`
}

func (c *Config) providerModelSectionState() providerModelSectionState {
	if c == nil {
		return providerModelSectionStateAbsent
	}
	return c.providerModelsStore.state
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
//
// user-facing 設定: BracketedPaste のみ
// MaxLines, MaxBytes は内部既定値（YAML 直接編集で変更可能）
type PasteConfig struct {
	BracketedPaste bool `yaml:"bracketed_paste"` // Bracketed Paste Mode を有効化（デフォルト: true）
	MaxLines       int  `yaml:"max_lines"`       // 内部: 最大行数（デフォルト10000）
	MaxBytes       int  `yaml:"max_bytes"`       // 内部: 最大バイト数（デフォルト1MB）
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

// ListDirConfig は list_dir ツールの設定（user-facing config から削除済み、YAML互換のみ保持）
// 将来的には shared ignore の正式な置き場（project_map.additional_ignore_dirs 等）へ統合予定
type ListDirConfig struct {
	AdditionalIgnoreDirs []string `yaml:"additional_ignore_dirs"` // 内部: デフォルト除外に追加するディレクトリ名
}

// ProjectMapConfig はプロジェクト構造マップの設定
type ProjectMapConfig struct {
	Enabled              bool     `yaml:"enabled"`                // 起動時に Project Map を生成・注入
	ContextRatio         float64  `yaml:"context_ratio"`          // ProjectMap のベース比率（大規模 repo では自動的に引き上げ）
	AdditionalIgnoreDirs []string `yaml:"additional_ignore_dirs"` // デフォルト除外に追加するディレクトリ名
}

// GitStageConfig はgit_addツールの設定（user-facing config から削除済み、YAML互換のみ保持）
type GitStageConfig struct {
	BatchConfirm bool `yaml:"batch_confirm"` // 標準挙動としてハードコード（設定不要）
}

// OpenAIConfig は OpenAI プロバイダーの内部設定（user-facing config から削除済み、YAML 互換は維持）
// Responses API ルーティングはプレフィックスマッチで自動判定し、このリストはフォールバック用
type OpenAIConfig struct {
	ResponsesAPIModels []string `yaml:"responses_api_models"` // 内部: Responses API フォールバックリスト
}

// ThinkingConfig は Extended Thinking の内部設定（user-facing config から削除済み、YAML 互換は維持）
// 正規の切り替えルートは /think コマンド。config のデフォルト値は runtime 初期値として使用
type ThinkingConfig struct {
	Enabled bool   `yaml:"enabled"` // 内部: デフォルト false（/think コマンドで変更）
	Level   string `yaml:"level"`   // 内部: low/medium/high/xhigh（デフォルト: medium、/think コマンドで変更）
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

// MCPConfig は MCP (Model Context Protocol) サーバー接続の設定
type MCPConfig struct {
	Enabled  bool `yaml:"enabled"`  // MCP接続を有効化（デフォルト: true）
	Headless bool `yaml:"headless"` // Headlessモードでも接続（デフォルト: false）
}

// FinalChecksConfig は明示完了時に実行する user-configured final checks 設定。
type FinalChecksConfig struct {
	Commands []string `yaml:"commands"` // completed_with_changes 時に実行するコマンド
	Timeout  int      `yaml:"timeout"`  // コマンドタイムアウト秒（デフォルト: 600）
}

// VerificationConfig は旧名互換の型エイリアス。
type VerificationConfig = FinalChecksConfig

// LSPConfig は LSP (Language Server Protocol) 連携の設定
type LSPConfig struct {
	Enabled           bool                       `yaml:"enabled"`                       // LSP機能を有効化（デフォルト: true）
	SkipInstallPrompt bool                       `yaml:"skip_install_prompt,omitempty"` // インストール提案をスキップ
	Servers           map[string]LSPServerConfig `yaml:"servers"`
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

// IsResponsesAPIModel はモデルが OpenAI Responses API を使用するか判定
// 対応モデルは prefix マッチで自動判定し、設定リストをフォールバックとして使用
func (c *Config) IsResponsesAPIModel(model string) bool {
	return llmcatalog.IsOpenAIResponsesModel(model, c.OpenAI.ResponsesAPIModels)
}
