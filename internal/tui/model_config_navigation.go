package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

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
