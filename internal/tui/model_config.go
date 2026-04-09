package tui

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	// データ
	cfg                    *config.Config
	categories             []config.ConfigCategory
	initialDefaultProvider string

	// ペインナビゲーション
	activePane  configPane
	catIndex    int
	fieldIndex  int
	fieldScroll int

	// 編集
	editMode         configEditMode
	editInput        textinput.Model
	editSelect       int             // select 型の候補インデックス
	editSliceItems   []string        // []string 編集中のアイテムリスト
	editSliceIndex   int             // []string のカーソル位置
	editSliceInput   textinput.Model // []string 新規/編集用入力
	editSliceAdding  bool            // []string 追加入力中
	editSliceEditing bool            // []string 編集入力中

	// structmap 編集
	editStructKeys   []string        // structmap のキー一覧
	editStructIndex  int             // structmap のカーソル位置
	editStructInput  textinput.Model // structmap キー入力用
	editStructAdding bool            // 新規キー追加中

	// structmap entry value 編集
	editEntryActive    bool               // entry 詳細編集中
	editEntryKey       string             // 編集中の entry key
	editEntryFields    []structEntryField // entry のフィールド一覧
	editEntryIndex     int                // entry field のカーソル
	editEntryFieldEdit string             // "" | "input" | "slice" — 個別フィールド編集中の種別

	// 状態
	dirty      bool
	saveStatus configSaveStatus
	saveError  string

	// 終了確認
	confirmQuit  bool
	confirmIdx   int  // 0=Save&quit, 1=Discard, 2=Cancel
	pendingClose bool // Save and quit で保存成功後に閉じるフラグ

	// フィルタ
	filterMode  bool
	filterInput textinput.Model
	filterText  string
}

// structEntryField は structmap entry 内の1フィールドを表す。
type structEntryField struct {
	Name  string      // 表示名（yaml key）
	Type  string      // "string", "int", "bool", "[]string"
	Value interface{} // 現在値
}

// providerModelEntryFields は ProviderModelConfig の user-facing フィールドを返す。
func providerModelEntryFields(pm config.ProviderModelConfig) []structEntryField {
	return []structEntryField{
		{Name: "default_model", Type: "string", Value: pm.DefaultModel},
		{Name: "max_output_tokens", Type: "int", Value: pm.MaxOutputTokens},
	}
}

// lspServerEntryFields は LSPServerConfig の user-facing フィールドを返す。
func lspServerEntryFields(ls config.LSPServerConfig) []structEntryField {
	return []structEntryField{
		{Name: "command", Type: "string", Value: ls.Command},
		{Name: "args", Type: "[]string", Value: ls.Args},
		{Name: "disabled", Type: "bool", Value: ls.Disabled},
	}
}

// loadEntryFields は path と key から entry フィールドを読み込む。
func (cs *configScreen) loadEntryFields(path, key string) []structEntryField {
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		if pm, ok := v[key]; ok {
			return providerModelEntryFields(pm)
		}
	case map[string]config.LSPServerConfig:
		if ls, ok := v[key]; ok {
			return lspServerEntryFields(ls)
		}
	}
	return nil
}

// applyEntryField は entry の1フィールドだけを Config にパッチ適用する。
func (cs *configScreen) applyEntryField(path, key string, ef structEntryField) {
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		_ = v
		cs.cfg.PatchProviderModelConfig(key, func(pm *config.ProviderModelConfig) {
			switch ef.Name {
			case "default_model":
				pm.DefaultModel, _ = ef.Value.(string)
			case "max_output_tokens":
				switch n := ef.Value.(type) {
				case int:
					pm.MaxOutputTokens = n
				}
			}
		})
	case map[string]config.LSPServerConfig:
		ls := v[key]
		switch ef.Name {
		case "command":
			ls.Command, _ = ef.Value.(string)
		case "args":
			if s, ok := ef.Value.([]string); ok {
				ls.Args = s
			}
		case "disabled":
			ls.Disabled, _ = ef.Value.(bool)
		}
		v[key] = ls
	}
}

// newConfigScreen は設定データから configScreen を初期化する。
func newConfigScreen(cfg *config.Config) *configScreen {
	categories := config.BuildConfigRegistry(cfg)

	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 256

	filterTI := textinput.New()
	filterTI.Prompt = "/"
	filterTI.CharLimit = 64

	sliceTI := textinput.New()
	sliceTI.Prompt = ""
	sliceTI.CharLimit = 256

	structTI := textinput.New()
	structTI.Prompt = ""
	structTI.CharLimit = 256

	return &configScreen{
		cfg:                    cfg,
		categories:             categories,
		initialDefaultProvider: cfg.DefaultProvider,
		editInput:              ti,
		filterInput:            filterTI,
		editSliceInput:         sliceTI,
		editStructInput:        structTI,
	}
}

