package agent

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/turnsupport"
)

type recordedTaskChangeSnapshot struct {
	files               []string
	progressFingerprint string
}

// recordedTaskChangedFiles は現在タスクで記録された変更ファイル一覧を返す。
// changeStack に記録された変更だけを source of truth とする。
func (a *Agent) recordedTaskChangedFiles() []string {
	return a.recordedTaskChangeSnapshot().files
}

func (a *Agent) recordedTaskChangeFingerprint() string {
	return a.recordedTaskChangeSnapshot().progressFingerprint
}

func (a *Agent) recordedTaskChangeSnapshot() recordedTaskChangeSnapshot {
	taskChanges := a.recordedTaskChangesSinceTaskStart()
	if len(taskChanges) == 0 {
		return recordedTaskChangeSnapshot{}
	}

	snapshot := turnsupport.SnapshotFileChanges(taskChanges)
	return recordedTaskChangeSnapshot{
		files:               snapshot.Files,
		progressFingerprint: snapshot.ProgressFingerprint,
	}
}

func (a *Agent) recordedTaskChangesSinceTaskStart() []tools.FileChange {
	if a == nil || a.taskChangeOffset >= len(a.changeStack) {
		return nil
	}
	return a.changeStack[a.taskChangeOffset:]
}
