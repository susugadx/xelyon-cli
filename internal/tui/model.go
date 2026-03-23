package tui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	statusBarHeight = 1
	inputHeight     = 1
	chromeHeight    = statusBarHeight + inputHeight
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

// Model は bubbletea の Model インターフェースを実装する TUI のメインモデル。
type Model struct {
	agent         AgentInterface
	viewport      viewport.Model
	textInput     textinput.Model
	messages      []ChatMessage
	content       *strings.Builder // viewport に表示するテキスト全体（ポインタ: bubbletea の値コピーで panic 回避）
	statusLine    string
	width         int
	height        int
	newOutput     bool // 上スクロール中に新出力があったか
	ready         bool // viewport 初期化済みか
	quitting      bool
	lastInterrupt time.Time
}

// NewModel は TUI Model を作成する。
func NewModel(agent AgentInterface, initialContent string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 0 // 無制限
	ti.Width = 80    // 後で WindowSize で更新

	m := Model{
		agent:      agent,
		textInput:  ti,
		content:    &strings.Builder{},
		messages:   []ChatMessage{},
		statusLine: agent.GetStatusLine(),
	}
	if initialContent != "" {
		m.content.WriteString(initialContent)
	}
	return m
}

// enableMouseWheel は mode 1000 + SGR 1006 を手動で有効化する tea.Cmd。
// bubbletea の WithMouseAllMotion/WithMouseCellMotion は使わず、
// ホイールイベントのみキャプチャしてネイティブテキスト選択を維持する。
func enableMouseWheel() tea.Msg {
	// stdout に直接書き込み（bubbletea のレンダラーを経由しない）
	_, _ = fmt.Fprint(os.Stdout, enableMouseWheelSeq)
	return nil
}

// Init は bubbletea の Init を実装する。
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.tickStatus(),
		enableMouseWheel,
	)
}

// tickStatusMsg はステータスバー定期更新用の Msg
type tickStatusMsg time.Time

func (m Model) tickStatus() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickStatusMsg(t)
	})
}

// Update は bubbletea の Update を実装する。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		viewportHeight := m.height - chromeHeight
		if viewportHeight < 1 {
			viewportHeight = 1
		}
		if !m.ready {
			m.viewport = viewport.New(m.width, viewportHeight)
			m.viewport.SetContent(m.content.String())
			m.viewport.GotoBottom()
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = viewportHeight
		}
		m.textInput.Width = m.width - 4 // "> " prefix + padding

	case tea.MouseMsg:
		// mode 1000 はクリック/リリース/ホイールを全て送信するが、
		// ホイールのみ viewport に渡す。クリック/リリースは無視して
		// ネイティブテキスト選択（Shift+ドラッグ）との干渉を最小化。
		if tea.MouseEvent(msg).IsWheel() {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case AppendMessageMsg:
		m.appendMessage(msg.Message)

	case StreamTextMsg:
		m.appendStreamText(msg.Text)
		if msg.Done {
			m.statusLine = m.agent.GetStatusLine()
		}

	case UpdateStatusMsg:
		m.statusLine = msg.Line

	case AgentDoneMsg:
		m.statusLine = m.agent.GetStatusLine()

	case tickStatusMsg:
		if m.agent.IsProcessing() {
			m.statusLine = m.agent.GetStatusLine()
		}
		cmds = append(cmds, m.tickStatus())
	}

	// viewport の更新
	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	// textInput の更新（KeyMsg 以外）
	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// isEnterKey は Enter キーかどうかを判定する。
// WSL2/Windows Terminal 環境で tea.KeyEnter が正しく認識されない場合の回避策を含む。
func isEnterKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyEnter {
		return true
	}
	// フォールバック: 文字列比較
	s := msg.String()
	return s == "enter" || s == "\r" || s == "\n"
}

