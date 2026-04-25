package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

// configPane は config screen 内の3ペインのいずれかを表す。
type configPane = configscreen.Pane

const (
	paneCategory = configscreen.PaneCategory
	paneField    = configscreen.PaneField
	paneDetail   = configscreen.PaneDetail
)

// configEditMode はフィールド編集中のモードを表す。
type configEditMode = configscreen.EditMode

const (
	editNone      = configscreen.EditNone
	editInput     = configscreen.EditInput
	editSelect    = configscreen.EditSelect
	editSlice     = configscreen.EditSlice
	editStructMap = configscreen.EditStructMap
)

// configSaveStatus は保存状態を表す。
type configSaveStatus = configscreen.SaveStatus

const (
	statusSaved    = configscreen.StatusSaved
	statusModified = configscreen.StatusModified
	statusSaving   = configscreen.StatusSaving
	statusFailed   = configscreen.StatusFailed
)

type configLayout = configscreen.Layout

// configScreen は /config TUI 画面の全状態を保持する。
type configScreen struct {
	cfg                    *config.Config
	categories             []config.ConfigCategory
	initialDefaultProvider string

	activePane  configPane
	catIndex    int
	fieldIndex  int
	fieldScroll int

	editMode         configEditMode
	editInput        textinput.Model
	editSelect       int
	editSliceItems   []string
	editSliceIndex   int
	editSliceInput   textinput.Model
	editSliceAdding  bool
	editSliceEditing bool

	editStructKeys   []string
	editStructIndex  int
	editStructInput  textinput.Model
	editStructAdding bool

	editEntryActive    bool
	editEntryKey       string
	editEntryFields    []structEntryField
	editEntryIndex     int
	editEntryFieldEdit string

	dirty      bool
	saveStatus configSaveStatus
	saveError  string

	confirmQuit  bool
	confirmIdx   int
	pendingClose bool

	filterMode  bool
	filterInput textinput.Model
	filterText  string
}

// structEntryField は structmap entry 内の1フィールドを表す。
type structEntryField struct {
	Name  string
	Type  string
	Value interface{}
}
