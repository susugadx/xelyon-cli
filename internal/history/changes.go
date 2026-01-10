package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	changesDir = ".xelyon/changes"
)

// PersistentChange は永続化される変更履歴
type PersistentChange struct {
	SessionID   string    `json:"session_id"`
	Timestamp   time.Time `json:"timestamp"`
	Tool        string    `json:"tool"`
	FilePath    string    `json:"file_path"`
	BackupPath  string    `json:"backup_path"`
	Description string    `json:"description"`
}

// ChangeStorage は変更履歴のストレージ
type ChangeStorage struct {
	changesPath string
}

// NewChangeStorage は新しいChangeStorageを作成
func NewChangeStorage() (*ChangeStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	changesPath := filepath.Join(home, changesDir)

	// ディレクトリ作成
	if err := os.MkdirAll(changesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create changes directory: %w", err)
	}

	return &ChangeStorage{
		changesPath: changesPath,
	}, nil
}

// AppendChange はセッションの変更履歴にエントリを追加
func (cs *ChangeStorage) AppendChange(sessionID string, change tools.FileChange) error {
	// セッション別のJSONLファイルパス
	filename := fmt.Sprintf("changes_%s.jsonl", sessionID)
	filepath := filepath.Join(cs.changesPath, filename)

	// ファイルをappendモードでオープン
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open changes file: %w", err)
	}
	defer file.Close()

	// PersistentChangeに変換
	pc := PersistentChange{
		SessionID:   sessionID,
		Timestamp:   change.Timestamp,
		Tool:        change.Tool,
		FilePath:    change.FilePath,
		BackupPath:  change.BackupPath,
		Description: change.Description,
	}

	// JSON化
	data, err := json.Marshal(pc)
	if err != nil {
		return fmt.Errorf("failed to marshal change: %w", err)
	}

	// 書き込み
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write change: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// LoadSessionChanges は指定セッションの変更履歴を読み込み
func (cs *ChangeStorage) LoadSessionChanges(sessionID string) ([]PersistentChange, error) {
	filename := fmt.Sprintf("changes_%s.jsonl", sessionID)
	filepath := filepath.Join(cs.changesPath, filename)

	// ファイルが存在しない場合は空配列を返す
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return []PersistentChange{}, nil
	}

	// ファイルを読み込み
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open changes file: %w", err)
	}
	defer file.Close()

	changes := []PersistentChange{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var change PersistentChange
		if err := json.Unmarshal([]byte(line), &change); err != nil {
			// 破損した行はスキップ
			continue
		}

		changes = append(changes, change)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read changes file: %w", err)
	}

	return changes, nil
}

// SessionInfo はセッション情報（変更履歴表示用）
type SessionInfo struct {
	SessionID    string
	ChangeCount  int
	FirstChange  time.Time
	LastChange   time.Time
	FilesChanged map[string]int // ファイルパス → 変更回数
}

// ListSessions は全セッションの変更履歴を取得
func (cs *ChangeStorage) ListSessions() ([]SessionInfo, error) {
	// changesディレクトリ内のchanges_*.jsonlファイルを探す
	pattern := filepath.Join(cs.changesPath, "changes_*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob changes files: %w", err)
	}

	sessions := []SessionInfo{}

	for _, file := range files {
		// セッションIDを抽出
		basename := filepath.Base(file)
		sessionID := basename[8 : len(basename)-6] // "changes_" と ".jsonl" を除去

		// 変更履歴を読み込み
		changes, err := cs.LoadSessionChanges(sessionID)
		if err != nil {
			continue // エラーのあるファイルはスキップ
		}

		if len(changes) == 0 {
			continue
		}

		// SessionInfo作成
		filesChanged := make(map[string]int)
		firstChange := changes[0].Timestamp
		lastChange := changes[len(changes)-1].Timestamp

		for _, change := range changes {
			filesChanged[change.FilePath]++
			if change.Timestamp.Before(firstChange) {
				firstChange = change.Timestamp
			}
			if change.Timestamp.After(lastChange) {
				lastChange = change.Timestamp
			}
		}

		sessions = append(sessions, SessionInfo{
			SessionID:    sessionID,
			ChangeCount:  len(changes),
			FirstChange:  firstChange,
			LastChange:   lastChange,
			FilesChanged: filesChanged,
		})
	}

	// 最終変更日時でソート（新しい順）
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastChange.After(sessions[j].LastChange)
	})

	return sessions, nil
}

// CleanupOldChanges は古い変更履歴を削除（デフォルト30日以上前）
func (cs *ChangeStorage) CleanupOldChanges(daysToKeep int) (int, error) {
	if daysToKeep <= 0 {
		daysToKeep = 30 // デフォルト30日
	}

	cutoff := time.Now().AddDate(0, 0, -daysToKeep)
	deletedCount := 0

	// changesディレクトリ内のchanges_*.jsonlファイルを探す
	pattern := filepath.Join(cs.changesPath, "changes_*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("failed to glob changes files: %w", err)
	}

	for _, file := range files {
		// ファイルの最終更新日時を確認
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			// 削除
			if err := os.Remove(file); err != nil {
				continue
			}
			deletedCount++
		}
	}

	return deletedCount, nil
}

// UndoSessionChange は指定セッションの指定変更を取り消す
func (cs *ChangeStorage) UndoSessionChange(change PersistentChange) error {
	// バックアップファイルが存在しない場合
	if change.BackupPath == "" {
		return fmt.Errorf("no backup available for this change")
	}

	// バックアップファイルを確認
	if _, err := os.Stat(change.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", change.BackupPath)
	}

	// バックアップから復元
	backupContent, err := os.ReadFile(change.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if err := os.WriteFile(change.FilePath, backupContent, 0644); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}
