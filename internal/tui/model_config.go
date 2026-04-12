package tui

import (
	"reflect"
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
