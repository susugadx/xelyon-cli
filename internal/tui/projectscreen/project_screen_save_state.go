package projectscreen

import (
	"reflect"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// SaveResult は /project 保存コマンドの結果を表す。
type SaveResult struct {
	Error    error
	Snapshot *config.ProjectConfig
	ScreenID int
	SaveSeq  int
}

// SaveAction は保存結果を受けた後に root が行う追加処理を表す。
type SaveAction struct {
	ShouldClose bool
	StartQueued bool
}

// PendingSave は root が実行する保存コマンドの入力を表す。
type PendingSave struct {
	ScreenID int
	SaveSeq  int
	Snapshot *config.ProjectConfig
}

// HandleSaveResult は保存完了結果を screen state に反映する。
func (ps *Screen) HandleSaveResult(msg SaveResult) SaveAction {
	if msg.ScreenID != 0 && msg.ScreenID != ps.screenID {
		return SaveAction{}
	}
	if msg.SaveSeq != 0 && msg.SaveSeq != ps.saveSeq {
		return SaveAction{}
	}
	ps.saveInFlight = false

	if msg.Error != nil {
		ps.saveStatus = projectStatusFailed
		ps.saveError = msg.Error.Error()
		ps.pendingClose = false
		ps.confirmQuit = false
		ps.saveQueued = false
		return SaveAction{}
	}

	ps.saveError = ""
	if reflect.DeepEqual(ps.pc, msg.Snapshot) {
		shouldClose := ps.pendingClose && ps.editMode == projectEditNone
		ps.markSaved(msg.Snapshot)
		return SaveAction{ShouldClose: shouldClose}
	}

	if ps.saveQueued {
		ps.dirty = true
		ps.saveStatus = projectStatusSaving
		ps.confirmQuit = false
		ps.message = ""
		return SaveAction{StartQueued: true}
	}

	ps.dirty = true
	ps.saveStatus = projectStatusModified
	ps.pendingClose = false
	ps.confirmQuit = false
	ps.message = ""
	return SaveAction{}
}

// BeginSave は保存開始状態に遷移し、root が永続化する snapshot を返す。
func (ps *Screen) BeginSave(closeOnSuccess bool) (PendingSave, bool) {
	if ps == nil || ps.pc == nil {
		return PendingSave{}, false
	}
	ps.confirmQuit = false
	closeIntent := ps.pendingClose || closeOnSuccess
	ps.pendingClose = closeIntent
	if ps.saveInFlight {
		ps.saveQueued = true
		ps.saveStatus = projectStatusSaving
		ps.saveError = ""
		ps.message = "save queued"
		return PendingSave{}, false
	}
	ps.saveSeq++
	ps.saveInFlight = true
	ps.saveQueued = false
	ps.saveStatus = projectStatusSaving
	ps.saveError = ""
	ps.message = ""
	return PendingSave{
		ScreenID: ps.screenID,
		SaveSeq:  ps.saveSeq,
		Snapshot: config.CloneProjectConfig(ps.pc),
	}, true
}

func (ps *Screen) markSaved(snapshot *config.ProjectConfig) {
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
