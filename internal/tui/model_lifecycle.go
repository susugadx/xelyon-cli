package tui

import (
	"strings"
	"time"

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
	return statusBarHeight +
		inputHeight +
		len(m.visibleSlashSuggestionRows()) +
		m.visibleSlashSuggestionDetailRowCount() +
		m.visibleCompactChipRowCount() +
		len(m.visibleComposerDraftSummaryRows())
}

// NewModel は TUI Model を作成する。
func NewModel(agent AgentInterface, initialContent string) Model {
	return NewModelWithStartupOptions(agent, initialContent, StartupOptions{})
}

// NewModelWithStartupSubmission は TUI 起動直後に送信する入力を持つ Model を作成する。
func NewModelWithStartupSubmission(agent AgentInterface, initialContent string, startupSubmission *StartupSubmission) Model {
	return NewModelWithStartupOptions(agent, initialContent, StartupOptions{Submission: startupSubmission})
}

// StartupSessionPicker は TUI 起動直後に開く session picker の条件を表す。
type StartupSessionPicker struct {
	All bool
}

// StartupOptions は TUI 起動直後の追加動作を表す。
type StartupOptions struct {
	Submission    *StartupSubmission
	SessionPicker *StartupSessionPicker
}

// NewModelWithStartupOptions は TUI 起動直後の追加動作を持つ Model を作成する。
func NewModelWithStartupOptions(agent AgentInterface, initialContent string, opts StartupOptions) Model {
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

	statusSnapshot := agent.StatusSnapshot()
	if statusSnapshot.LegacyLine == "" {
		statusSnapshot.LegacyLine = agent.GetStatusLine()
	}
	reviewAgent, _ := agent.(ReviewAgent)
	skillCatalog, _ := agent.(SkillCatalogAgent)

	return Model{
		conversation:   agent,
		commands:       agent,
		clipboard:      agent,
		configAgent:    agent,
		providerModels: agent,
		sessions:       agent,
		projectAgent:   agent,
		reviewAgent:    reviewAgent,
		skillCatalog:   skillCatalog,
		textInput:      ti,
		spinner:        sp,
		rawLines:       append([]string(nil), initLines...),
		messages:       []ChatMessage{},
		focusedBlock:   -1,
		navigationState: navigationState{
			visualStart: visualPosition{line: -1, col: -1},
		},
		mouseSelectionState: mouseSelectionState{
			mouseSelAnchor: visualPosition{line: -1, col: -1},
			mouseSelEnd:    visualPosition{line: -1, col: -1},
		},
		statusLine:        statusSnapshot.LegacyLine,
		statusSnapshot:    statusSnapshot,
		workingDir:        currentWorkingDirForStatus(),
		startupSubmission: opts.Submission,
		startupPicker:     opts.SessionPicker,
	}
}

// Init は bubbletea の Init を実装する。
func (m Model) Init() tea.Cmd {
	cleanupStaleClipboardAttachmentTemps(time.Now())

	cmds := []tea.Cmd{
		textinput.Blink,
		m.spinner.Tick,
	}
	if cmd := startupSubmissionCmd(m.startupSubmission); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if m.startupPicker != nil {
		cmds = append(cmds, openSessionPickerCmd(m.startupPicker.All))
	}
	return tea.Batch(cmds...)
}

// applyChatWindowSize は chat 画面で使う viewport/layout/chrome を最新の端末サイズに同期する。
func (m *Model) applyChatWindowSize(width, height int) {
	wasAtBottom := m.ready && m.vp.atBottom()
	widthChanged := m.width != width
	m.width = width
	m.height = height

	viewportHeight := m.height - m.footerHeight()
	if viewportHeight < minChatViewportHeight {
		viewportHeight = minChatViewportHeight
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
	m.statusSnapshot = m.conversation.StatusSnapshot()
	m.statusLine = m.statusSnapshot.LegacyLine
	if m.statusLine == "" {
		m.statusLine = m.conversation.GetStatusLine()
		m.statusSnapshot.LegacyLine = m.statusLine
	}
	m.chromeDirty = true
}
