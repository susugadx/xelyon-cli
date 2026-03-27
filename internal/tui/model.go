package tui

import (
	"fmt"
	"log"
	"os"
	"strconv"
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
	"j/k:move • v:visual • V:lines • yy:copy • Tab:blocks • q:back",
	"j/k • v • V • yy • Tab • q",
}
var statusHintsVisual = []string{
	"-- VISUAL -- j/k:lines • h/l:chars • y:copy • Esc:cancel",
	"-- VISUAL -- y • Esc",
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
	renderedLines        []string        // viewport に渡す現在幅向けの行データ
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
	yPressed             bool // y キーが1回押された状態（yy用）
	cursorLine           int  // NAVモードの現在行（rawLines基準）
	cursorCol            int  // NAVモードの現在列（表示幅基準）
	visualMode           int  // 0=OFF, 1=文字単位, 2=行単位
	visualStart          visualPosition
	transientStatus      string    // 一時通知メッセージ
	transientStatusUntil time.Time // 一時通知の有効期限
	lastInterrupt        time.Time
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
		// ホイールイベントのみ viewport スクロール。それ以外は完全に無視。
		// textInput に MouseMsg を渡すと挙動がおかしくなるため即 return。
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.vp.scrollUp(3)
		case tea.MouseButtonWheelDown:
			m.vp.scrollDown(3)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

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
			m.rebuildRenderedLines()
			m.vp.setLines(m.renderedLines)
			m.vp.gotoBottom()
			m.ready = true
		} else {
			m.vp.width = m.width
			m.vp.height = viewportHeight
			if widthChanged {
				m.rebuildRenderedLines()
			}
			m.vp.setLines(m.renderedLines)
			if wasAtBottom {
				m.vp.gotoBottom()
			}
		}
		m.textInput.Width = max(0, m.width-inputPromptWidth-1)
		m.padLineCache = "\033[48;5;236m" + strings.Repeat(" ", m.width) + "\033[0m"
		m.chromeDirty = true

	case AppendMessageMsg:
		cmds = append(cmds, m.appendMessage(msg.Message))

	case AppendToolResultMsg:
		cmds = append(cmds, m.appendToolResult(msg.Tool))

	case StreamTextMsg:
		cmds = append(cmds, m.appendStreamText(msg.Text))
		if msg.Done {
			m.statusLine = m.agent.GetStatusLine()
			m.chromeDirty = true
		}

	case UpdateStatusMsg:
		m.statusLine = msg.Line
		m.chromeDirty = true

	case AgentDoneMsg:
		m.statusLine = m.agent.GetStatusLine()
		m.chromeDirty = true

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		if m.agent.IsProcessing() {
			m.statusLine = m.agent.GetStatusLine()
		}
		m.chromeDirty = true // スピナーフレーム変化
	}

	// lightViewport は状態を持たないので、キー以外のイベントでの Update は不要

	// textInput の更新（KeyMsg 以外）
	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	// chrome（入力欄+ステータスバー）を再構築。
	// スクロール時は chromeDirty=false なのでスキップされ、bubbletea の diff で変化なし→描画スキップ。
	if m.chromeDirty {
		m.chromeDirty = false
		m.rebuildChrome()
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

	// Ctrl+C は常に最優先
	if msg.Type == tea.KeyCtrlC {
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
	}

	// ナビゲーションモード
	if m.navigationMode {
		return m.handleNavigationKey(msg)
	}

	// 入力モード
	switch {
	case msg.Type == tea.KeyEsc:
		// 入力欄が空の場合のみ NAV モードに入る
		if strings.TrimSpace(m.textInput.Value()) == "" {
			m.navigationMode = true
			m.syncCursorToViewportTop()
			m.cursorCol = 0
			m.textInput.Blur()
			m.chromeDirty = true
			return m, nil
		}

	case isEnterKey(msg):
		tuiDebugf("Enter detected, textInput value=%q", m.textInput.Value())
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return m, nil
		}
		m.textInput.Reset()

		m.appendMessage(ChatMessage{
			Role:      "user",
			Content:   input,
			Timestamp: time.Now(),
		})

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

		return m, m.sendChat(input)

	case msg.Type == tea.KeyUp:
		m.vp.scrollUp(1)
		return m, nil
	case msg.Type == tea.KeyDown:
		m.vp.scrollDown(1)
		return m, nil
	case msg.Type == tea.KeyPgUp:
		m.vp.scrollUp(m.vp.height)
		return m, nil
	case msg.Type == tea.KeyPgDown:
		m.vp.scrollDown(m.vp.height)
		return m, nil
	}

	// その他のキーは textInput に渡す
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.chromeDirty = true
	return m, cmd
}

