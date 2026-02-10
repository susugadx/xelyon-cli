package agent

import (
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// WorkerResult は Worker → Supervisor への結果報告
type WorkerResult struct {
	WorkerID      int                // Worker ID
	StepID        int                // ステップ ID
	Success       bool               // 成功フラグ
	SkipRetry     bool               // true の場合リトライせず即ユーザー確認
	Output        string             // 出力
	Changes       []tools.FileChange // ファイル変更
	Error         error              // エラー
	ToolsExecuted []string           // 実行したツール名
	TokensUsed    int                // 使用トークン数
	Duration      time.Duration      // 実行時間
	Query         string             // 調査クエリ（調査フェーズ用）
}

// WorkerCommand は Supervisor → Worker へのコマンド
type WorkerCommand struct {
	Type         CommandType    // コマンドタイプ
	StepID       int            // ステップ ID
	Step         *plan.PlanStep // 実行するステップ
	Query        string         // 調査クエリ（調査フェーズ用）
	ConfirmLevel string         // 確認レベル
}

// CommandType はコマンドの種類
type CommandType int

const (
	CmdExecuteStep CommandType = iota // ステップ実行
	CmdInvestigate                    // 調査実行
	CmdStop                           // 停止
)

// WorkerStatus はWorkerの状態
type WorkerStatus string

const (
	WorkerIdle    WorkerStatus = "idle"    // 待機中
	WorkerRunning WorkerStatus = "running" // 実行中
	WorkerDone    WorkerStatus = "done"    // 完了
	WorkerFailed  WorkerStatus = "failed"  // 失敗
)

// UI表示用のステータスアイコン
const (
	IconWaiting   = "⏸️" // 待機中（依存解決待ち）
	IconRunning   = "⏳"  // 実行中
	IconCompleted = "✅"  // 完了
	IconFailed    = "❌"  // 失敗
	IconRetrying  = "🔄"  // リトライ中
)

// StepFailure はステップ失敗のエラー
type StepFailure struct {
	StepID int
	Reason string
}

func (e *StepFailure) Error() string {
	return fmt.Sprintf("step %d failed: %s", e.StepID, e.Reason)
}

// StepTimeout はステップタイムアウトのエラー
type StepTimeout struct {
	StepID  int
	Timeout int // 秒
}

func (e *StepTimeout) Error() string {
	return fmt.Sprintf("step %d timed out after %d seconds", e.StepID, e.Timeout)
}

// InvestigationQuery は調査クエリ
type InvestigationQuery struct {
	Query    string // 調査クエリ
	WorkerID int    // 割り当てられた Worker ID
}

// InvestigationResult は調査結果
type InvestigationResult struct {
	Query    string        // 調査クエリ
	Result   string        // 結果
	Files    []string      // 発見したファイル
	Duration time.Duration // 実行時間
	WorkerID int           // 実行した Worker ID
}
