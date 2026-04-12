package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	screenChat   screenMode = iota // 通常のチャット画面
	screenConfig                   // /config 設定画面
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

// toolBlockInfo は表示上のツール結果ブロックを追跡する。
type toolBlockInfo struct {
	lineStart int        // rawLines でのブロック開始行インデックス
	lineCount int        // ブロックが占める行数
	tool      ToolResult // ツール結果データ
}

type visualPosition struct {
	line int
	col  int
}

// Model は bubbletea の Model インターフェースを実装する TUI のメインモデル。
type Model struct {
	agent                AgentInterface
	screen               screenMode    // 現在の画面モード
	configScreen         *configScreen // /config 画面の状態（screenConfig 時のみ非 nil）
	vp                   lightViewport // 軽量 viewport（bubbles/viewport は lipgloss が重いため自前実装）
	textInput            textinput.Model
	spinner              spinner.Model
	messages             []ChatMessage
	composerParts        []composerPart
	pasteBlocks          []pasteBlock
	nextPasteUID         int
	rawLines             []string        // 元の行データ。リサイズ時はこれを再レンダリングする
	layout               *Layout         // 表示幅に応じたvisual rowレイアウト
	toolBlocks           []toolBlockInfo // ツール結果ブロック
	focusedBlock         int             // NAVモードでフォーカス中のツールブロックインデックス（-1=なし）
	statusLine           string
	padLineCache         string // View() 用の背景パディング行キャッシュ
	chromeCache          string // View() 用の chrome 部分キャッシュ（入力欄+ステータス）
	chromeDirty          bool   // chrome 再構築が必要か
	width                int
	height               int
	newOutput            bool // 上スクロール中に新出力があったか
	ready                bool // viewport 初期化済みか
	quitting             bool
	navigationMode       bool // Vim ナビゲーションモード
	gPressed             bool // g キーが1回押された状態
	pendingCount         int  // 数字プレフィックスで入力中の移動回数
	yPressed             bool // y キーが1回押された状態（yy用）
	cursorLine           int  // NAVモードの現在行（rawLines基準）
	cursorCol            int  // NAVモードの現在列（表示幅基準）
	visualMode           int  // 0=OFF, 1=文字単位, 2=行単位
	visualStart          visualPosition
	transientStatus      string    // 一時通知メッセージ
	transientStatusUntil time.Time // 一時通知の有効期限
	lastInterrupt        time.Time
	streamingActive      bool
	streamCursorCol      int
	streamActiveANSI     string
	streamPendingANSI    string
	mouseSelAnchor       visualPosition // マウスドラッグ選択の開始位置（line=-1=選択なし）
	mouseSelEnd          visualPosition // マウスドラッグ選択の終了位置
	mouseDragging        bool           // マウスドラッグ中か
	mouseAutoScrolling   bool           // 自動スクロールティック発行済みか
	mouseLastScreenX     int            // 最後のマウスX座標（自動スクロール用）
	mouseLastScreenY     int            // 最後のマウスY座標（自動スクロール用）
}
