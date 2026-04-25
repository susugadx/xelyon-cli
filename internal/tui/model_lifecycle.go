package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// footerHeight は下部 chrome（入力欄+ステータスバー）の合計高さを返す。
// 将来の compact footer や compose mode では動的に切り替えられる。
func (m Model) footerHeight() int {
	return statusBarHeight + inputHeight + len(m.visibleComposerRows())
}

// NewModel は TUI Model を作成する。
func NewModel(agent AgentInterface, initialContent string) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 0 // 無制限
	ti.Width = 80    // 後で WindowSize で更新

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	initLines := []string{}
	if initialContent != "" {
		initLines = strings.Split(initialContent, "\n")
	}

	return Model{
		conversation: agent,
		commands:     agent,
		clipboard:    agent,
		configAgent:  agent,
		textInput:    ti,
		spinner:      sp,
		rawLines:     append([]string(nil), initLines...),
		messages:     []ChatMessage{},
		focusedBlock: -1,
		navigationState: navigationState{
			visualStart: visualPosition{line: -1, col: -1},
		},
		mouseSelectionState: mouseSelectionState{
			mouseSelAnchor: visualPosition{line: -1, col: -1},
			mouseSelEnd:    visualPosition{line: -1, col: -1},
		},
		statusLine: agent.GetStatusLine(),
	}
}

// Init は bubbletea の Init を実装する。
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

// applyChatWindowSize は chat 画面で使う viewport/layout/chrome を最新の端末サイズに同期する。
func (m *Model) applyChatWindowSize(width, height int) {
	wasAtBottom := m.ready && m.vp.atBottom()
	widthChanged := m.width != width
	m.width = width
	m.height = height

	viewportHeight := m.height - m.footerHeight()
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
	m.padLineCache = termtext.FillANSITextWidth("", m.width, theme.Chrome.InputBg)
	m.chromeDirty = true
}

// refreshStatusLine は agent の最新 runtime state から footer 用の statusLine を再取得する。
func (m *Model) refreshStatusLine() {
	m.statusLine = m.conversation.GetStatusLine()
	m.chromeDirty = true
}
