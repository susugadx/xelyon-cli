package history

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrNoResumeSessions は resume 対象の session が見つからないことを表す。
var ErrNoResumeSessions = errors.New("no sessions found")

// ResumeListOptions は resume 候補の絞り込み条件です。
type ResumeListOptions struct {
	WorkingDir       string
	All              bool
	ExcludeSessionID string
}

// ListResumeSessions は resume picker に出す保存済み session を新しい順で返す。
func (st *Storage) ListResumeSessions(opts ResumeListOptions) ([]SessionMetadata, error) {
	sessions, err := st.ListSessions()
	if err != nil {
		return nil, err
	}

	workingDir := normalizeSessionWorkingDir(opts.WorkingDir)
	if workingDir == "" {
		workingDir = currentWorkingDirForSession()
	}

	filtered := make([]SessionMetadata, 0, len(sessions))
	for _, session := range sessions {
		if resumeCandidateExcluded(session, opts.ExcludeSessionID) {
			continue
		}
		if !resumeCandidateHasContent(session) {
			continue
		}
		if opts.All || resumeCandidateMatchesWorkingDir(session, workingDir) {
			filtered = append(filtered, session)
		}
	}
	return filtered, nil
}

// GetLastResumeSession は resume scope 内の最新 session ID を返す。
func (st *Storage) GetLastResumeSession(opts ResumeListOptions) (string, error) {
	sessions, err := st.ListResumeSessions(opts)
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", ErrNoResumeSessions
	}
	return sessions[0].ID, nil
}

func resumeCandidateHasContent(session SessionMetadata) bool {
	return session.MessageCount > 0 || session.CompactedItemCount > 0 || session.IsCompactedMode
}

func resumeCandidateExcluded(session SessionMetadata, excludeSessionID string) bool {
	excludeSessionID = strings.TrimSpace(excludeSessionID)
	return excludeSessionID != "" && strings.TrimSpace(session.ID) == excludeSessionID
}

func resumeCandidateMatchesWorkingDir(session SessionMetadata, workingDir string) bool {
	sessionWorkingDir := normalizeSessionWorkingDir(session.WorkingDir)
	if sessionWorkingDir == "" {
		return true
	}
	if workingDir == "" {
		return true
	}
	return sessionWorkingDir == workingDir
}

func normalizeSessionWorkingDir(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}
