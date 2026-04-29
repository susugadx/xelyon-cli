package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type projectPane int

const (
	projectPaneSection projectPane = iota
	projectPaneItem
)

type projectSection int

const (
	projectSectionContext projectSection = iota
	projectSectionRules
	projectSectionConditional
	projectSectionIgnore
	projectSectionFinalCommands
	projectSectionFinalTimeout
)

type projectEditMode int

const (
	projectEditNone projectEditMode = iota
	projectEditContext
	projectEditLine
)

type projectLineEditKind int

const (
	projectLineEditNone projectLineEditKind = iota
	projectLineEditList
	projectLineEditTimeout
)

type projectSaveStatus int

const (
	projectStatusSaved projectSaveStatus = iota
	projectStatusModified
	projectStatusSaving
	projectStatusFailed
)

type projectCommand int

const (
	projectCommandNone projectCommand = iota
	projectCommandClose
	projectCommandSave
	projectCommandSaveAndClose
	projectCommandCreateTemplate
	projectCommandDelegateCtrlC
)

type projectScreen struct {
	screenID int

	pc                    *config.ProjectConfig
	missing               bool
	savedHasFinalChecks   bool
	tuiCreatedFinalChecks bool

	activePane   projectPane
	sectionIndex int
	itemIndex    map[projectSection]int

	editMode        projectEditMode
	lineEditKind    projectLineEditKind
	lineEditAdd     bool
	editInput       textinput.Model
	contextArea     textarea.Model
	contextDraft    string
	lineEditSection projectSection

	dirty      bool
	saveStatus projectSaveStatus
	saveError  string
	message    string

	confirmQuit  bool
	confirmIdx   int
	pendingClose bool
	saveSeq      int
	saveInFlight bool
	saveQueued   bool
}

type projectSectionInfo struct {
	section     projectSection
	title       string
	description string
}

var projectSections = []projectSectionInfo{
	{section: projectSectionContext, title: "Context", description: "Project context injected into prompts"},
	{section: projectSectionRules, title: "Rules", description: "Required rules injected into prompts"},
	{section: projectSectionConditional, title: "Conditional", description: "Path-scoped rules and context preview"},
	{section: projectSectionIgnore, title: "Ignore patterns", description: "Shared ignore patterns for project map/search"},
	{section: projectSectionFinalCommands, title: "Final check commands", description: "Commands for completed-with-changes checks"},
	{section: projectSectionFinalTimeout, title: "Final check timeout", description: "Timeout in seconds for final checks"},
}

func newProjectScreen(pc *config.ProjectConfig) *projectScreen {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 512

	area := textarea.New()
	area.Prompt = ""
	area.ShowLineNumbers = false
	area.CharLimit = 0
	area.SetHeight(8)

	ps := &projectScreen{
		pc:                  config.CloneProjectConfig(pc),
		missing:             pc == nil,
		savedHasFinalChecks: pc != nil && pc.FinalChecks != nil,
		itemIndex:           make(map[projectSection]int),
		editInput:           ti,
		contextArea:         area,
		saveStatus:          projectStatusSaved,
	}
	return ps
}

func (ps *projectScreen) normalizeSize(width, height int) {
	editorWidth := max(20, width-6)
	ps.editInput.Width = editorWidth
	ps.contextArea.SetWidth(editorWidth)
	ps.contextArea.SetHeight(max(4, min(12, height-6)))
}

func (ps *projectScreen) selectedSection() projectSection {
	if ps.sectionIndex < 0 {
		ps.sectionIndex = 0
	}
	if ps.sectionIndex >= len(projectSections) {
		ps.sectionIndex = len(projectSections) - 1
	}
	return projectSections[ps.sectionIndex].section
}

func (ps *projectScreen) selectedSectionInfo() projectSectionInfo {
	section := ps.selectedSection()
	for _, info := range projectSections {
		if info.section == section {
			return info
		}
	}
	return projectSections[0]
}
