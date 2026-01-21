package config

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
	PlanMode        PlanModeConfig                 `yaml:"plan_mode,omitempty"`
	LSP             LSPConfig                      `yaml:"lsp,omitempty"`
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

// PlanModeConfig は Plan Mode の設定
type PlanModeConfig struct {
	MaxParallelSteps int `yaml:"max_parallel_steps"` // 並列実行数（デフォルト: 3）
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
