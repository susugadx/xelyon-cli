package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/projectscreen"
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
	m.projectScreen.NormalizeSize(m.width, m.height)
	return m, nil
}

// closeProjectScreen は project screen を閉じて chat に戻る。
func (m Model) closeProjectScreen() (tea.Model, tea.Cmd) {
	m.projectScreen = nil
	m.deactivateModalScreen(true)
	return m, nil
}

func (m Model) projectView() string {
	if m.projectScreen == nil {
		return "Loading..."
	}
	return m.projectScreen.View(m.width, m.height)
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
		ps.NormalizeSize(msg.Width, msg.Height)
		return m, nil

	case projectscreen.SaveResult:
		result := ps.HandleSaveResult(msg)
		if result.ShouldClose {
			return m.closeProjectScreen()
		}
		if result.StartQueued {
			return m.beginProjectSave(false)
		}
		return m, nil

	case projectscreen.TemplateResult:
		if !ps.InstallTemplateResult(msg) {
			return m, nil
		}
		if m.projectScreen != nil {
			m.projectScreen.NormalizeSize(m.width, m.height)
		}
		return m, nil

	case tea.KeyMsg:
		action, cmd := ps.HandleKey(msg, m.conversation.IsProcessing())
		switch action {
		case projectscreen.CommandDelegateCtrlC:
			return m.handleCtrlC()
		case projectscreen.CommandClose:
			return m.closeProjectScreen()
		case projectscreen.CommandSave:
			return m.beginProjectSave(false)
		case projectscreen.CommandSaveAndClose:
			return m.beginProjectSave(true)
		case projectscreen.CommandCreateTemplate:
			return m, m.createProjectTemplateCmd(ps.Snapshot().ScreenID)
		default:
			return m, cmd
		}

	default:
		return m.forwardMessageToChatFromModal(msg, screenProject)
	}
}

func (m *Model) installProjectScreen(pc *config.ProjectConfig) {
	m.projectScreenSeq++
	m.projectScreen = projectscreen.New(pc, m.projectScreenSeq)
}

func (m Model) beginProjectSave(closeOnSuccess bool) (tea.Model, tea.Cmd) {
	ps := m.projectScreen
	pending, ok := ps.BeginSave(closeOnSuccess)
	if !ok {
		return m, nil
	}
	return m, m.saveProjectCmd(pending)
}

func (m Model) saveProjectCmd(pending projectscreen.PendingSave) tea.Cmd {
	projectAgent := m.projectAgent
	return func() tea.Msg {
		err := projectAgent.SaveProjectConfig(pending.Snapshot)
		return projectscreen.SaveResult{
			Error:    err,
			Snapshot: pending.Snapshot,
			ScreenID: pending.ScreenID,
			SaveSeq:  pending.SaveSeq,
		}
	}
}

func (m Model) createProjectTemplateCmd(screenID int) tea.Cmd {
	projectAgent := m.projectAgent
	return func() tea.Msg {
		pc, err := projectAgent.CreateProjectConfigTemplate()
		return projectscreen.TemplateResult{Error: err, Config: pc, ScreenID: screenID}
	}
}
