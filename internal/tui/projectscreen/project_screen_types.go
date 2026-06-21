package projectscreen

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

// Command は /project 画面が root orchestration に要求する処理を表す。
type Command int

const (
	CommandNone Command = iota
	CommandClose
	CommandSave
	CommandSaveAndClose
	CommandCreateTemplate
	CommandDelegateCtrlC
)

// Screen は /project 画面の state/input/render/save-result handling を保持する。
type Screen struct {
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

// Snapshot は root package や tests が参照する /project 画面状態の読み取り専用投影。
type Snapshot struct {
	ScreenID        int
	Config          *config.ProjectConfig
	Missing         bool
	Dirty           bool
	SaveStatus      string
	SaveError       string
	Message         string
	ConfirmQuit     bool
	PendingClose    bool
	SaveInFlight    bool
	SaveQueued      bool
	Section         string
	ActivePane      string
	EditMode        string
	ContextDraft    string
	ContextValue    string
	LineEditValue   string
	SelectedIndex   int
	SelectedItems   []string
	FinalCheckSaved bool
}

type projectSectionInfo struct {
	section     projectSection
	title       string
	description string
}

var projectSections = []projectSectionInfo{
	{section: projectSectionContext, title: "Legacy context", description: "Legacy xelyon.yaml context; prefer AGENTS.md for guidance"},
	{section: projectSectionRules, title: "Legacy rules", description: "Legacy mandatory rules; prefer AGENTS.md for guidance"},
	{section: projectSectionConditional, title: "Legacy conditional", description: "Path-scoped legacy rules/context preview"},
	{section: projectSectionIgnore, title: "Ignore patterns", description: "Shared ignore patterns for project map/search"},
	{section: projectSectionFinalCommands, title: "Final check commands", description: "Commands for completed-with-changes checks"},
	{section: projectSectionFinalTimeout, title: "Final check timeout", description: "Timeout in seconds for final checks"},
}

// New は project config の編集画面 state を生成する。
func New(pc *config.ProjectConfig, screenID int) *Screen {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 512

	area := textarea.New()
	area.Prompt = ""
	area.ShowLineNumbers = false
	area.CharLimit = 0
	area.SetHeight(8)

	ps := &Screen{
		screenID:            screenID,
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

// NormalizeSize は画面サイズに合わせて editor width/height を更新する。
func (ps *Screen) NormalizeSize(width, height int) {
	editorWidth := max(20, width-6)
	ps.editInput.Width = editorWidth
	ps.contextArea.SetWidth(editorWidth)
	ps.contextArea.SetHeight(max(4, min(12, height-6)))
}

func (ps *Screen) selectedSection() projectSection {
	if ps.sectionIndex < 0 {
		ps.sectionIndex = 0
	}
	if ps.sectionIndex >= len(projectSections) {
		ps.sectionIndex = len(projectSections) - 1
	}
	return projectSections[ps.sectionIndex].section
}

func (ps *Screen) selectedSectionInfo() projectSectionInfo {
	section := ps.selectedSection()
	for _, info := range projectSections {
		if info.section == section {
			return info
		}
	}
	return projectSections[0]
}

// Snapshot は外部 package から直接 field を読まずに状態を確認するための投影を返す。
func (ps *Screen) Snapshot() Snapshot {
	if ps == nil {
		return Snapshot{}
	}
	return Snapshot{
		ScreenID:        ps.screenID,
		Config:          config.CloneProjectConfig(ps.pc),
		Missing:         ps.missing,
		Dirty:           ps.dirty,
		SaveStatus:      ps.saveStatus.String(),
		SaveError:       ps.saveError,
		Message:         ps.message,
		ConfirmQuit:     ps.confirmQuit,
		PendingClose:    ps.pendingClose,
		SaveInFlight:    ps.saveInFlight,
		SaveQueued:      ps.saveQueued,
		Section:         ps.selectedSection().String(),
		ActivePane:      ps.activePane.String(),
		EditMode:        ps.editMode.String(),
		ContextDraft:    ps.contextDraft,
		ContextValue:    ps.contextArea.Value(),
		LineEditValue:   ps.editInput.Value(),
		SelectedIndex:   ps.selectedItemIndex(),
		SelectedItems:   append([]string(nil), ps.selectedItems()...),
		FinalCheckSaved: ps.savedHasFinalChecks,
	}
}

func (p projectPane) String() string {
	switch p {
	case projectPaneItem:
		return "item"
	default:
		return "section"
	}
}

func (s projectSection) String() string {
	switch s {
	case projectSectionContext:
		return "context"
	case projectSectionRules:
		return "rules"
	case projectSectionConditional:
		return "conditional"
	case projectSectionIgnore:
		return "ignore"
	case projectSectionFinalCommands:
		return "final_commands"
	case projectSectionFinalTimeout:
		return "final_timeout"
	default:
		return "unknown"
	}
}

func (m projectEditMode) String() string {
	switch m {
	case projectEditContext:
		return "context"
	case projectEditLine:
		return "line"
	default:
		return "none"
	}
}

func (s projectSaveStatus) String() string {
	switch s {
	case projectStatusModified:
		return "modified"
	case projectStatusSaving:
		return "saving"
	case projectStatusFailed:
		return "failed"
	default:
		return "saved"
	}
}
