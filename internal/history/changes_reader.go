package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// LoadSessionChanges は指定セッションの変更履歴を読み込み
func (cs *ChangeStorage) LoadSessionChanges(sessionID string) ([]PersistentChange, error) {
	filePath := cs.sessionFilePath(sessionID)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []PersistentChange{}, nil
	}

	file, err := os.Open(filePath)
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
