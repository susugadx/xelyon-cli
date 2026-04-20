package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// turnMutationState は normal turn 内で観測した FileChange イベントを保持する。
// final checks の起動判定と進捗判定は changeStack 逆算ではなく、この turn-local state を source of truth にする。
type turnMutationState struct {
	mutationCount int
	changedFiles  []string
	seenFiles     map[string]bool

	fingerprintHasher hash.Hash
	fingerprintIndex  int
}

type turnMutationSnapshot struct {
	mutationCount       int
	files               []string
	progressFingerprint string
}

func newTurnMutationState() turnMutationState {
	state := turnMutationState{}
	state.reset()
	return state
}

func (s *turnMutationState) reset() {
	if s == nil {
		return
	}
	s.mutationCount = 0
	s.changedFiles = nil
	s.seenFiles = make(map[string]bool)
	s.fingerprintHasher = sha256.New()
	s.fingerprintIndex = 0
}

func (s *turnMutationState) ensureInitialized() {
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

func (s *turnMutationState) recordFileChange(change tools.FileChange) {
	if s == nil {
		return
	}
	s.ensureInitialized()

	s.mutationCount++
	s.changedFiles = collectRecordedChangedFiles(s.changedFiles, s.seenFiles, change)
	writeChangeFingerprint(s.fingerprintHasher, s.fingerprintIndex, change)
	s.fingerprintIndex++
}

func (s *turnMutationState) hasMutations() bool {
	return s != nil && s.mutationCount > 0
}

func (s *turnMutationState) snapshot() turnMutationSnapshot {
	if s == nil || !s.hasMutations() {
		return turnMutationSnapshot{}
	}
	files := append([]string(nil), s.changedFiles...)
	return turnMutationSnapshot{
		mutationCount:       s.mutationCount,
		files:               files,
		progressFingerprint: hex.EncodeToString(s.fingerprintHasher.Sum(nil)),
	}
}
