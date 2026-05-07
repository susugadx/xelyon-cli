package tui

// renderInputDock は入力欄部分（パディング行+入力行+パディング行）を構築する。
func (m *Model) renderInputDock() string {
	return joinChromeLines(m.renderInputDockLines())
}

// renderStatusBar はステータスバー行を構築する。
func (m *Model) renderStatusBar() string {
	return m.buildStatusBarLine()
}

// rebuildChrome は入力欄+ステータスバーを再構築する。
// Update() 内で chromeDirty 時のみ呼ばれる（View() は値レシーバーなので書き込み不可）。
func (m *Model) rebuildChrome() {
	m.chromeCache = m.renderInputDock() + "\n" + m.renderStatusBar()
}

// View は bubbletea の View を実装する。
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "Initializing..."
	}
	base := m.baseView()
	if m.prompt != nil {
		return m.renderPromptOverlay(base)
	}
	if m.providerPicker != nil {
		return m.renderProviderPickerOverlay(base)
	}
	return base
}

func (m Model) baseView() string {
	if m.screen == screenConfig {
		return m.configView()
	}
	if m.screen == screenReview {
		return m.reviewView()
	}
	if m.screen == screenProject {
		return m.projectView()
	}
	return m.viewportView() + "\n" + m.chromeCache
}