// selectedCategory は現在選択中のカテゴリを返す。
func (cs *configScreen) selectedCategory() *config.ConfigCategory {
	if cs.catIndex >= 0 && cs.catIndex < len(cs.categories) {
		return &cs.categories[cs.catIndex]
	}
	return nil
}

// filteredFields はフィルタ適用済みのフィールドリストを返す。
func (cs *configScreen) filteredFields() []config.ConfigField {
	cat := cs.selectedCategory()
	if cat == nil {
		return nil
	}
	if cs.filterText == "" {
		return cat.Fields
	}
	lower := strings.ToLower(cs.filterText)
	var result []config.ConfigField
	for _, f := range cat.Fields {
		if strings.Contains(strings.ToLower(f.DisplayName), lower) ||
			strings.Contains(strings.ToLower(f.Path), lower) ||
			strings.Contains(strings.ToLower(f.Description), lower) {
			result = append(result, f)
		}
	}
	return result
}

// selectedField は現在選択中のフィールドを返す。
func (cs *configScreen) selectedField() *config.ConfigField {
	fields := cs.filteredFields()
	if cs.fieldIndex >= 0 && cs.fieldIndex < len(fields) {
		return &fields[cs.fieldIndex]
	}
	return nil
}

// refreshCategories はカテゴリを再構築する。
func (cs *configScreen) refreshCategories() {
	cs.categories = config.BuildConfigRegistry(cs.cfg)
}

// statusText は保存状態のテキストを返す。
func (cs *configScreen) statusText() string {
	switch cs.saveStatus {
	case statusModified:
		return "modified"
	case statusSaving:
		return "saving..."
	case statusFailed:
		return "save failed: " + cs.saveError
	default:
		return "saved"
	}
}

// openConfigScreen は config screen を開く。
func (m Model) openConfigScreen() (tea.Model, tea.Cmd) {
	cfg, err := m.agent.LoadConfigForEdit()
	if err != nil {
		m.appendSystemInfo("Failed to load config: " + err.Error())
		return m, nil
	}
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	m.navigationMode = false
	m.chromeDirty = true
	return m, nil
}

// closeConfigScreen は config screen を閉じて chat に戻る。
func (m Model) closeConfigScreen() (tea.Model, tea.Cmd) {
	m.screen = screenChat
	m.configScreen = nil
	m.refreshStatusLine()
	m.applyChatWindowSize(m.width, m.height)
	m.textInput.Focus()
	return m, nil
}

// updateConfigScreen は screenConfig 中のメッセージ処理。
// config screen が処理しないメッセージ（StreamTextMsg, AppendMessageMsg, AgentDoneMsg,
// spinner.TickMsg 等）は chat 側の状態にバッファする。
// これにより config screen 表示中も chat goroutine の出力が失われない。
func (m Model) updateConfigScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	if cs == nil {
		return m.closeConfigScreen()
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyChatWindowSize(msg.Width, msg.Height)
		m.normalizeConfigPaneState()
		return m, nil

	case ConfigSavedMsg:
		if msg.Error != nil {
			cs.saveStatus = statusFailed
			cs.saveError = msg.Error.Error()
			cs.pendingClose = false
		} else {
			cs.saveError = ""
			cs.refreshCategories()
			if reflect.DeepEqual(cs.cfg, msg.Snapshot) {
				cs.dirty = false
				cs.saveStatus = statusSaved
				m.refreshStatusLine()
				if cs.pendingClose {
					return m.closeConfigScreen()
				}
			} else {
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.pendingClose = false
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleConfigKey(msg)

	default:
		// config screen が関知しないメッセージ（StreamTextMsg, AppendMessageMsg,
		// AgentDoneMsg, spinner.TickMsg 等）は chat 側の状態更新に通す。
		// 一時的に screenChat に戻して処理し、screenConfig に復帰する。
		m.screen = screenChat
		updated, cmd := m.Update(msg)
		m = updated.(Model)
		m.screen = screenConfig
		return m, cmd
	}
}

func (m Model) configVisiblePanes() (fieldVisible, detailVisible bool) {
	_, midW, rightW := configPaneWidths(m.width)
	return midW > 0, rightW > 0
}

func (m Model) configEditTargetPane() configPane {
	_, detailVisible := m.configVisiblePanes()
	if detailVisible {
		return paneDetail
	}
	return paneField
}

func (m *Model) normalizeConfigPaneState() {
	cs := m.configScreen
	if cs == nil {
		return
	}
	_, detailVisible := m.configVisiblePanes()
	if cs.activePane == paneDetail && !detailVisible {
		cs.activePane = paneField
	}
}

// handleConfigKey は config screen のキー入力を処理する。
func (m Model) handleConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// Ctrl+C keeps request cancel semantics while preventing dirty config from being discarded.
	// confirmQuit 表示中でも processing 中の Ctrl+C は request cancel を優先する。
	if msg.Type == tea.KeyCtrlC {
		return m.handleConfigCtrlC()
	}

	// 終了確認ダイアログ
	if cs.confirmQuit {
		return m.handleConfigConfirmKey(msg)
	}

	// フィルタモード
	if cs.filterMode {
		return m.handleConfigFilterKey(msg)
	}

	// 編集モード
	if cs.editMode != editNone {
		return m.handleConfigEditKey(msg)
	}

	// 通常ナビゲーション
	s := msg.String()

	switch {
	case msg.Type == tea.KeyEsc:
		switch cs.activePane {
		case paneField:
			cs.activePane = paneCategory
			cs.fieldIndex = 0
			cs.fieldScroll = 0
		case paneDetail:
			cs.activePane = paneField
		default:
			// category ペインで Esc → 閉じる
			return m.tryCloseConfig()
		}
		return m, nil

	case s == "q":
		return m.tryCloseConfig()

	case s == "s":
		return m.saveConfig()

	case s == "/":
		cs.filterMode = true
		cs.filterInput.SetValue("")
		cs.filterInput.Focus()
		return m, nil

	case s == "r":
		return m.resetFieldToDefault()

	case msg.Type == tea.KeyUp || s == "k":
		m.configNavUp()
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		m.configNavDown()
		return m, nil

	case msg.Type == tea.KeyLeft || s == "h":
		m.configNavLeft()
		return m, nil

	case msg.Type == tea.KeyRight || s == "l":
		m.configNavRight()
		return m, nil

	case s == " ":
		return m.configSpaceToggle()

	case isEnterKey(msg):
		return m.configEnter()
	}

	return m, nil
}

