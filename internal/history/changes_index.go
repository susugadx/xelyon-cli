package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ListSessions は全セッションの変更履歴を取得
func (cs *ChangeStorage) ListSessions() ([]SessionInfo, error) {
	pattern := filepath.Join(cs.changesPath, changeFilePattern)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob changes files: %w", err)
	}

	sessions := []SessionInfo{}
	for _, file := range files {
		basename := filepath.Base(file)
		sessionID, ok := parseChangeSessionIDFromFileName(basename)
		if !ok {
			continue
		}

		summary, err := summarizeSessionChangesFile(file)
		if err != nil {
			continue
		}
		if !summary.hasChanges() {
			continue
		}

		sessions = append(sessions, SessionInfo{
			SessionID:    sessionID,
			ChangeCount:  summary.changeCount,
			FirstChange:  summary.firstChange,
			LastChange:   summary.lastChange,
			FilesChanged: summary.filesChanged,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastChange.After(sessions[j].LastChange)
	})

	return sessions, nil
}

// CleanupOldChanges は古い変更履歴を削除（デフォルト30日以上前）
func (cs *ChangeStorage) CleanupOldChanges(daysToKeep int) (int, error) {
	if daysToKeep <= 0 {
		daysToKeep = 30
	}

	cutoff := time.Now().AddDate(0, 0, -daysToKeep)
	deletedCount := 0

	pattern := filepath.Join(cs.changesPath, changeFilePattern)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("failed to glob changes files: %w", err)
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				continue
			}
			deletedCount++
		}
	}

	return deletedCount, nil
}

type sessionChangeSummary struct {
	changeCount  int
	firstChange  time.Time
	lastChange   time.Time
	filesChanged map[string]int
}

func (s sessionChangeSummary) hasChanges() bool {
	return s.changeCount > 0
}

func summarizeSessionChangesFile(filePath string) (sessionChangeSummary, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return sessionChangeSummary{}, fmt.Errorf("failed to open changes file: %w", err)
	}
	defer file.Close()

	return summarizeSessionChanges(file)
}

func summarizeSessionChanges(file *os.File) (sessionChangeSummary, error) {
	summary := sessionChangeSummary{
		filesChanged: make(map[string]int),
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var change PersistentChange
		if err := json.Unmarshal([]byte(line), &change); err != nil {
			continue
		}

		summary.changeCount++
		summary.filesChanged[change.FilePath]++
		if summary.changeCount == 1 {
			summary.firstChange = change.Timestamp
			summary.lastChange = change.Timestamp
			continue
		}
		if change.Timestamp.Before(summary.firstChange) {
			summary.firstChange = change.Timestamp
		}
		if change.Timestamp.After(summary.lastChange) {
			summary.lastChange = change.Timestamp
		}
	}
	if err := scanner.Err(); err != nil {
		return sessionChangeSummary{}, fmt.Errorf("failed to read changes file: %w", err)
	}
	return summary, nil
}
