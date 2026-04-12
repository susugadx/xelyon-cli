package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// configPane は config screen 内の3ペインのいずれかを表す。
type configPane int

const (
	paneCategory configPane = iota
	paneField
	paneDetail
)

// configEditMode はフィールド編集中のモードを表す。
type configEditMode int

const (
	editNone      configEditMode = iota
	editInput                    // string/int/float の入力
	editSelect                   // select の選択
	editSlice                    // []string のサブビュー
	editStructMap                // structmap のサブビュー
)

// configSaveStatus は保存状態を表す。
type configSaveStatus int

const (
	statusSaved    configSaveStatus = iota
	statusModified                  // 未保存変更あり
	statusSaving                    // 保存中
	statusFailed                    // 保存失敗
)

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