// configSpaceToggle は Space で bool フィールドのみトグルする。
func (m Model) configSpaceToggle() (tea.Model, tea.Cmd) {
	cs := m.configScreen
	if cs.activePane != paneField && cs.activePane != paneDetail {
		return m, nil
	}
	field := cs.selectedField()
	if field == nil || field.FieldType != config.FieldTypeBool {
		return m, nil // bool 以外は no-op
	}
	current, _ := field.Current.(bool)
	if err := config.SetFieldValue(cs.cfg, field.Path, !current); err == nil {
		cs.dirty = true
		cs.saveStatus = statusModified
		cs.refreshCategories()
	}
	return m, nil
}

// configNavUp は現在ペインで上移動する。
func (m *Model) configNavUp() {
	cs := m.configScreen
	switch cs.activePane {
	case paneCategory:
		if cs.catIndex > 0 {
			cs.catIndex--
			cs.fieldIndex = 0
			cs.fieldScroll = 0
			cs.filterText = ""
		}
	case paneField:
		fields := cs.filteredFields()
		if cs.fieldIndex > 0 {
			cs.fieldIndex--
		} else if len(fields) > 0 {
			cs.fieldIndex = len(fields) - 1
		}
		cs.ensureFieldVisible(m.fieldPaneVisibleRows())
	}
}

// configNavDown は現在ペインで下移動する。
func (m *Model) configNavDown() {
	cs := m.configScreen
	switch cs.activePane {
	case paneCategory:
		if cs.catIndex < len(cs.categories)-1 {
			cs.catIndex++
			cs.fieldIndex = 0
			cs.fieldScroll = 0
			cs.filterText = ""
		}
	case paneField:
		fields := cs.filteredFields()
		if cs.fieldIndex < len(fields)-1 {
			cs.fieldIndex++
		} else if len(fields) > 0 {
			cs.fieldIndex = 0
		}
		cs.ensureFieldVisible(m.fieldPaneVisibleRows())
	}
}

// fieldPaneVisibleRows は field pane の可視行数を返す。
// フィルタ表示が1行使う場合はそのぶん減らす。
func (m *Model) fieldPaneVisibleRows() int {
	bodyHeight := m.height - 2 // ヘッダー1 + ステータス1
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	cs := m.configScreen
	if cs.filterMode || cs.filterText != "" {
		bodyHeight-- // フィルタ行が最下部に表示される
	}
	return bodyHeight
}

// ensureFieldVisible は fieldIndex が可視範囲内に入るよう fieldScroll を調整する。
func (cs *configScreen) ensureFieldVisible(visibleRows int) {
	if visibleRows <= 0 {
		return
	}
	if cs.fieldIndex < cs.fieldScroll {
		cs.fieldScroll = cs.fieldIndex
	}
	if cs.fieldIndex >= cs.fieldScroll+visibleRows {
		cs.fieldScroll = cs.fieldIndex - visibleRows + 1
	}
	if cs.fieldScroll < 0 {
		cs.fieldScroll = 0
	}
}

