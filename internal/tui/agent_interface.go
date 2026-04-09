package tui

import "github.com/susugadx/xelyon-cli/internal/config"

// AgentInterface は tui パッケージから agent パッケージへの依存を逆転させる。
// agent パッケージがこのインターフェースを実装する。
type AgentInterface interface {
	// Chat はユーザー入力をAIに送信する（非同期、goroutineで呼ぶ）
	Chat(input string)

	// HandleCommand は /status, /use, /clear 等のコマンドを処理する。
	// 処理した場合 true を返す。
	HandleCommand(cmd string) bool

	// GetStatusLine はステータスバーに表示する文字列を返す。
	GetStatusLine() string

	// Cancel は現在のAPI呼び出しをキャンセルする。
	Cancel()

	// Cleanup は終了処理を行う。
	Cleanup()

	// IsProcessing はAI処理中（chat実行中）かどうかを返す。
	IsProcessing() bool

	// CopyLastOutput は直近のAI出力をクリップボードにコピーする。
	// 成功時は要約（例: "Copied 12 lines"）、失敗時はエラーを返す。
	CopyLastOutput() (string, error)

	// CopyText は指定テキストをクリップボードにコピーする。
	CopyText(text string) error

	// LoadConfigForEdit は設定ファイルを読み込み、編集用のクローンを返す。
	LoadConfigForEdit() (*config.Config, error)

	// SaveAndSyncConfig は設定をファイルに保存し、runtime に反映する。
	SaveAndSyncConfig(cfg *config.Config) error

	// GetProviderName は現在のプロバイダー名を返す。
	GetProviderName() string

	// GetProviderConfigKey は現在セッションが代表する provider_models key を返す。
	GetProviderConfigKey() string

	// ResolveAlias はコマンド名を alias 解決する（例: "/c" → "/config"）。
	ResolveAlias(cmd string) string
}
