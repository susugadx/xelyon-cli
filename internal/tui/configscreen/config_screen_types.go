package configscreen

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// configPane は config screen 内の3ペインのいずれかを表す。
type configPane = Pane

const (
	paneCategory = PaneCategory
	paneField    = PaneField
	paneDetail   = PaneDetail
)

// configEditMode はフィールド編集中のモードを表す。
type configEditMode = EditMode

const (
	editNone      = EditNone
	editInput     = EditInput
	editSelect    = EditSelect
	editSlice     = EditSlice
	editStructMap = EditStructMap
)

// configSaveStatus は保存状態を表す。
type configSaveStatus = SaveStatus

const (
	statusSaved    = StatusSaved
	statusModified = StatusModified
	statusSaving   = StatusSaving
	statusFailed   = StatusFailed
)

type configLayout = Layout

// Screen は /config TUI 画面の全状態を保持する。
type Screen struct {
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

	editGuidanceChoices []guidanceFileChoice
	editGuidanceIndex   int

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

type guidanceFileChoice struct {
	Path   string
	Preset bool
}

// Snapshot は /config screen の表示・編集状態を読み取り専用に写す。
type Snapshot struct {
	CategoryIndex       int
	FieldIndex          int
	FieldScroll         int
	ActivePane          Pane
	EditMode            EditMode
	Dirty               bool
	SaveStatus          SaveStatus
	SaveError           string
	ConfirmQuit         bool
	ConfirmIndex        int
	PendingClose        bool
	SelectedField       *config.ConfigField
	EditStructKeys      []string
	EditStructIndex     int
	EditStructAdding    bool
	EditEntryActive     bool
	EditEntryKey        string
	EditEntryFields     []StructEntryField
	EditEntryIndex      int
	EditEntryFieldEdit  string
	EditSliceItems      []string
	EditSliceIndex      int
	EditSliceAdding     bool
	EditGuidanceChoices []GuidanceFileChoice
	EditSelect          int
	FilterMode          bool
	CategoryNames       []string
	FilteredFields      []config.ConfigField
}

// StructEntryField は structmap entry 内の1フィールドの読み取り用 snapshot。
type StructEntryField struct {
	Name  string
	Type  string
	Value interface{}
}

// GuidanceFileChoice は guidance file chooser の選択肢 snapshot。
type GuidanceFileChoice struct {
	Path   string
	Preset bool
}