// configNavLeft はペインを左に移動する。
func (m *Model) configNavLeft() {
	cs := m.configScreen
	switch cs.activePane {
	case paneField:
		cs.activePane = paneCategory
	case paneDetail:
		cs.activePane = paneField
	}
}

// configNavRight はペインを右に移動する。
func (m *Model) configNavRight() {
	cs := m.configScreen
	switch cs.activePane {
	case paneCategory:
		cs.activePane = paneField
	case paneField:
		_, detailVisible := m.configVisiblePanes()
		if detailVisible {
			cs.activePane = paneDetail
		}
	}
}

// configEnter は Enter を処理する。
func (m Model) configEnter() (tea.Model, tea.Cmd) {
	cs := m.configScreen

	switch cs.activePane {
	case paneCategory:
		cs.activePane = paneField
		cs.fieldIndex = 0
		cs.fieldScroll = 0
		return m, nil

	case paneField, paneDetail:
		field := cs.selectedField()
		if field == nil {
			return m, nil
		}
		return m.startFieldEdit(field)
	}
	return m, nil
}

// startFieldEdit はフィールドの編集を開始する。
func (m Model) startFieldEdit(field *config.ConfigField) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	targetPane := m.configEditTargetPane()

	switch field.FieldType {
	case config.FieldTypeBool:
		// Enter でも bool をトグル
		current, _ := field.Current.(bool)
		if err := config.SetFieldValue(cs.cfg, field.Path, !current); err == nil {
			cs.dirty = true
			cs.saveStatus = statusModified
			cs.refreshCategories()
		}
		return m, nil

	case config.FieldTypeSelect:
		cs.editMode = editSelect
		cs.editSelect = 0
		current, _ := field.Current.(string)
		for i, opt := range field.Options {
			if opt == current {
				cs.editSelect = i
				break
			}
		}
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeString:
		cs.editMode = editInput
		current, _ := field.Current.(string)
		cs.editInput.SetValue(current)
		cs.editInput.Focus()
		cs.editInput.CursorEnd()
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeInt:
		cs.editMode = editInput
		current := 0
		switch v := field.Current.(type) {
		case int:
			current = v
		case int64:
			current = int(v)
		case float64:
			current = int(v)
		}
		cs.editInput.SetValue(strconv.Itoa(current))
		cs.editInput.Focus()
		cs.editInput.CursorEnd()
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeFloat:
		cs.editMode = editInput
		current := 0.0
		switch v := field.Current.(type) {
		case float64:
			current = v
		case float32:
			current = float64(v)
		case int:
			current = float64(v)
		}
		cs.editInput.SetValue(fmt.Sprintf("%g", current))
		cs.editInput.Focus()
		cs.editInput.CursorEnd()
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeStringSlice:
		cs.editMode = editSlice
		if slice, ok := field.Current.([]string); ok {
			cs.editSliceItems = make([]string, len(slice))
			copy(cs.editSliceItems, slice)
		} else {
			cs.editSliceItems = nil
		}
		cs.editSliceIndex = 0
		cs.editSliceAdding = false
		cs.editSliceEditing = false
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeStructMap:
		cs.editMode = editStructMap
		cs.editStructKeys = nil
		val, _ := config.GetFieldValue(cs.cfg, field.Path)
		switch v := val.(type) {
		case map[string]config.ProviderModelConfig:
			for k := range v {
				cs.editStructKeys = append(cs.editStructKeys, k)
			}
		case map[string]config.LSPServerConfig:
			for k := range v {
				cs.editStructKeys = append(cs.editStructKeys, k)
			}
		}
		sort.Strings(cs.editStructKeys)
		cs.editStructIndex = 0
		cs.editStructAdding = false
		cs.activePane = targetPane
		return m, nil
	}

	return m, nil
}

// handleConfigEditKey は編集モード中のキー処理。
func (m Model) handleConfigEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	switch cs.editMode {
	case editSelect:
		return m.handleSelectEdit(msg)
	case editInput:
		return m.handleInputEdit(msg)
	case editSlice:
		return m.handleSliceEdit(msg)
	case editStructMap:
		return m.handleStructMapEdit(msg)
	}
	return m, nil
}

