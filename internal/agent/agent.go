package agent

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// 色定義
var (
	cyan   = color.New(color.FgCyan, color.Bold)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	dim    = color.New(color.Faint)
)

type agentConversationState struct {
	session         *history.Session
	storage         *history.Storage
	lastOutputs     []string
	compactedItems  []api.InputItem
	isCompactedMode bool
}

type agentRequestState struct {
	cancelFunc                           context.CancelFunc
	requestCtx                           context.Context
	requestPromptCancelCtx               context.Context
	lastCancelReason                     string
	strReplaceErrorCount                 int
	tokenLimitRetryCount                 int
	autoCompressUnknownContextWarningKey string
}

type agentWorkspaceState struct {
	changeStack          []tools.FileChange
	taskChangeOffset     int
	changeStorage        *history.ChangeStorage
	taskTestResult       *bool
	taskTestCommand      string
	taskPlanVerification []string
	pendingLSPFiles      []string
}

// Agent は CLI エージェントの高レベル owner。
// bootstrap / session_persistence / tool_observability / commands は
// 専用ファイルに分離し、ここには共有状態と実行同期の責務を残す。
type Agent struct {
	Model             string // 初期モデル（後方互換性のため保持）
	CurrentModel      string // 現在のモデル（再起動なしで切り替え可能）
	CurrentProvider   api.Provider
	ProviderName      string
	ProviderConfigKey string
	promptMgr         *PromptManager
	Runtime           *AgentRuntime
	History           []api.Message
	SystemPrompt      string
	mcpManager        *mcp.Manager
	mcpSurface        mcpToolSurfaceSelection
	Headless          bool
	lspClient         *lsp.Client       // LSPクライアント
	AutoApprove       bool              // --auto-approve フラグ
	Stats             *SessionStats     // セッション統計情報
	PlanModeEnabled   bool              // Plan Mode ON/OFF（デフォルト: false）
	ToolCache         *ToolCache        // ツール結果キャッシュ（read_file, list_dir）
	LocatorRegistry   *locator.Registry // Locator ID レジストリ（セッション内追記のみ）
	status            statusHolder

	// Conversation/Persistence mirrors
	agentConversationState
	// Request lifecycle state
	agentRequestState
	// Workspace mutation state
	agentWorkspaceState
	// Project prompt generation state
	agentProjectPromptState

	// exitHook は os.Exit 前に呼ばれるフック（TUI モードのターミナル復旧等）
	exitHook func()
	// signalCleanup は legacy REPL の signal.Notify goroutine を停止する。
	signalCleanup func()

	// tuiToolResultCh は TUI モードでツール実行結果を構造化データとして送信するチャネル。
	// nil の場合は従来の stdout 出力を使用する。
	tuiToolResultCh     chan tools.ToolResultInfo
	tuiToolResultClosed atomic.Bool // TUI 終了後の send panic / deadlock 防止

	// 並列実行用ミューテックス
	historyMu     sync.Mutex
	changeStackMu sync.Mutex
	statsMu       sync.Mutex

	// per-turn observability for conservatively detecting serial single-pattern
	// search_code calls that could have been grouped into one multi-pattern call.
	searchCodeRecentSinglePatternByFamily map[string]string
	searchCodeMissedMultiCountedFamilies  map[string]struct{}
}

func (a *Agent) setPromptReader(reader *ui.MultilineReader) {
	if a == nil {
		return
	}
	a.ui().SetPromptReader(reader)
}

func (a *Agent) output() io.Writer {
	if a == nil {
		return ui.DefaultRuntime().Output()
	}
	return a.ui().Output()
}

func (a *Agent) errorOutput() io.Writer {
	if a == nil {
		return ui.DefaultRuntime().ErrorOutput()
	}
	return a.ui().ErrorOutput()
}

// GetLSPClient はLSPクライアントを返す（コマンド用）
func (a *Agent) GetLSPClient() *lsp.Client {
	return a.lspClient
}

// appendHistory は History へスレッドセーフに追加（並列実行時用）
func (a *Agent) appendHistory(msg api.Message) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	a.History = append(a.History, msg)
}

// getHistorySnapshot は History のスナップショットをスレッドセーフに取得（並列実行時用）
func (a *Agent) getHistorySnapshot() []api.Message {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	snapshot := make([]api.Message, len(a.History))
	copy(snapshot, a.History)
	return snapshot
}

// appendChange は changeStack へスレッドセーフに追加（並列実行時用）
func (a *Agent) appendChange(change tools.FileChange) {
	a.changeStackMu.Lock()
	defer a.changeStackMu.Unlock()
	a.changeStack = append(a.changeStack, change)
	if len(a.changeStack) > config.MaxChangeStack {
		a.changeStack = a.changeStack[1:]
	}
}

// incrementToolExecution は Stats をスレッドセーフに更新（並列実行時用）
func (a *Agent) incrementToolExecution(toolName string) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.AddToolExecution(toolName)
	}
}

// incrementAssistantMessages は Stats をスレッドセーフに更新（並列実行時用）
func (a *Agent) incrementAssistantMessages() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}
}

func (a *Agent) addOptimizationMetrics(metrics OptimizationMetrics) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.Optimizations.add(metrics)
	}
}

func (a *Agent) addCompactionMetrics(metrics CompactionMetrics) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.Optimizations.addCompaction(metrics)
	}
}
