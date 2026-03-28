package tui

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	statusBarHeight  = 1
	inputHeight      = 3 // 上パディング1 + 入力行1 + 下パディング1
	chromeHeight     = statusBarHeight + inputHeight
	inputPrompt      = "› "
	inputPromptWidth = 2 // lipgloss.Width(inputPrompt) の事前計算値
)

var statusHintsNormal = []string{
	"Esc:NAV • /copy • Shift+drag",
	"Esc:NAV • /copy",
}
var statusHintsNav = []string{
	"count+j/k/h/l • w/b/e • 0/^/$ • v:visual • yy:copy • Tab:blocks • q:back",
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
	"Tab/Enter:toggle • j/k:blocks • y:copy • Esc:unfocus",
	"Tab • j/k • y • Esc",
}

const (
	visualModeOff = iota
	visualModeChar
	visualModeLine
)

// debugLog は TUI デバッグログ用のロガー（XELYON_TUI_DEBUG=1 で有効化）
var debugLog *log.Logger

func init() {
	if os.Getenv("XELYON_TUI_DEBUG") == "1" {
		f, err := os.OpenFile("/tmp/xelyon-tui.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			debugLog = log.New(f, "[TUI] ", log.Ltime|log.Lmicroseconds)
		}
	}
}

func tuiDebugf(format string, args ...any) {
	if debugLog != nil {
		debugLog.Printf(format, args...)
	}
}

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
	vp                   lightViewport // 軽量 viewport（bubbles/viewport は lipgloss が重いため自前実装）
	textInput            textinput.Model
	spinner              spinner.Model
	messages             []ChatMessage
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
}

// NewModel は TUI Model を作成する。
func NewModel(agent AgentInterface, initialContent string) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 0 // 無制限
	ti.Width = 80    // 後で WindowSize で更新

	// XELYON ブランドカラーのスピナー（processing中にステータスバーに表示）
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	initLines := []string{}
	if initialContent != "" {
		initLines = strings.Split(initialContent, "\n")
	}
	return Model{
		agent:        agent,
		textInput:    ti,
		spinner:      sp,
		rawLines:     append([]string(nil), initLines...),
		messages:     []ChatMessage{},
		focusedBlock: -1,
		visualStart:  visualPosition{line: -1, col: -1},
		statusLine:   agent.GetStatusLine(),
	}
}

// Init は bubbletea の Init を実装する。
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

// Update は bubbletea の Update を実装する。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.vp.scrollUp(3)
			m.afterViewportScroll()
		case tea.MouseButtonWheelDown:
			m.vp.scrollDown(3)
			m.afterViewportScroll()
		}
		if m.chromeDirty {
			m.chromeDirty = false
			m.rebuildChrome()
		}
		return m, nil

	case tea.KeyMsg:
		// Key handling may mark chromeDirty; keep chrome rebuild centralized in
		// Update after the updated model value has been applied.
		updated, cmd := m.handleKeyMsg(msg)
		m = updated.(Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		wasAtBottom := m.ready && m.vp.atBottom()
		widthChanged := m.width != msg.Width
		m.width = msg.Width
		m.height = msg.Height
		viewportHeight := m.height - chromeHeight
		if viewportHeight < 1 {
			viewportHeight = 1
		}
		if !m.ready {
			m.vp = lightViewport{width: m.width, height: viewportHeight}
			m.rebuildLayout()
			m.vp.setLines(m.getVisualRowContents())
			m.vp.gotoBottom()
			m.ready = true
		} else {
			m.vp.width = m.width
			m.vp.height = viewportHeight
			if widthChanged {
				m.rebuildLayout()
			}
			m.vp.setLines(m.getVisualRowContents())
			if wasAtBottom {
				m.vp.gotoBottom()
			}
		}
		m.textInput.Width = max(0, m.width-inputPromptWidth-1)
		m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
		m.chromeDirty = true

	case AppendMessageMsg:
		m.streamingActive = false
		cmds = append(cmds, m.appendMessage(msg.Message))

	case AppendToolResultMsg:
		m.streamingActive = false
		cmds = append(cmds, m.appendToolResult(msg.Tool))

	case StreamTextMsg:
		cmds = append(cmds, m.appendStreamText(msg.Text))
		if msg.Done {
			m.streamingActive = false
			m.statusLine = m.agent.GetStatusLine()
			m.chromeDirty = true
		}

	case UpdateStatusMsg:
		m.streamingActive = false
		m.statusLine = msg.Line
		m.chromeDirty = true

	case AgentDoneMsg:
		m.streamingActive = false
		m.statusLine = m.agent.GetStatusLine()
		m.chromeDirty = true

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		if m.agent.IsProcessing() {
			m.statusLine = m.agent.GetStatusLine()
		}
		m.chromeDirty = true
	}

	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.chromeDirty {
		m.chromeDirty = false
		m.rebuildChrome()
	}

	return m, tea.Batch(cmds...)
}