// handleConfigCtrlC は config screen 上の Ctrl+C を処理する。
// processing 中の Ctrl+C は confirmQuit 表示中でも request cancel を優先する。
func (m Model) handleConfigCtrlC() (tea.Model, tea.Cmd) {
	cs := m.configScreen
	if cs == nil {
		return m.handleCtrlC()
	}
	// processing 中は confirmQuit の有無に関係なく request cancel を優先する
	if m.agent.IsProcessing() {
		return m.handleCtrlC()
	}
	if cs.confirmQuit {
		return m, nil
	}
	if cs.dirty {
		return m.tryCloseConfig()
	}
	return m.handleCtrlC()
}

// handleSelectEdit は select 型の編集キー処理。
func (m Model) handleSelectEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return m, nil
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editSelect > 0 {
			cs.editSelect--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editSelect < len(field.Options)-1 {
			cs.editSelect++
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editSelect >= 0 && cs.editSelect < len(field.Options) {
			newVal := field.Options[cs.editSelect]
			if err := config.SetFieldValue(cs.cfg, field.Path, newVal); err == nil {
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		cs.editMode = editNone
		return m, nil
	}
	return m, nil
}

// handleInputEdit は string/int/float 型の入力キー処理。
func (m Model) handleInputEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return m, nil
	}

	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		cs.editInput.Blur()
		return m, nil

	case isEnterKey(msg):
		raw := cs.editInput.Value()
		var newVal interface{}
		var err error

		switch field.FieldType {
		case config.FieldTypeString:
			newVal = raw
		case config.FieldTypeInt:
			v, e := strconv.Atoi(raw)
			if e != nil {
				return m, nil // 不正入力は無視
			}
			newVal = v
		case config.FieldTypeFloat:
			v, e := strconv.ParseFloat(raw, 64)
			if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return m, nil
			}
			if field.Path == "project_map.context_ratio" &&
				(v < config.ProjectMapContextRatioMin || v > config.ProjectMapContextRatioMax) {
				return m, nil
			}
			newVal = v
		}

		if newVal != nil {
			err = config.SetFieldValue(cs.cfg, field.Path, newVal)
			if err == nil {
				// default_model 変更時は編集中 config の default provider override も同期する。
				if field.Path == "default_model" {
					m.syncEditedProviderDefaultModel()
				}
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		cs.editMode = editNone
		cs.editInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editInput, cmd = cs.editInput.Update(msg)
		return m, cmd
	}
}

// handleSliceEdit は []string 型のサブビューキー処理。
func (m Model) handleSliceEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// 追加入力中
	if cs.editSliceAdding || cs.editSliceEditing {
		return m.handleSliceInputKey(msg)
	}

	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return m, nil
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		// []string の変更を確定して戻る
		if err := config.SetFieldValue(cs.cfg, field.Path, cs.editSliceItems); err == nil {
			if !sliceEqual(cs.editSliceItems, field.Current) {
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		cs.editMode = editNone
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editSliceIndex > 0 {
			cs.editSliceIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editSliceIndex < len(cs.editSliceItems)-1 {
			cs.editSliceIndex++
		}
		return m, nil

	case s == "a":
		cs.editSliceAdding = true
		cs.editSliceInput.SetValue("")
		cs.editSliceInput.Focus()
		return m, nil

	case s == "d":
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceItems = append(cs.editSliceItems[:cs.editSliceIndex], cs.editSliceItems[cs.editSliceIndex+1:]...)
			if cs.editSliceIndex >= len(cs.editSliceItems) && cs.editSliceIndex > 0 {
				cs.editSliceIndex--
			}
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceEditing = true
			cs.editSliceInput.SetValue(cs.editSliceItems[cs.editSliceIndex])
			cs.editSliceInput.Focus()
			cs.editSliceInput.CursorEnd()
		}
		return m, nil
	}
	return m, nil
}

// handleSliceInputKey は []string の入力モードキー処理。
func (m Model) handleSliceInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editSliceAdding = false
		cs.editSliceEditing = false
		cs.editSliceInput.Blur()
		return m, nil

	case isEnterKey(msg):
		val := strings.TrimSpace(cs.editSliceInput.Value())
		if val != "" {
			if cs.editSliceAdding {
				cs.editSliceItems = append(cs.editSliceItems, val)
				cs.editSliceIndex = len(cs.editSliceItems) - 1
			} else if cs.editSliceEditing && cs.editSliceIndex < len(cs.editSliceItems) {
				cs.editSliceItems[cs.editSliceIndex] = val
			}
		}
		cs.editSliceAdding = false
		cs.editSliceEditing = false
		cs.editSliceInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editSliceInput, cmd = cs.editSliceInput.Update(msg)
		return m, cmd
	}
}