// handleNavigationKey はナビゲーションモードのキー処理。
func (m Model) handleNavigationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.visualMode != visualModeOff {
			m.clearVisualSelection()
			m.chromeDirty = true
			return m, nil
		}
		// ブロックフォーカス中 → フォーカス解除
		if m.focusedBlock >= 0 {
			m.clearBlockFocus()
			m.chromeDirty = true
			return m, nil
		}
		// NAVモード終了
		m.clearVisualSelection()
		m.navigationMode = false
		m.resetNavPending()
		m.textInput.Focus()
		m.chromeDirty = true
		return m, nil
	case tea.KeyEnter:
		// ブロックフォーカス中 → 折りたたみトグル
		if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
			m.toggleToolBlock(m.focusedBlock)
			return m, nil
		}
		// NAVモード終了
		m.clearVisualSelection()
		if m.focusedBlock >= 0 {
			m.clearBlockFocus()
		}
		m.navigationMode = false
		m.resetNavPending()
		m.textInput.Focus()
		m.chromeDirty = true
		return m, nil
	case tea.KeyTab:
		if m.visualMode != visualModeOff {
			return m, nil
		}
		// Tab: フォーカス中 → トグル、未フォーカス → 最後のブロックにフォーカス
		if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
			m.toggleToolBlock(m.focusedBlock)
		} else if len(m.toolBlocks) > 0 {
			m.setBlockFocus(len(m.toolBlocks) - 1)
			m.chromeDirty = true
		}
		return m, nil
	}

	s := msg.String()

	if m.yPressed {
		m.yPressed = false
		if s == "y" && m.focusedBlock < 0 && m.visualMode == visualModeOff {
			m.copyCursorLine()
			return m, nil
		}
		if m.focusedBlock < 0 && m.visualMode == visualModeOff {
			m.copyDefaultSelectionTarget()
		}
	}

	// g の2回押し判定
	if m.gPressed {
		m.gPressed = false
		if s == "g" && m.focusedBlock < 0 {
			m.cursorLine = 0
			m.vp.gotoTop()
			m.ensureCursorVisible()
			return m, nil
		}
		// g + 別キー → リセットして通常ナビ処理に落とす
	}

	switch s {
	case "q", "i":
		m.clearVisualSelection()
		if m.focusedBlock >= 0 {
			m.clearBlockFocus()
		}
		m.navigationMode = false
		m.resetNavPending()
		m.textInput.Focus()
		m.chromeDirty = true
		return m, nil
	case "j":
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock + 1)
		} else {
			m.moveCursorTo(m.cursorLine + 1)
		}
	case "k":
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock - 1)
		} else {
			m.moveCursorTo(m.cursorLine - 1)
		}
	case "h":
		if m.focusedBlock < 0 {
			m.moveCursorCol(-1)
		}
	case "l":
		if m.focusedBlock < 0 {
			m.moveCursorCol(1)
		}
	case "d":
		if m.focusedBlock < 0 {
			m.moveCursorTo(m.cursorLine + max(1, m.vp.height/2))
		}
	case "u":
		if m.focusedBlock < 0 {
			m.moveCursorTo(m.cursorLine - max(1, m.vp.height/2))
		}
	case "G":
		if m.focusedBlock < 0 {
			m.moveCursorTo(len(m.rawLines) - 1)
		}
	case "g":
		if m.focusedBlock < 0 {
			m.gPressed = true
		}
	case "v":
		if m.focusedBlock < 0 {
			m.visualMode = visualModeChar
			m.visualStart = visualPosition{line: m.cursorLine, col: m.cursorCol}
			m.yPressed = false
			m.chromeDirty = true
		}
	case "V":
		if m.focusedBlock < 0 {
			m.visualMode = visualModeLine
			m.visualStart = visualPosition{line: m.cursorLine, col: 0}
			m.yPressed = false
			m.chromeDirty = true
		}
	case "y":
		if m.visualMode != visualModeOff {
			m.copyVisualSelection()
		} else if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
			// フォーカス中のブロック内容をコピー
			content := m.toolBlocks[m.focusedBlock].tool.Detail
			if err := m.agent.CopyText(content); err == nil {
				m.setTransientStatus("✅ Copied block to clipboard")
			} else {
				m.setTransientStatus("Copy failed: " + err.Error())
			}
		} else {
			m.yPressed = true
		}
	default:
		// スクロールキーもサポート
		switch msg.Type {
		case tea.KeyUp:
			if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
				m.moveBlockFocus(m.focusedBlock - 1)
			} else {
				m.moveCursorTo(m.cursorLine - 1)
			}
		case tea.KeyDown:
			if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
				m.moveBlockFocus(m.focusedBlock + 1)
			} else {
				m.moveCursorTo(m.cursorLine + 1)
			}
		case tea.KeyPgUp:
			if m.focusedBlock < 0 {
				m.moveCursorTo(m.cursorLine - m.vp.height)
			}
		case tea.KeyPgDown:
			if m.focusedBlock < 0 {
				m.moveCursorTo(m.cursorLine + m.vp.height)
			}
		}
	}
	return m, nil
}

