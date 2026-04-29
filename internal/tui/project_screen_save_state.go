package tui

import (
	"reflect"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type projectSaveResult struct {
	shouldClose bool
	startQueued bool
}

func (ps *projectScreen) handleSaveResult(msg ProjectSavedMsg) projectSaveResult {
	if msg.ScreenID != 0 && msg.ScreenID != ps.screenID {
		return projectSaveResult{}
	}
	if msg.SaveSeq != 0 && msg.SaveSeq != ps.saveSeq {
		return projectSaveResult{}
	}
	ps.saveInFlight = false

	if msg.Error != nil {
		ps.saveStatus = projectStatusFailed
		ps.saveError = msg.Error.Error()
		ps.pendingClose = false
		ps.confirmQuit = false
		ps.saveQueued = false
		return projectSaveResult{}
	}

	ps.saveError = ""
	if reflect.DeepEqual(ps.pc, msg.Snapshot) {
		shouldClose := ps.pendingClose && ps.editMode == projectEditNone
		ps.markSaved(msg.Snapshot)
		return projectSaveResult{shouldClose: shouldClose}
	}

	if ps.saveQueued {
		ps.dirty = true
		ps.saveStatus = projectStatusSaving
		ps.confirmQuit = false
		ps.message = ""
		return projectSaveResult{startQueued: true}
	}

	ps.dirty = true
	ps.saveStatus = projectStatusModified
	ps.pendingClose = false
	ps.confirmQuit = false
	ps.message = ""
	return projectSaveResult{}
}

func (ps *projectScreen) markSaved(snapshot *config.ProjectConfig) {
	ps.pc = config.CloneProjectConfig(snapshot)
	ps.missing = ps.pc == nil
	ps.savedHasFinalChecks = ps.pc != nil && ps.pc.FinalChecks != nil
	ps.tuiCreatedFinalChecks = false
	ps.dirty = false
	ps.saveStatus = projectStatusSaved
	ps.saveError = ""
	ps.pendingClose = false
	ps.confirmQuit = false
	ps.saveInFlight = false
	ps.saveQueued = false
	ps.message = "saved"
}