// handleStructMapEdit は structmap 型のサブビューキー処理。
func (m Model) handleStructMapEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// entry value 編集中
	if cs.editEntryActive {
		return m.handleStructEntryEdit(msg)
	}

	// 追加入力中
	if cs.editStructAdding {
		return m.handleStructMapAddKey(msg)
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editStructIndex > 0 {
			cs.editStructIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editStructIndex < len(cs.editStructKeys)-1 {
			cs.editStructIndex++
		}
		return m, nil

	case s == "d":
		// structmap のキー削除
		if cs.editStructIndex >= 0 && cs.editStructIndex < len(cs.editStructKeys) {
			field := cs.selectedField()
			if field != nil {
				key := cs.editStructKeys[cs.editStructIndex]
				m.deleteStructMapKey(field.Path, key)
				cs.editStructKeys = append(cs.editStructKeys[:cs.editStructIndex], cs.editStructKeys[cs.editStructIndex+1:]...)
				if cs.editStructIndex >= len(cs.editStructKeys) && cs.editStructIndex > 0 {
					cs.editStructIndex--
				}
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		return m, nil

	case s == "a":
		cs.editStructAdding = true
		cs.editStructInput.SetValue("")
		cs.editStructInput.Focus()
		return m, nil

	case isEnterKey(msg):
		// 選択中 key の entry 編集に入る
		if cs.editStructIndex >= 0 && cs.editStructIndex < len(cs.editStructKeys) {
			field := cs.selectedField()
			if field != nil {
				key := cs.editStructKeys[cs.editStructIndex]
				fields := cs.loadEntryFields(field.Path, key)
				if len(fields) > 0 {
					cs.editEntryActive = true
					cs.editEntryKey = key
					cs.editEntryFields = fields
					cs.editEntryIndex = 0
					cs.editEntryFieldEdit = ""
				}
			}
		}
		return m, nil
	}
	return m, nil
}

// handleStructEntryEdit は structmap entry の value 編集キー処理。
func (m Model) handleStructEntryEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// 個別フィールド編集中
	if cs.editEntryFieldEdit != "" {
		return m.handleStructEntryFieldEdit(msg)
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		// entry 編集を抜けて key list に戻る
		cs.editEntryActive = false
		cs.editEntryFields = nil
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editEntryIndex > 0 {
			cs.editEntryIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editEntryIndex < len(cs.editEntryFields)-1 {
			cs.editEntryIndex++
		}
		return m, nil

	case s == " ":
		// bool フィールドの即トグル
		if cs.editEntryIndex < len(cs.editEntryFields) {
			ef := &cs.editEntryFields[cs.editEntryIndex]
			if ef.Type == "bool" {
				cur, _ := ef.Value.(bool)
				ef.Value = !cur
				m.applyEntryFieldAndMark(ef)
			}
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editEntryIndex >= 0 && cs.editEntryIndex < len(cs.editEntryFields) {
			ef := &cs.editEntryFields[cs.editEntryIndex]
			switch ef.Type {
			case "bool":
				cur, _ := ef.Value.(bool)
				ef.Value = !cur
				m.applyEntryFieldAndMark(ef)
			case "string":
				cs.editEntryFieldEdit = "input"
				val, _ := ef.Value.(string)
				cs.editInput.SetValue(val)
				cs.editInput.Focus()
				cs.editInput.CursorEnd()
			case "int":
				cs.editEntryFieldEdit = "input"
				v := 0
				switch n := ef.Value.(type) {
				case int:
					v = n
				}
				cs.editInput.SetValue(strconv.Itoa(v))
				cs.editInput.Focus()
				cs.editInput.CursorEnd()
			case "[]string":
				cs.editEntryFieldEdit = "slice"
				if s, ok := ef.Value.([]string); ok {
					cs.editSliceItems = make([]string, len(s))
					copy(cs.editSliceItems, s)
				} else {
					cs.editSliceItems = nil
				}
				cs.editSliceIndex = 0
				cs.editSliceAdding = false
				cs.editSliceEditing = false
			}
		}
		return m, nil
	}
	return m, nil
}

// handleStructEntryFieldEdit は entry 内の個別フィールド編集キー処理。
func (m Model) handleStructEntryFieldEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	switch cs.editEntryFieldEdit {
	case "input":
		return m.handleEntryInputEdit(msg)
	case "slice":
		return m.handleEntrySliceEdit(msg)
	}
	return m, nil
}

// handleEntryInputEdit は entry 内の string/int フィールド入力処理。
func (m Model) handleEntryInputEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editEntryFieldEdit = ""
		cs.editInput.Blur()
		return m, nil

	case isEnterKey(msg):
		ef := &cs.editEntryFields[cs.editEntryIndex]
		raw := cs.editInput.Value()
		switch ef.Type {
		case "string":
			ef.Value = raw
		case "int":
			v, err := strconv.Atoi(raw)
			if err != nil {
				return m, nil
			}
			ef.Value = v
		}
		m.applyEntryFieldAndMark(ef)
		cs.editEntryFieldEdit = ""
		cs.editInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editInput, cmd = cs.editInput.Update(msg)
		return m, cmd
	}
}