// sendChat は goroutine で agent.Chat を呼び出す tea.Cmd を返す。
func (m Model) sendChat(input string) tea.Cmd {
	return func() tea.Msg {
		m.agent.Chat(input)
		return AgentDoneMsg{}
	}
}

// appendMessage は会話ログにメッセージを追加する。
func (m *Model) appendMessage(msg ChatMessage) tea.Cmd {
	m.messages = append(m.messages, msg)

	switch msg.Role {
	case "user":
		return m.appendContentLines("", "> "+msg.Content, "")
	default:
		return m.appendContentLines(strings.Split(msg.Content, "\n")...)
	}
}

// appendStreamText はストリーミングテキストを追加する。
func (m *Model) appendStreamText(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	return m.appendContentLines(strings.Split(text, "\n")...)
}

// appendSystemInfo はシステム情報メッセージを追加する。
func (m *Model) appendSystemInfo(text string) tea.Cmd {
	return m.appendMessage(ChatMessage{
		Role:      "system_info",
		Content:   text,
		Timestamp: time.Now(),
	})
}

// appendContentLines は生ログと描画済みログの両方に新しい行を追加する。
func (m *Model) appendContentLines(lines ...string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}
	m.rawLines = append(m.rawLines, lines...)
	for _, line := range lines {
		m.renderedLines = append(m.renderedLines, m.renderLine(line))
	}
	m.clampCursorLine()
	if m.ready {
		atBottom := m.vp.atBottom()
		m.vp.setLines(m.renderedLines)
		if atBottom {
			m.vp.gotoBottom()
			m.newOutput = false
		} else {
			m.newOutput = true
		}
	}
	return nil
}

// appendToolResult はツール結果ブロックを追加する。
func (m *Model) appendToolResult(tool ToolResult) tea.Cmd {
	blockIdx := len(m.toolBlocks)
	lineStart := len(m.rawLines)

	block := toolBlockInfo{
		lineStart: lineStart,
		tool:      tool,
	}
	m.toolBlocks = append(m.toolBlocks, block)

	lines := m.buildToolBlockLines(blockIdx)
	m.toolBlocks[blockIdx].lineCount = len(lines)

	return m.appendContentLines(lines...)
}

