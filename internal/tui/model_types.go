package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tuicomposer "github.com/susugadx/xelyon-cli/internal/tui/composer"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

const (
	statusBarHeight  = 1
	inputHeight      = 3 // 上パディング1 + 入力行1 + 下パディング1
	inputPrompt      = "› "
	inputPromptWidth = 2 // lipgloss.Width(inputPrompt) の事前計算値
)

// screenMode は TUI の画面モードを表す。
type screenMode int

const (
	screenChat    screenMode = iota // 通常のチャット画面
	screenConfig                    // /config 設定画面
	screenReview                    // /review preset 画面
	screenProject                   // /project 設定画面
)

var statusHintsNormal = []string{
	"Esc:NAV • /copy • drag:select",
	"Esc:NAV • /copy",
}

var statusHintsNav = []string{
	"count+j/k/h/l • w/b/e • 0/^/$ • v:visual • yy:copy • Tab/Shift+Tab:blocks • q:back",
	"count+j/k • w/b/e • 0/$ • v • yy",
}

var statusHintsVisual = []string{
	"-- VISUAL -- count+j/k/h/l • w/b/e • 0/^/$ • y:copy • Esc:cancel",
	"-- VISUAL -- w/b/e • 0/$ • y • Esc",
}

var statusHintsVisualLine = []string{
	"-- VISUAL LINE -- j/k:select • y:copy • Esc:cancel",
	"-- VISUAL LINE -- y • Esc",
}

var statusHintsBlockFocus = []string{
	"Tab/Shift+Tab/j/k/↑/↓:move • Enter:toggle • y:copy • Esc:unfocus",
	"Tab/Shift+Tab • j/k/↑/↓ • Enter • y • Esc",
}

var statusHintsMouseSel = []string{
	"-- SELECT -- Ctrl+C:copy • /copy • Esc:clear",
	"-- SELECT -- Ctrl+C • Esc",
}

const (
	visualModeOff = iota
	visualModeChar
	visualModeLine
)

// trackedBlock は rawLines 上で更新可能な transcript block の範囲を追跡する。
type trackedBlock struct {
	lineStart int // rawLines でのブロック開始行インデックス
	lineCount int // ブロックが占める行数
}

// toolBlockInfo は表示上のツール結果ブロックを追跡する。
type toolBlockInfo struct {
	block trackedBlock
	tool  ToolResult // ツール結果データ
}

type agentActivityStatus string

const (
	agentActivityStatusWorking agentActivityStatus = "working"
	agentActivityStatusDone    agentActivityStatus = "done"
	agentActivityStatusBlocked agentActivityStatus = "blocked"
)

// agentActivityState は進行中 turn の agent activity block を追跡する。
type agentActivityState struct {
	active     bool
	block      trackedBlock
	startedAt  time.Time
	finishedAt time.Time
	status     agentActivityStatus
	tools      []agentActivityTool
	errorText  string
	errorKind  AgentErrorKind
}

type agentActivityTool struct {
	tool ToolResult
}

type visualPosition struct {
	line int
	col  int
}

// Model は bubbletea の Model インターフェースを実装する TUI のメインモデル。
type Model struct {
	conversation     ConversationAgent
	commands         CommandAgent
	clipboard        ClipboardAgent
	configAgent      ConfigAgent
	providerModels   ProviderModelAgent
	projectAgent     ProjectAgent
	screen           screenMode     // 現在の画面モード
	configScreen     *configScreen  // /config 画面の状態（screenConfig 時のみ非 nil）
	reviewScreen     *reviewScreen  // /review 画面の状態（screenReview 時のみ非 nil）
	projectScreen    *projectScreen // /project 画面の状態（screenProject 時のみ非 nil）
	projectScreenSeq int            // /project 非同期メッセージの画面識別子
	vp               lightViewport  // 軽量 viewport（bubbles/viewport は lipgloss が重いため自前実装）
	textInput        textinput.Model
	spinner          spinner.Model
	messages         []ChatMessage
	composer         tuicomposer.State
	attachments      []composerAttachment
	rawLines         []string         // 元の行データ。リサイズ時はこれを再レンダリングする
	layout           *termtext.Layout // 表示幅に応じたvisual rowレイアウト
	toolBlocks       []toolBlockInfo  // ツール結果ブロック
	agentActivity    agentActivityState
	focusedBlock     int // NAVモードでフォーカス中のツールブロックインデックス（-1=なし）
	statusLine       string
	statusSnapshot   StatusSnapshot
	workingDir       string
	padLineCache     string // View() 用の背景パディング行キャッシュ
	chromeCache      string // View() 用の chrome 部分キャッシュ（入力欄+ステータス）
	chromeDirty      bool   // chrome 再構築が必要か
	width            int
	height           int
	newOutput        bool // 上スクロール中に新出力があったか
	ready            bool // viewport 初期化済みか
	quitting         bool
	navigationState
	transientStatus      string    // 一時通知メッセージ
	transientStatusUntil time.Time // 一時通知の有効期限
	lastInterrupt        time.Time
	streamState
	mouseSelectionState
	startupSubmission *StartupSubmission
	slashSuggestions  slashSuggestionState
	prompt            *promptModalState
	providerPicker    *providerPickerState
}