// handleEntrySliceEdit は entry 内の []string フィールド編集処理。
// 既存の slice editor state を再利用する。
func (m Model) handleEntrySliceEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// 追加/編集入力中
	if cs.editSliceAdding || cs.editSliceEditing {
		return m.handleSliceInputKey(msg)
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		// slice 変更を entry field に書き戻す
		ef := &cs.editEntryFields[cs.editEntryIndex]
		items := make([]string, len(cs.editSliceItems))
		copy(items, cs.editSliceItems)
		ef.Value = items
		m.applyEntryFieldAndMark(ef)
		cs.editEntryFieldEdit = ""
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editSliceIndex > 0 {
			cs.editSliceIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editSliceIndex < len(cs.editSliceItems)-1 {
			cs.editSliceIndex++
		}
		return m, nil

	case s == "a":
		cs.editSliceAdding = true
		cs.editSliceInput.SetValue("")
		cs.editSliceInput.Focus()
		return m, nil

	case s == "d":
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceItems = append(cs.editSliceItems[:cs.editSliceIndex], cs.editSliceItems[cs.editSliceIndex+1:]...)
			if cs.editSliceIndex >= len(cs.editSliceItems) && cs.editSliceIndex > 0 {
				cs.editSliceIndex--
			}
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceEditing = true
			cs.editSliceInput.SetValue(cs.editSliceItems[cs.editSliceIndex])
			cs.editSliceInput.Focus()
			cs.editSliceInput.CursorEnd()
		}
		return m, nil
	}
	return m, nil
}

// applyEntryFieldAndMark は entry field の変更を Config に書き戻し dirty をマークする。
func (m *Model) applyEntryFieldAndMark(ef *structEntryField) {
	cs := m.configScreen
	field := cs.selectedField()
	if field == nil {
		return
	}
	cs.applyEntryField(field.Path, cs.editEntryKey, *ef)
	cs.dirty = true
	cs.saveStatus = statusModified
	cs.refreshCategories()
}

// handleStructMapAddKey は structmap の新規キー追加入力処理。
func (m Model) handleStructMapAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editStructAdding = false
		cs.editStructInput.Blur()
		return m, nil

	case isEnterKey(msg):
		key := strings.TrimSpace(cs.editStructInput.Value())
		if key != "" {
			field := cs.selectedField()
			if field != nil && m.addStructMapKey(field.Path, key) {
				cs.editStructKeys = append(cs.editStructKeys, key)
				sort.Strings(cs.editStructKeys)
				cs.editStructIndex = sort.SearchStrings(cs.editStructKeys, key)
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
			// 既存 key の場合は UI / dirty を変更しない
		}
		cs.editStructAdding = false
		cs.editStructInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editStructInput, cmd = cs.editStructInput.Update(msg)
		return m, cmd
	}
}

// deleteStructMapKey は structmap のキーを削除する。
func (m *Model) deleteStructMapKey(path, key string) {
	cs := m.configScreen
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		cs.cfg.DeleteProviderModelConfig(key)
	case map[string]config.LSPServerConfig:
		delete(v, key)
	}
}

// addStructMapKey は structmap に空のキーを追加する。
// 既存キーの場合は false を返し、新規追加時は true を返す。
func (m *Model) addStructMapKey(path, key string) bool {
	cs := m.configScreen
	m.ensureStructMapInitialized(path)
	if path == "provider_models" {
		providerModels := cs.cfg.ProviderModelsForEdit()
		if providerModels == nil {
			providerModels = map[string]config.ProviderModelConfig{}
		}
		if _, ok := providerModels[key]; ok {
			return false
		}
		providerModels[key] = config.ProviderModelConfig{}
		cs.cfg.SetProviderModelsForEdit(providerModels)
		return true
	}
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.LSPServerConfig:
		if _, ok := v[key]; ok {
			return false
		}
		v[key] = config.LSPServerConfig{}
		return true
	}
	return false
}

// ensureStructMapInitialized は TUI が直接 mutate する struct map を必要に応じて初期化する。
func (m *Model) ensureStructMapInitialized(path string) {
	cs := m.configScreen
	if cs == nil {
		return
	}

	switch path {
	case "lsp.servers":
		if cs.cfg.LSP.Servers == nil {
			cs.cfg.LSP.Servers = make(map[string]config.LSPServerConfig)
		}
	}
}