// buildToolBlockLines はツールブロックの表示行を生成する。
func (m *Model) buildToolBlockLines(blockIdx int) []string {
	block := &m.toolBlocks[blockIdx]
	focused := m.focusedBlock == blockIdx

	indicator := " "
	if focused {
		indicator = "→"
	}

	prefix := "▶"
	if !block.tool.Collapsed {
		prefix = "▼"
	}

	summary := indicator + prefix + " " + block.tool.Summary

	if block.tool.Collapsed {
		return []string{summary}
	}

	// 展開状態: サマリー + インデント済み詳細行
	lines := []string{summary}
	for _, line := range strings.Split(block.tool.Detail, "\n") {
		lines = append(lines, "  "+line)
	}
	return lines
}

// toggleToolBlock はツールブロックの折りたたみ/展開をトグルする。
func (m *Model) toggleToolBlock(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	block := &m.toolBlocks[blockIdx]
	block.tool.Collapsed = !block.tool.Collapsed

	newLines := m.buildToolBlockLines(blockIdx)
	oldCount := block.lineCount
	newCount := len(newLines)

	// rawLines をスプライス: 旧行を除去して新行を挿入
	after := make([]string, len(m.rawLines[block.lineStart+oldCount:]))
	copy(after, m.rawLines[block.lineStart+oldCount:])
	m.rawLines = append(m.rawLines[:block.lineStart], newLines...)
	m.rawLines = append(m.rawLines, after...)

	block.lineCount = newCount

	// 後続ブロックの位置を更新
	delta := newCount - oldCount
	for i := blockIdx + 1; i < len(m.toolBlocks); i++ {
		m.toolBlocks[i].lineStart += delta
	}

	// 描画行を再構築して viewport に反映
	m.rebuildRenderedLines()
	if m.ready {
		m.vp.setLines(m.renderedLines)
	}
}

// setBlockFocus はブロックフォーカスを設定する。
func (m *Model) setBlockFocus(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	m.clearVisualSelection()
	old := m.focusedBlock
	m.focusedBlock = blockIdx
	m.cursorLine = m.toolBlocks[blockIdx].lineStart
	m.updateBlockIndicator(old)
	m.updateBlockIndicator(m.focusedBlock)
	m.scrollToBlock(m.focusedBlock)
}

// clearBlockFocus はブロックフォーカスを解除する。
func (m *Model) clearBlockFocus() {
	old := m.focusedBlock
	m.focusedBlock = -1
	m.updateBlockIndicator(old)
}

// moveBlockFocus はブロックフォーカスを移動する。
func (m *Model) moveBlockFocus(newIdx int) {
	newIdx = max(0, min(newIdx, len(m.toolBlocks)-1))
	if newIdx == m.focusedBlock {
		return
	}
	m.setBlockFocus(newIdx)
}

// updateBlockIndicator はブロックのフォーカスインジケータを更新する。
func (m *Model) updateBlockIndicator(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	block := &m.toolBlocks[blockIdx]
	focused := m.focusedBlock == blockIdx

	indicator := " "
	if focused {
		indicator = "→"
	}

	prefix := "▶"
	if !block.tool.Collapsed {
		prefix = "▼"
	}

	newFirstLine := indicator + prefix + " " + block.tool.Summary
	if block.lineStart < len(m.rawLines) {
		m.rawLines[block.lineStart] = newFirstLine
		if block.lineStart < len(m.renderedLines) {
			m.renderedLines[block.lineStart] = m.renderLine(newFirstLine)
		}
		if m.ready {
			m.vp.setLines(m.renderedLines)
		}
	}
}

// scrollToBlock はブロックの先頭行が表示されるようにスクロールする。
func (m *Model) scrollToBlock(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	block := &m.toolBlocks[blockIdx]
	// ブロックの先頭行をビューポートの上部付近に配置
	target := max(0, block.lineStart-2)
	maxOffset := m.vp.maxYOffset()
	if target > maxOffset {
		target = maxOffset
	}
	m.vp.yOffset = target
}

