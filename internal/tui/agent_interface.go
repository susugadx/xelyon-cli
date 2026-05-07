package tui

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
)

// ConversationAgent は chat 実行と処理状態を TUI に提供する。
type ConversationAgent interface {
	// Chat はユーザー入力をAIに送信する（非同期、goroutineで呼ぶ）
	Chat(input string)

	// GetStatusLine はステータスバーに表示する文字列を返す。
	GetStatusLine() string

	// Cancel は現在のAPI呼び出しをキャンセルする。
	Cancel()

	// Cleanup は終了処理を行う。
	Cleanup()

	// IsProcessing はAI処理中（chat実行中）かどうかを返す。
	IsProcessing() bool
}

// CommandAgent は slash command の実行と alias 解決を TUI に提供する。
type CommandAgent interface {
	// HandleCommand は /status, /use, /clear 等のコマンドを処理する。
	// 処理した場合 true を返す。
	HandleCommand(cmd string) bool

	// ResolveAlias はコマンド名を alias 解決する（例: "/c" -> "/config"）。
	ResolveAlias(cmd string) string
}

// ClipboardAgent は TUI から利用する clipboard 操作を提供する。
type ClipboardAgent interface {
	// CopyLastOutput は直近のAI出力をクリップボードにコピーする。
	// 成功時は要約（例: "Copied 12 lines"）、失敗時はエラーを返す。
	CopyLastOutput() (string, error)

	// CopyText は指定テキストをクリップボードにコピーする。
	CopyText(text string) error
}

// ConfigAgent は /config 画面が必要とする設定の load/save と provider 情報を提供する。
type ConfigAgent interface {
	// LoadConfigForEdit は設定ファイルを読み込み、編集用のクローンを返す。
	LoadConfigForEdit() (*config.Config, error)

	// SaveAndSyncConfig は設定をファイルに保存し、runtime に反映する。
	SaveAndSyncConfig(cfg *config.Config) error

	// GetProviderName は現在のプロバイダー名を返す。
	GetProviderName() string

	// GetProviderConfigKey は現在セッションが代表する provider_models key を返す。
	GetProviderConfigKey() string
}

// ProjectAgent は /project 画面が必要とする xelyon.yaml の load/save を提供する。
type ProjectAgent interface {
	// LoadProjectForEdit は xelyon.yaml を読み込み、編集用のコピーを返す。
	// xelyon.yaml が見つからない場合は nil, nil を返す。
	LoadProjectForEdit() (*config.ProjectConfig, error)

	// SaveProjectConfig は xelyon.yaml に project config を保存する。
	SaveProjectConfig(pc *config.ProjectConfig) error

	// CreateProjectConfigTemplate は xelyon.yaml のテンプレートを作成して読み込む。
	CreateProjectConfigTemplate() (*config.ProjectConfig, error)
}

// ReviewAgent は /review 実行を TUI の外側へ渡す optional capability を表す。
type ReviewAgent interface {
	RunReview(context.Context, review.ReviewRequest) (review.ReviewReport, error)
}

// AgentInterface は tui パッケージから agent パッケージへの依存を逆転させる。
// agent パッケージが各 capability interface を実装する。
type AgentInterface interface {
	ConversationAgent
	CommandAgent
	ClipboardAgent
	ConfigAgent
	ProjectAgent
}
