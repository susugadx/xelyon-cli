package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// openProjectScreen は project config screen を開く。
func (m Model) openProjectScreen() (tea.Model, tea.Cmd) {
	pc, err := m.projectAgent.LoadProjectForEdit()
	if err != nil {
		m.appendSystemInfo("Failed to load project config: " + err.Error())
		return m, nil
	}
	m.activateModalScreen(screenProject)
	m.installProjectScreen(pc)
	m.projectScreen.normalizeSize(m.width, m.height)
	return m, nil
}

// closeProjectScreen は project screen を閉じて chat に戻る。
func (m Model) closeProjectScreen() (tea.Model, tea.Cmd) {
	m.projectScreen = nil
	m.deactivateModalScreen(true)
	return m, nil
}

// updateProjectScreen は screenProject 中のメッセージ処理。
func (m Model) updateProjectScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	ps := m.projectScreen
	if ps == nil {
		return m.closeProjectScreen()
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyChatWindowSize(msg.Width, msg.Height)
		ps.normalizeSize(msg.Width, msg.Height)
		return m, nil

	case ProjectSavedMsg:
		result := ps.handleSaveResult(msg)
		if result.shouldClose {
			return m.closeProjectScreen()
		}
		if result.startQueued {
			return m.beginProjectSave(ps.pendingClose)
		}
		return m, nil

	case ProjectTemplateCreatedMsg:
		if msg.ScreenID != ps.screenID {
			return m, nil
		}
		if msg.Error != nil {
			ps.saveStatus = projectStatusFailed
			ps.saveError = msg.Error.Error()
			ps.message = ""
			return m, nil
		}
		m.installProjectScreen(msg.Config)
		m.projectScreen.normalizeSize(m.width, m.height)
		m.projectScreen.message = "template created"
		return m, nil

	case tea.KeyMsg:
		action, cmd := ps.handleKey(msg, m.conversation.IsProcessing())
		switch action {
		case projectCommandDelegateCtrlC:
			return m.handleCtrlC()
		case projectCommandClose:
			return m.closeProjectScreen()
		case projectCommandSave:
			return m.beginProjectSave(false)
		case projectCommandSaveAndClose:
			return m.beginProjectSave(true)
		case projectCommandCreateTemplate:
			ps.saveStatus = projectStatusSaving
			ps.saveError = ""
			ps.message = "creating template"
			return m, m.createProjectTemplateCmd(m.ensureProjectScreenID(ps))
		default:
			return m, cmd
		}

	default:
		return m.forwardMessageToChatFromModal(msg, screenProject)
	}
}

func (m *Model) installProjectScreen(pc *config.ProjectConfig) {
	m.projectScreenSeq++
	m.projectScreen = newProjectScreen(pc)
	m.projectScreen.screenID = m.projectScreenSeq
}

func (m *Model) ensureProjectScreenID(ps *projectScreen) int {
	if ps == nil {
		return 0
	}
	if ps.screenID == 0 {
		m.projectScreenSeq++
		ps.screenID = m.projectScreenSeq
	}
	return ps.screenID
}

func (m Model) beginProjectSave(closeOnSuccess bool) (tea.Model, tea.Cmd) {
	ps := m.projectScreen
	if ps == nil || ps.pc == nil {
		return m, nil
	}
	ps.confirmQuit = false
	closeIntent := ps.pendingClose || closeOnSuccess
	ps.pendingClose = closeIntent
	if ps.saveInFlight {
		ps.saveQueued = true
		ps.saveStatus = projectStatusSaving
		ps.saveError = ""
		ps.message = "save queued"
		return m, nil
	}
	ps.saveSeq++
	ps.saveInFlight = true
	ps.saveQueued = false
	ps.saveStatus = projectStatusSaving
	ps.saveError = ""
	ps.message = ""
	return m, m.saveProjectCmd(m.ensureProjectScreenID(ps), ps.saveSeq)
}

func (m Model) saveProjectCmd(screenID, saveSeq int) tea.Cmd {
	snapshot := config.CloneProjectConfig(m.projectScreen.pc)
	projectAgent := m.projectAgent
	return func() tea.Msg {
		err := projectAgent.SaveProjectConfig(snapshot)
		return ProjectSavedMsg{
			Error:    err,
			Snapshot: snapshot,
			ScreenID: screenID,
			SaveSeq:  saveSeq,
		}
	}
}

func (m Model) createProjectTemplateCmd(screenID int) tea.Cmd {
	projectAgent := m.projectAgent
	return func() tea.Msg {
		pc, err := projectAgent.CreateProjectConfigTemplate()
		return ProjectTemplateCreatedMsg{Error: err, Config: pc, ScreenID: screenID}
	}
}