// setTransientStatus は一時通知メッセージを設定する。
func (m *Model) setTransientStatus(text string) {
	m.transientStatus = text
	m.transientStatusUntil = time.Now().Add(2 * time.Second)
	m.chromeDirty = true
}

func (m *Model) resetNavPending() {
	m.gPressed = false
	m.yPressed = false
}

func (m *Model) clearVisualSelection() {
	m.visualMode = visualModeOff
	m.visualStart = visualPosition{line: -1, col: -1}
	m.yPressed = false
}

func (m *Model) clampCursorLine() {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorLine >= len(m.rawLines) {
		m.cursorLine = len(m.rawLines) - 1
	}
	m.clampCursorCol()
}

func (m *Model) syncCursorToViewportTop() {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	m.cursorLine = max(0, min(m.vp.yOffset, len(m.rawLines)-1))
	m.clampCursorCol()
}

func (m *Model) ensureCursorVisible() {
	m.clampCursorLine()
	if m.vp.height <= 0 {
		return
	}
	if m.cursorLine < m.vp.yOffset {
		m.vp.yOffset = m.cursorLine
	}
	if m.cursorLine >= m.vp.yOffset+m.vp.height {
		m.vp.yOffset = m.cursorLine - m.vp.height + 1
	}
	if m.vp.yOffset > m.vp.maxYOffset() {
		m.vp.yOffset = m.vp.maxYOffset()
	}
	if m.vp.yOffset < 0 {
		m.vp.yOffset = 0
	}
}

func (m *Model) moveCursorTo(line int) {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	m.cursorLine = line
	m.clampCursorLine()
	m.ensureCursorVisible()
	m.chromeDirty = true
}

func (m *Model) moveCursorCol(delta int) {
	if len(m.rawLines) == 0 {
		m.cursorCol = 0
		return
	}
	m.cursorCol += delta
	m.clampCursorCol()
	m.chromeDirty = true
}

func (m *Model) clampCursorCol() {
	maxCol := m.maxCursorColForLine(m.cursorLine)
	if m.cursorCol < 0 {
		m.cursorCol = 0
	}
	if m.cursorCol > maxCol {
		m.cursorCol = maxCol
	}
}

func (m Model) maxCursorColForLine(line int) int {
	if line < 0 || line >= len(m.rawLines) {
		return 0
	}
	width := lipgloss.Width(stripANSI(m.rawLines[line]))
	if width <= 0 {
		return 0
	}
	return width - 1
}

func (m Model) lineSelectionRange() (start, end int, ok bool) {
	if m.visualMode == visualModeOff || m.visualStart.line < 0 {
		return 0, 0, false
	}
	start = min(m.visualStart.line, m.cursorLine)
	end = max(m.visualStart.line, m.cursorLine)
	return start, end, true
}

func (m Model) normalizedCharSelection() (start, end visualPosition, ok bool) {
	if m.visualMode != visualModeChar || m.visualStart.line < 0 {
		return visualPosition{}, visualPosition{}, false
	}
	start = m.visualStart
	end = visualPosition{line: m.cursorLine, col: m.cursorCol}
	if start.line > end.line || (start.line == end.line && start.col > end.col) {
		start, end = end, start
	}
	return start, end, true
}

func (m *Model) copyCursorLine() {
	if err := m.copyRawRangePlain(m.cursorLine, m.cursorLine); err != nil {
		m.setTransientStatus("Copy failed: " + err.Error())
		return
	}
	m.setTransientStatus("✅ Copied 1 line")
}