// handleConfigFilterKey はフィルタモードのキー処理。
func (m Model) handleConfigFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	switch {
	case msg.Type == tea.KeyEsc:
		cs.filterMode = false
		cs.filterText = ""
		cs.filterInput.Blur()
		cs.fieldIndex = 0
		cs.fieldScroll = 0
		return m, nil

	case isEnterKey(msg):
		cs.filterText = cs.filterInput.Value()
		cs.filterMode = false
		cs.filterInput.Blur()
		cs.fieldIndex = 0
		cs.fieldScroll = 0
		return m, nil

	default:
		var cmd tea.Cmd
		cs.filterInput, cmd = cs.filterInput.Update(msg)
		cs.filterText = cs.filterInput.Value()
		cs.fieldIndex = 0
		cs.fieldScroll = 0
		return m, cmd
	}
}

// handleConfigConfirmKey は終了確認ダイアログのキー処理。
func (m Model) handleConfigConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	s := msg.String()

	switch {
	case msg.Type == tea.KeyEsc:
		cs.confirmQuit = false
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.confirmIdx > 0 {
			cs.confirmIdx--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.confirmIdx < 2 {
			cs.confirmIdx++
		}
		return m, nil

	case isEnterKey(msg):
		switch cs.confirmIdx {
		case 0: // Save and quit — async 保存して ConfigSavedMsg で閉じる
			cs.confirmQuit = false
			cs.pendingClose = true
			cs.saveStatus = statusSaving
			return m, m.saveConfigCmd()
		case 1: // Discard and quit
			// save が in-flight の場合は discard を許可しない（保存結果が後から適用される問題を防ぐ）
			if cs.saveStatus == statusSaving {
				return m, nil
			}
			cs.confirmQuit = false
			return m.closeConfigScreen()
		case 2: // Cancel
			cs.confirmQuit = false
			return m, nil
		}
	}
	return m, nil
}

// tryCloseConfig は config screen を閉じようとする。未保存変更があれば確認する。
func (m Model) tryCloseConfig() (tea.Model, tea.Cmd) {
	cs := m.configScreen
	if cs.dirty {
		cs.confirmQuit = true
		cs.confirmIdx = 0
		return m, nil
	}
	return m.closeConfigScreen()
}

// saveConfig は設定を保存する。
func (m Model) saveConfig() (tea.Model, tea.Cmd) {
	cs := m.configScreen
	cs.saveStatus = statusSaving
	return m, m.saveConfigCmd()
}

// saveConfigCmd は非同期で設定を保存する tea.Cmd を返す。
// Cmd クロージャには cfg の snapshot を渡し、非同期実行中に元 cfg が変更されても保存対象が変わらないようにする。
func (m Model) saveConfigCmd() tea.Cmd {
	snapshot := config.CloneConfig(m.configScreen.cfg)
	agent := m.agent
	return func() tea.Msg {
		err := agent.SaveAndSyncConfig(snapshot)
		return ConfigSavedMsg{Error: err, Snapshot: snapshot}
	}
}

// resetFieldToDefault は現在フィールドをデフォルト値に戻す。
func (m Model) resetFieldToDefault() (tea.Model, tea.Cmd) {
	cs := m.configScreen
	field := cs.selectedField()
	if field == nil {
		return m, nil
	}

	if field.Default == nil {
		return m, nil
	}

	if err := config.SetFieldValue(cs.cfg, field.Path, field.Default); err == nil {
		if field.Path == "default_model" {
			m.syncEditedProviderDefaultModel()
		}
		cs.dirty = true
		cs.saveStatus = statusModified
		cs.refreshCategories()
	}
	return m, nil
}

// syncEditedProviderDefaultModel は global default_model を選択中 provider override に同期する。
func (m Model) syncEditedProviderDefaultModel() {
	cs := m.configScreen
	if cs == nil {
		return
	}

	provName := m.defaultModelSyncProvider()
	if provName == "" {
		return
	}
	cs.cfg.SyncProviderDefaultModel(provName, cs.cfg.DefaultModel)
}

// defaultModelSyncProvider は global default_model を同期する provider_models key を返す。
// default_provider が別 runtime へ切り替わっていない限り、現在セッションの exact alias owner を優先する。
func (m Model) defaultModelSyncProvider() string {
	cs := m.configScreen
	if cs == nil {
		return ""
	}
	if cs.cfg == nil {
		return ""
	}
	return cs.cfg.DefaultModelSyncProviderKey(m.agent.GetProviderConfigKey(), cs.initialDefaultProvider)
}

// sliceEqual は []string の等値比較。
func sliceEqual(a []string, b interface{}) bool {
	bs, ok := b.([]string)
	if !ok {
		return false
	}
	if len(a) != len(bs) {
		return false
	}
	for i := range a {
		if a[i] != bs[i] {
			return false
		}
	}
	return true
}
