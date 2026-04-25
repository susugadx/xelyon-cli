package tui

import "reflect"

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

func (cs *configScreen) normalizePaneState(layout configLayout) {
	if cs == nil {
		return
	}
	if cs.activePane == paneDetail && !layout.DetailVisible() {
		cs.activePane = paneField
	}
}

func (cs *configScreen) handleSaveResult(msg ConfigSavedMsg) (shouldClose bool, saved bool) {
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
