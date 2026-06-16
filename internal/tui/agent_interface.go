package tui

import (
	"context"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/review"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

// ConversationAgent は chat 実行と処理状態を TUI に提供する。
type ConversationAgent interface {
	// Chat はユーザー入力をAIに送信する（非同期、goroutineで呼ぶ）。
	// 送信ターンが失敗または中断された場合は error を返す。
	Chat(input string) error

	// ChatWithImagePath は画像ファイル付きでユーザー入力をAIに送信する（非同期、goroutineで呼ぶ）。
	// 画像の読み込み/検証は実装側で行う。
	ChatWithImagePath(input string, imagePath string) error

	// GetStatusLine はステータスバーに表示する文字列を返す。
	GetStatusLine() string

	// StatusSnapshot は TUI ステータスバー用の構造化状態を返す。
	StatusSnapshot() StatusSnapshot

	// Cancel は現在のAPI呼び出しをキャンセルする。
	Cancel()

	// Cleanup は終了処理を行う。
	Cleanup()

	// IsProcessing はAI処理中（chat実行中）かどうかを返す。
	IsProcessing() bool
}

// CommandAgent は slash command の実行を TUI に提供する。
type CommandAgent interface {
	// HandleCommand は /status, /use, /clear 等のコマンドを処理する。
	// 処理した場合 true を返す。
	HandleCommand(cmd string) bool
}

// SessionCandidate は TUI の session picker に表示する再開候補を表す。
type SessionCandidate struct {
	ID           string
	Preview      string
	ProviderName string
	Model        string
	WorkingDir   string
	LastModified time.Time
	MessageCount int
}

// SessionResumeOptions は session picker/resume の候補範囲を表す。
type SessionResumeOptions struct {
	All bool
}

// SessionAgent は session の新規作成と再開を TUI に提供する。
type SessionAgent interface {
	ResumeSessionCandidates(SessionResumeOptions) ([]SessionCandidate, error)
	ResumeLastSession(SessionResumeOptions) (SessionCandidate, error)
	ResumeSession(id string) error
	ResumeStartupSession(id string) error
	StartNewSession() (string, error)
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

// ProviderModelAgent は /provider と /model picker が必要とする候補取得と切替を提供する。
type ProviderModelAgent interface {
	// ProviderCandidates は provider picker の候補を返す。
	ProviderCandidates() []providerpicker.ProviderCandidate

	// ModelCandidates は provider に対応する model/deployment picker 候補を返す。
	ModelCandidates(provider string) []providerpicker.ModelCandidate

	// AzureCatalogModelCandidates は Azure deployment に紐づける catalog_model 候補を返す。
	AzureCatalogModelCandidates(deployment string) []providerpicker.ModelCandidate

	// SwitchProviderModel は provider と model/deployment を現在セッションに適用する。
	SwitchProviderModel(provider string, model string) error

	// SwitchModelForCurrentProvider は current provider 内で model/deployment を切り替える。
	SwitchModelForCurrentProvider(model string) error

	// ConfigureAndSwitchAzureDeployment は Azure deployment setup を保存して provider を切り替える。
	ConfigureAndSwitchAzureDeployment(deployment string, catalogModel string) error
}

// ProjectAgent は /project 画面が必要とする xelyon.yaml の load/save を提供する。
type ProjectAgent interface {
	// LoadProjectForEdit は xelyon.yaml を読み込み、編集用のコピーを返す。
	// xelyon.yaml が見つからない場合は nil, nil を返す。
	LoadProjectForEdit() (*config.ProjectConfig, error)

	// SaveProjectConfig は xelyon.yaml に project config を保存する。
	SaveProjectConfig(pc *config.ProjectConfig) error

	// CreateProjectConfigTemplate は xelyon.yaml の repo config template を作成して読み込む。
	CreateProjectConfigTemplate() (*config.ProjectConfig, error)
}

// ReviewAgent は /review 実行を TUI の外側へ渡す optional capability を表す。
type ReviewAgent interface {
	RunReview(context.Context, review.ReviewRequest) (ReviewRunResult, error)
}

// SkillCatalogAgent は /skills 補完が利用する optional capability を表す。
type SkillCatalogAgent interface {
	SkillCatalog() agentskills.SkillCatalog
}

// AgentInterface は tui パッケージから agent パッケージへの依存を逆転させる。
// agent パッケージが各 capability interface を実装する。
type AgentInterface interface {
	ConversationAgent
	CommandAgent
	ClipboardAgent
	ConfigAgent
	ProviderModelAgent
	SessionAgent
	ProjectAgent
}
