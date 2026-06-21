package configscreen

import "reflect"

// statusText は保存状態のテキストを返す。
func (cs *Screen) statusText() string {
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

func (cs *Screen) normalizePaneState(layout configLayout) {
	if cs == nil {
		return
	}
	if cs.activePane == paneDetail && !layout.DetailVisible() {
		cs.activePane = paneField
	}
}

// NormalizePaneState は現在レイアウトで表示できない pane 選択を補正する。
func (cs *Screen) NormalizePaneState(layout Layout) {
	cs.normalizePaneState(layout)
}

// BeginSave は保存開始時の screen 状態を反映する。
func (cs *Screen) BeginSave(closeOnSuccess bool) {
	if cs == nil {
		return
	}
	cs.confirmQuit = false
	cs.pendingClose = cs.pendingClose || closeOnSuccess
	cs.saveStatus = statusSaving
}

// HandleSaveResult は保存完了 Msg を screen 状態に反映する。
func (cs *Screen) HandleSaveResult(msg SavedMsg) (shouldClose bool, saved bool) {
	if msg.Error != nil {
		cs.saveStatus = statusFailed
		cs.saveError = msg.Error.Error()
		cs.pendingClose = false
		return false, false
	}

	cs.saveError = ""
	cs.refreshCategories()
	if reflect.DeepEqual(cs.cfg, msg.Snapshot) {
		cs.dirty = false
		cs.saveStatus = statusSaved
		shouldClose = cs.pendingClose
		cs.pendingClose = false
		return shouldClose, true
	}

	cs.dirty = true
	cs.saveStatus = statusModified
	cs.pendingClose = false
	return false, false
}