// handleKeyMsg はキー入力を処理する。
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tuiDebugf("KeyMsg: Type=%d(%s) Runes=%v String=%q", msg.Type, msg.Type, msg.Runes, msg.String())

	switch {
	case msg.Type == tea.KeyCtrlC:
		if m.agent.IsProcessing() {
			m.agent.Cancel()
			m.appendSystemInfo("⚠️  Interrupted. Press Ctrl+C again to exit.")
			return m, nil
		}
		now := time.Now()
		if !m.lastInterrupt.IsZero() && now.Sub(m.lastInterrupt) < 3*time.Second {
			m.quitting = true
			m.agent.Cleanup()
			return m, tea.Quit
		}
		m.lastInterrupt = now
		m.appendSystemInfo("⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")
		return m, nil

	case isEnterKey(msg):
		tuiDebugf("Enter detected, textInput value=%q", m.textInput.Value())
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return m, nil
		}
		m.textInput.Reset()

		// ユーザー入力を会話ログに追加
		m.appendMessage(ChatMessage{
			Role:      "user",
			Content:   input,
			Timestamp: time.Now(),
		})

		// コマンドチェック
		if strings.HasPrefix(input, "/") {
			if input == "/exit" || input == "/quit" {
				m.quitting = true
				m.agent.Cleanup()
				return m, tea.Quit
			}
			if m.agent.HandleCommand(input) {
				m.statusLine = m.agent.GetStatusLine()
				return m, nil
			}
		}

		// AIに送信（goroutine）
		return m, m.sendChat(input)

	case msg.Type == tea.KeyUp, msg.Type == tea.KeyDown,
		msg.Type == tea.KeyPgUp, msg.Type == tea.KeyPgDown:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	// その他のキーは textInput に渡す
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// sendChat は goroutine で agent.Chat を呼び出す tea.Cmd を返す。
func (m Model) sendChat(input string) tea.Cmd {
	return func() tea.Msg {
		m.agent.Chat(input)
		return AgentDoneMsg{}
	}
}

// appendMessage は会話ログにメッセージを追加する。
func (m *Model) appendMessage(msg ChatMessage) {
	m.messages = append(m.messages, msg)

	// viewport の content に追加
	var line string
	switch msg.Role {
	case "user":
		line = fmt.Sprintf("\n> %s\n", msg.Content)
	case "assistant":
		line = msg.Content + "\n"
	case "system_info":
		line = msg.Content + "\n"
	default:
		line = msg.Content + "\n"
	}

	m.content.WriteString(line)
	m.updateViewport()
}

// appendStreamText はストリーミングテキストを追加する。
func (m *Model) appendStreamText(text string) {
	m.content.WriteString(text)
	m.updateViewport()
}

// appendSystemInfo はシステム情報メッセージを追加する。
func (m *Model) appendSystemInfo(text string) {
	m.appendMessage(ChatMessage{
		Role:      "system_info",
		Content:   text,
		Timestamp: time.Now(),
	})
}

// updateViewport は viewport の内容を更新し、必要に応じて最下部に追従する。
func (m *Model) updateViewport() {
	if !m.ready {
		return
	}
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(m.content.String())
	if atBottom {
		m.viewport.GotoBottom()
		m.newOutput = false
	} else {
		m.newOutput = true
	}
}

// View は bubbletea の View を実装する。
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "Initializing..."
	}

	// Viewport
	viewportView := m.viewport.View()

	// 上スクロール中の「↓ New output」バッジ
	if m.newOutput && !m.viewport.AtBottom() {
		badge := newOutputBadge.Render("↓ New output")
		// viewport の最終行に右寄せでオーバーレイ
		lines := strings.Split(viewportView, "\n")
		if len(lines) > 0 {
			lastIdx := len(lines) - 1
			padding := m.width - lipgloss.Width(badge)
			if padding < 0 {
				padding = 0
			}
			lines[lastIdx] = strings.Repeat(" ", padding) + badge
			viewportView = strings.Join(lines, "\n")
		}
	}

	// ステータスバー
	statusBar := statusBarStyle.Width(m.width).Render(m.statusLine)

	// 入力欄
	inputView := inputPrefixStyle.Render("> ") + m.textInput.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		viewportView,
		statusBar,
		inputView,
	)
}
