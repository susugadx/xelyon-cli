package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

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