func (m *Model) copyVisualSelection() {
	switch m.visualMode {
	case visualModeChar:
		text, lines := m.copyCharVisualSelectionText()
		if err := m.agent.CopyText(text); err != nil {
			m.setTransientStatus("Copy failed: " + err.Error())
			return
		}
		m.clearVisualSelection()
		m.setTransientStatus("✅ Copied " + lineLabel(lines))
	case visualModeLine:
		start, end, ok := m.lineSelectionRange()
		if !ok {
			return
		}
		if err := m.copyRawRangePlain(start, end); err != nil {
			m.setTransientStatus("Copy failed: " + err.Error())
			return
		}
		m.clearVisualSelection()
		m.setTransientStatus("✅ Copied " + lineLabel(end-start+1))
	}
}

func (m Model) copyCharVisualSelectionText() (string, int) {
	start, end, ok := m.normalizedCharSelection()
	if !ok || len(m.rawLines) == 0 {
		return "", 0
	}

	var result strings.Builder
	for i := start.line; i <= end.line; i++ {
		line := stripANSI(m.rawLines[i])
		runes := []rune(line)
		from := 0
		to := len(runes)

		if i == start.line {
			from = displayColToRuneIndex(line, start.col)
		}
		if i == end.line {
			to = displayColToRuneIndexAfter(line, end.col)
		}
		if from > len(runes) {
			from = len(runes)
		}
		if to > len(runes) {
			to = len(runes)
		}
		if from > to {
			from = to
		}

		if i > start.line {
			result.WriteByte('\n')
		}
		result.WriteString(string(runes[from:to]))
	}

	return result.String(), end.line - start.line + 1
}

func (m *Model) copyDefaultSelectionTarget() {
	if msg, err := m.agent.CopyLastOutput(); err == nil {
		m.setTransientStatus("✅ " + msg)
	} else {
		m.setTransientStatus("Copy failed: " + err.Error())
	}
}

func (m Model) copyRawRangePlain(start, end int) error {
	if len(m.rawLines) == 0 {
		return fmt.Errorf("no lines to copy")
	}
	start = max(0, start)
	end = min(len(m.rawLines)-1, end)
	if start > end {
		return nil
	}

	lines := make([]string, 0, end-start+1)
	for _, line := range m.rawLines[start : end+1] {
		lines = append(lines, stripANSI(line))
	}
	return m.agent.CopyText(strings.Join(lines, "\n"))
}

func lineLabel(n int) string {
	if n == 1 {
		return "1 line"
	}
	return strconv.Itoa(n) + " lines"
}

// rebuildRenderedLines は現在幅に合わせて表示用の行キャッシュを再構築する。
func (m *Model) rebuildRenderedLines() {
	m.renderedLines = make([]string, len(m.rawLines))
	for i, line := range m.rawLines {
		m.renderedLines[i] = m.renderLine(line)
	}
}

// renderLine は現在幅に合わせて1行だけ表示用に切り詰める。
func (m *Model) renderLine(line string) string {
	if m.width <= 0 || len(line) <= m.width {
		return line
	}
	if lipgloss.Width(line) <= m.width {
		return line
	}
	return truncateWithANSI(line, m.width)
}

// truncateWithANSI は ANSI エスケープを保持しつつ表示幅を制限する。
// CJK 全角文字（幅2）を正しくカウントする。
func truncateWithANSI(s string, maxWidth int) string {
	var result strings.Builder
	width := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		w := runeWidth(r)
		if width+w > maxWidth {
			break
		}
		result.WriteRune(r)
		width += w
	}
	return result.String()
}

