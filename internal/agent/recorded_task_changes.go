package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
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

	seen := make(map[string]bool, len(taskChanges))
	files := make([]string, 0, len(taskChanges))
	for _, change := range taskChanges {
		files = collectRecordedChangedFiles(files, seen, change)
	}

	return recordedTaskChangeSnapshot{
		files:               files,
		progressFingerprint: fingerprintRecordedTaskChanges(taskChanges),
	}
}

func (a *Agent) recordedTaskChangesSinceTaskStart() []tools.FileChange {
	if a == nil || a.taskChangeOffset >= len(a.changeStack) {
		return nil
	}
	return a.changeStack[a.taskChangeOffset:]
}

func collectRecordedChangedFiles(files []string, seen map[string]bool, change tools.FileChange) []string {
	if len(change.Details) > 0 {
		for _, detail := range change.Details {
			files = appendRecordedChangedFile(files, seen, detail.FilePath)
		}
		return files
	}
	return appendRecordedChangedFile(files, seen, change.FilePath)
}

func appendRecordedChangedFile(files []string, seen map[string]bool, path string) []string {
	if path == "" || seen[path] {
		return files
	}
	seen[path] = true
	return append(files, path)
}

func fingerprintRecordedTaskChanges(changes []tools.FileChange) string {
	if len(changes) == 0 {
		return ""
	}

	hasher := sha256.New()
	for idx, change := range changes {
		writeChangeFingerprint(hasher, idx, change)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func writeChangeFingerprint(hasher hash.Hash, idx int, change tools.FileChange) {
	_, _ = hasher.Write([]byte(strconv.Itoa(idx)))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write([]byte(change.Tool))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write([]byte(change.FilePath))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write([]byte(change.Description))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write([]byte(strconv.Itoa(change.LinesAdded)))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write([]byte(strconv.Itoa(change.LinesRemoved)))
	_, _ = hasher.Write([]byte{'\n'})
	for _, detail := range change.Details {
		_, _ = hasher.Write([]byte(detail.FilePath))
		_, _ = hasher.Write([]byte{'\n'})
		_, _ = hasher.Write([]byte(detail.Action))
		_, _ = hasher.Write([]byte{'\n'})
		_, _ = hasher.Write([]byte(strconv.Itoa(detail.LinesAdded)))
		_, _ = hasher.Write([]byte{'\n'})
		_, _ = hasher.Write([]byte(strconv.Itoa(detail.LinesRemoved)))
		_, _ = hasher.Write([]byte{'\n'})
	}
}
