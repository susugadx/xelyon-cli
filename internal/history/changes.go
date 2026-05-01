package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	changesDir        = ".xelyon/changes"
	changeFilePrefix  = "changes_"
	changeFileSuffix  = ".jsonl"
	changeFilePattern = changeFilePrefix + "*" + changeFileSuffix
)

var userHomeDirForChanges = os.UserHomeDir

// PersistentChange は永続化される変更履歴
type PersistentChange struct {
	SessionID   string    `json:"session_id"`
	Timestamp   time.Time `json:"timestamp"`
	Tool        string    `json:"tool"`
	FilePath    string    `json:"file_path"`
	Description string    `json:"description"`
}

// ChangeDetail は変更対象ファイルの詳細です。
type ChangeDetail struct {
	FilePath string
}

// ChangeRecordInput は変更履歴永続化の入力です。
type ChangeRecordInput struct {
	FilePath    string
	Details     []ChangeDetail
	Timestamp   time.Time
	Tool        string
	Description string
}

// ChangeStorage は変更履歴のストレージ
type ChangeStorage struct {
	changesPath string
}

// SessionInfo はセッション情報（変更履歴表示用）
type SessionInfo struct {
	SessionID    string
	ChangeCount  int
	FirstChange  time.Time
	LastChange   time.Time
	FilesChanged map[string]int // ファイルパス → 変更回数
}

// NewChangeStorage は新しいChangeStorageを作成
func NewChangeStorage() (*ChangeStorage, error) {
	home, err := userHomeDirForChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	changesPath := filepath.Join(home, changesDir)
	if err := os.MkdirAll(changesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create changes directory: %w", err)
	}

	return &ChangeStorage{
		changesPath: changesPath,
	}, nil
}

func (cs *ChangeStorage) sessionFilePath(sessionID string) string {
	return filepath.Join(cs.changesPath, changeFileName(sessionID))
}

func changeFileName(sessionID string) string {
	return changeFilePrefix + sessionID + changeFileSuffix
}

func parseChangeSessionIDFromFileName(name string) (string, bool) {
	if !strings.HasPrefix(name, changeFilePrefix) || !strings.HasSuffix(name, changeFileSuffix) {
		return "", false
	}
	sessionID := strings.TrimSuffix(strings.TrimPrefix(name, changeFilePrefix), changeFileSuffix)
	if sessionID == "" {
		return "", false
	}
	return sessionID, true
}