// runeWidth は文字の表示幅を返す（CJK 全角 = 2、それ以外 = 1）。
func runeWidth(r rune) int {
	// CJK Unified Ideographs, Hiragana, Katakana, Fullwidth Forms, etc.
	if r >= 0x1100 && // Korean Jamo
		(r <= 0x115F || r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0x303E) || // CJK Radicals, Kangxi, CJK Symbols
			(r >= 0x3040 && r <= 0x33BF) || // Hiragana, Katakana, Bopomofo, etc.
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
			(r >= 0x4E00 && r <= 0xA4CF) || // CJK Unified Ideographs, Yi
			(r >= 0xA960 && r <= 0xA97C) || // Hangul Jamo Extended-A
			(r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
			(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
			(r >= 0xFE10 && r <= 0xFE6F) || // Vertical forms, CJK Compatibility Forms
			(r >= 0xFF01 && r <= 0xFF60) || // Fullwidth Forms
			(r >= 0xFFE0 && r <= 0xFFE6) || // Fullwidth Signs
			(r >= 0x1F000 && r <= 0x1FFFF) || // Emoji, Mahjong, etc.
			(r >= 0x20000 && r <= 0x3FFFF)) { // CJK Unified Ideographs Extensions
		return 2
	}
	return 1
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func displayColToRuneIndex(s string, col int) int {
	if col <= 0 {
		return 0
	}
	width := 0
	for idx, r := range []rune(s) {
		next := width + runeWidth(r)
		if col < next {
			return idx
		}
		width = next
	}
	return len([]rune(s))
}

func displayColToRuneIndexAfter(s string, col int) int {
	if col < 0 {
		return 0
	}
	width := 0
	for idx, r := range []rune(s) {
		next := width + runeWidth(r)
		if col < next {
			return idx + 1
		}
		width = next
	}
	return len([]rune(s))
}

func stylePlainTextRange(s string, startCol, endCol int, bg string) string {
	if bg == "" || startCol >= endCol {
		return s
	}
	var result strings.Builder
	width := 0
	for _, r := range s {
		rw := runeWidth(r)
		if width < endCol && width+rw > startCol {
			result.WriteString(bg)
			result.WriteRune(r)
			result.WriteString("\033[0m")
		} else {
			result.WriteRune(r)
		}
		width += rw
	}
	return result.String()
}

func stylePlainTextRangeWithCursor(s string, startCol, endCol int, rangeBg string, cursorCol int, cursorBg string, lineBg string) string {
	var result strings.Builder
	width := 0
	cursorPainted := false
	for _, r := range s {
		rw := runeWidth(r)
		if lineBg != "" {
			result.WriteString(lineBg)
		}
		if rangeBg != "" && startCol < endCol && width < endCol && width+rw > startCol {
			result.WriteString(rangeBg)
		}
		if !cursorPainted && cursorCol < width+rw {
			result.WriteString(cursorBg)
			cursorPainted = true
		}
		result.WriteRune(r)
		result.WriteString("\033[0m")
		width += rw
	}
	return result.String()
}

func (m Model) charSelectionColumnsForLine(line int) (startCol, endCol int, ok bool) {
	start, end, ok := m.normalizedCharSelection()
	if !ok || line < start.line || line > end.line {
		return 0, 0, false
	}

	plain := stripANSI(m.rawLines[line])
	lineWidth := lipgloss.Width(plain)
	switch {
	case start.line == end.line:
		return start.col, min(lineWidth, end.col+1), true
	case line == start.line:
		return start.col, lineWidth, true
	case line == end.line:
		return 0, min(lineWidth, end.col+1), true
	default:
		return 0, lineWidth, true
	}
}

func decorateViewportLine(line string, width int, bg string) string {
	if bg == "" {
		return line
	}
	padded := strings.ReplaceAll(line, "\033[0m", "\033[0m"+bg)
	padding := max(0, width-lipgloss.Width(line))
	return bg + padded + strings.Repeat(" ", padding) + "\033[0m"
}

func (m Model) viewportView() string {
	if !m.navigationMode || m.focusedBlock >= 0 {
		return m.vp.view()
	}

	const cursorLineBg = "\033[48;5;236m"
	const cursorCharBg = "\033[48;5;255;38;5;16m"
	const visualBg = "\033[48;5;240m"
	const visualCursorBg = "\033[48;5;255;38;5;16m"

	visible := m.vp.visibleLines()
	var sb strings.Builder
	sb.Grow(m.vp.height * (m.vp.width + 1))

	for i := 0; i < m.vp.height; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i >= len(visible) {
			continue
		}

		rawIdx := m.vp.yOffset + i
		line := visible[i]

		switch m.visualMode {
		case visualModeChar:
			if startCol, endCol, ok := m.charSelectionColumnsForLine(rawIdx); ok {
				plain := stripANSI(line)
				styled := stylePlainTextRange(plain, startCol, endCol, visualBg)
				if rawIdx == m.cursorLine {
					styled = stylePlainTextRangeWithCursor(plain, startCol, endCol, visualBg, m.cursorCol, visualCursorBg, "")
				}
				sb.WriteString(decorateViewportLine(styled, m.vp.width, ""))
				continue
			}
			if rawIdx == m.cursorLine {
				plain := stripANSI(line)
				sb.WriteString(decorateViewportLine(stylePlainTextRangeWithCursor(plain, 0, 0, "", m.cursorCol, visualCursorBg, ""), m.vp.width, ""))
				continue
			}
		case visualModeLine:
			if start, end, ok := m.lineSelectionRange(); ok && rawIdx >= start && rawIdx <= end {
				if rawIdx == m.cursorLine {
					plain := stripANSI(line)
					sb.WriteString(decorateViewportLine(stylePlainTextRangeWithCursor(plain, 0, lipgloss.Width(plain), visualBg, m.cursorCol, visualCursorBg, ""), m.vp.width, visualBg))
					continue
				}
				sb.WriteString(decorateViewportLine(line, m.vp.width, visualBg))
				continue
			}
		}

		if rawIdx == m.cursorLine {
			plain := stripANSI(line)
			sb.WriteString(decorateViewportLine(stylePlainTextRangeWithCursor(plain, 0, 0, "", m.cursorCol, cursorCharBg, cursorLineBg), m.vp.width, cursorLineBg))
			continue
		}

		sb.WriteString(line)
	}

	return sb.String()
}

// rebuildChrome は入力欄+ステータスバーを再構築する。
// Update() 内で chromeDirty 時のみ呼ばれる（View() は値レシーバーなので書き込み不可）。
func (m *Model) rebuildChrome() {
	const inputBg = "\033[48;5;236m"
	const hintColor = "\033[38;5;244m"
	tiView := strings.ReplaceAll(m.textInput.View(), "\033[0m", "\033[0m"+inputBg)
	inputLine := inputBg + " \033[38;5;46m" + inputPrompt + "\033[38;5;252m" + tiView + "\033[0m"

	var statusText string
	if m.navigationMode {
		statusText = " \033[48;5;33;38;5;255m NAV \033[0m " + m.statusLine
	} else if m.agent.IsProcessing() {
		statusText = " " + m.spinner.View() + " " + m.statusLine
	} else {
		statusText = " " + m.statusLine
	}

	// スクロールアップ中の新出力通知
	if m.newOutput && !m.vp.atBottom() {
		statusText += "  \033[48;5;63;38;5;230m ↓ New output \033[0m"
	}

	// 一時通知があればステータスに上書き表示
	if m.transientStatus != "" && time.Now().Before(m.transientStatusUntil) {
		statusText += "  \033[38;5;82m" + m.transientStatus + "\033[0m"
	}

	hints := statusHintsNormal
	if m.navigationMode {
		if m.visualMode == visualModeChar {
			hints = statusHintsVisual
		} else if m.visualMode == visualModeLine {
			hints = statusHintsVisualLine
		} else if m.focusedBlock >= 0 {
			hints = statusHintsBlockFocus
		} else {
			hints = statusHintsNav
		}
	}
	statusBar := statusText
	for _, hint := range hints {
		padding := m.width - lipgloss.Width(statusText) - lipgloss.Width(hint)
		if padding >= 2 {
			statusBar = statusText + strings.Repeat(" ", padding) + hintColor + hint + "\033[0m"
			break
		}
	}

	m.chromeCache = m.padLineCache + "\n" + inputLine + "\n" + m.padLineCache + "\n" + statusBar
}

// syncViewport は高速スクロール領域全体を再描画する。
// View は bubbletea の View を実装する。
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "Initializing..."
	}
	return m.viewportView() + "\n" + m.chromeCache
}
