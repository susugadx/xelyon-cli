package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

// openConfigScreen は config screen を開く。
func (m Model) openConfigScreen() (tea.Model, tea.Cmd) {
	cfg, err := m.configAgent.LoadConfigForEdit()
	if err != nil {
		m.appendSystemInfo("Failed to load config: " + err.Error())
		return m, nil
	}
	m.activateModalScreen(screenConfig)
	m.configScreen = newConfigScreen(cfg)
	return m, nil
}

// closeConfigScreen は config screen を閉じて chat に戻る。
func (m Model) closeConfigScreen() (tea.Model, tea.Cmd) {
	m.configScreen = nil
	m.deactivateModalScreen(false)
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
		cs.normalizePaneState(m.configLayout())
		return m, nil

	case ConfigSavedMsg:
		shouldClose, saved := cs.handleSaveResult(msg)
		if shouldClose {
			return m.closeConfigScreen()
		}
		if saved {
			m.refreshStatusLine()
		}
		return m, nil

	case tea.KeyMsg:
		action, cmd := cs.handleKey(msg, m.configLayout(), m.conversation.IsProcessing(), m.configAgent.GetProviderConfigKey())
		switch action {
		case configCommandDelegateCtrlC:
			return m.handleCtrlC()
		case configCommandClose:
			return m.closeConfigScreen()
		case configCommandSave:
			return m.beginConfigSave(false)
		case configCommandSaveAndClose:
			return m.beginConfigSave(true)
		default:
			return m, cmd
		}

	default:
		// config screen が関知しないメッセージは chat 側更新に通す。
		return m.forwardMessageToChatFromModal(msg, screenConfig)
	}
}
func (m Model) configLayout() configLayout {
	return configscreen.NewLayout(m.width, m.height)
}

func (m Model) beginConfigSave(closeOnSuccess bool) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	if cs == nil {
		return m, nil
	}
	cs.confirmQuit = false
	cs.pendingClose = cs.pendingClose || closeOnSuccess
	cs.saveStatus = statusSaving
	return m, m.saveConfigCmd()
}

// saveConfigCmd は非同期で設定を保存する tea.Cmd を返す。
// Cmd クロージャには cfg の snapshot を渡し、非同期実行中に元 cfg が変更されても保存対象が変わらないようにする。
func (m Model) saveConfigCmd() tea.Cmd {
	snapshot := config.CloneConfig(m.configScreen.cfg)
	configAgent := m.configAgent
	return func() tea.Msg {
		if err := geminiFunctionCallingConfigSaveError(snapshot); err != nil {
			return ConfigSavedMsg{Error: err, Snapshot: snapshot}
		}
		err := configAgent.SaveAndSyncConfig(snapshot)
		return ConfigSavedMsg{Error: err, Snapshot: snapshot}
	}
}

// syncEditedProviderDefaultModel は global default_model を選択中 provider override に同期する。
func (m Model) syncEditedProviderDefaultModel() {
	if m.configScreen == nil {
		return
	}
	m.configScreen.syncEditedProviderDefaultModel(m.configAgent.GetProviderConfigKey())
}

// defaultModelSyncProvider は global default_model を同期する provider_models key を返す。
// default_provider が別 runtime へ切り替わっていない限り、現在セッションの exact alias owner を優先する。
func (m Model) defaultModelSyncProvider() string {
	if m.configScreen == nil {
		return ""
	}
	return m.configScreen.defaultModelSyncProvider(m.configAgent.GetProviderConfigKey())
}

// addStructMapKey は structmap に空のキーを追加する。
// 既存キーの場合は false を返し、新規追加時は true を返す。
func (m *Model) addStructMapKey(path, key string) bool {
	if m.configScreen == nil {
		return false
	}
	return m.configScreen.addStructMapKey(path, key)
}
