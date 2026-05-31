package turnsupport

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// MutationState は normal turn 内で観測した FileChange イベントを保持する。
//
// final checks の起動判定と進捗判定は changeStack 逆算ではなく、
// この turn-local state を source of truth にする。
type MutationState struct {
	mutationCount int
	changedFiles  []string
	seenFiles     map[string]bool

	fingerprintHasher hash.Hash
	fingerprintIndex  int
}

// MutationSnapshot は normal turn 内の mutation 状態の読み取り専用 snapshot。
type MutationSnapshot struct {
	MutationCount       int
	Files               []string
	ProgressFingerprint string
}

// ChangeSnapshot は FileChange 一覧から作った変更ファイルと進捗 fingerprint。
type ChangeSnapshot struct {
	Files               []string
	ProgressFingerprint string
}

// NewMutationState は初期化済みの MutationState を返す。
func NewMutationState() MutationState {
	state := MutationState{}
	state.reset()
	return state
}

func (s *MutationState) reset() {
	if s == nil {
		return
	}
	s.mutationCount = 0
	s.changedFiles = nil
	s.seenFiles = make(map[string]bool)
	s.fingerprintHasher = sha256.New()
	s.fingerprintIndex = 0
}

func (s *MutationState) ensureInitialized() {
	if s == nil {
		return
	}
	if s.seenFiles == nil {
		s.seenFiles = make(map[string]bool)
	}
	if s.fingerprintHasher == nil {
		s.fingerprintHasher = sha256.New()
	}
}

// RecordFileChange は turn 内で観測した FileChange を記録する。
func (s *MutationState) RecordFileChange(change tools.FileChange) {
	if s == nil {
		return
	}
	s.ensureInitialized()

	s.mutationCount++
	s.changedFiles = collectChangedFiles(s.changedFiles, s.seenFiles, change)
	writeChangeFingerprint(s.fingerprintHasher, s.fingerprintIndex, change)
	s.fingerprintIndex++
}

// HasMutations は turn 内で mutation が記録されていれば true を返す。
func (s *MutationState) HasMutations() bool {
	return s != nil && s.mutationCount > 0
}

// Snapshot は現在の mutation 状態を返す。
func (s *MutationState) Snapshot() MutationSnapshot {
	if s == nil || !s.HasMutations() {
		return MutationSnapshot{}
	}
	files := append([]string(nil), s.changedFiles...)
	return MutationSnapshot{
		MutationCount:       s.mutationCount,
		Files:               files,
		ProgressFingerprint: hex.EncodeToString(s.fingerprintHasher.Sum(nil)),
	}
}

// SnapshotFileChanges は FileChange 一覧から変更ファイルと進捗 fingerprint を作る。
func SnapshotFileChanges(changes []tools.FileChange) ChangeSnapshot {
	if len(changes) == 0 {
		return ChangeSnapshot{}
	}

	seen := make(map[string]bool, len(changes))
	files := make([]string, 0, len(changes))
	for _, change := range changes {
		files = collectChangedFiles(files, seen, change)
	}

	return ChangeSnapshot{
		Files:               files,
		ProgressFingerprint: fingerprintFileChanges(changes),
	}
}

func collectChangedFiles(files []string, seen map[string]bool, change tools.FileChange) []string {
	if len(change.Details) > 0 {
		for _, detail := range change.Details {
			files = appendChangedFile(files, seen, detail.FilePath)
		}
		return files
	}
	return appendChangedFile(files, seen, change.FilePath)
}

func appendChangedFile(files []string, seen map[string]bool, path string) []string {
	if path == "" || seen[path] {
		return files
	}
	seen[path] = true
	return append(files, path)
}

func fingerprintFileChanges(changes []tools.FileChange) string {
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
