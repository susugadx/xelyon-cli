package history

import (
	"encoding/json"
	"fmt"
	"os"
)

// AppendChange はセッションの変更履歴にエントリを追加
func (cs *ChangeStorage) AppendChange(sessionID string, change ChangeRecordInput) error {
	filePath := cs.sessionFilePath(sessionID)

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open changes file: %w", err)
	}
	defer file.Close()

	paths := collectChangePaths(change)
	if len(paths) == 0 {
		return nil
	}

	for _, path := range paths {
		pc := PersistentChange{
			SessionID:   sessionID,
			Timestamp:   change.Timestamp,
			Tool:        change.Tool,
			FilePath:    path,
			Description: change.Description,
		}

		data, err := json.Marshal(pc)
		if err != nil {
			return fmt.Errorf("failed to marshal change: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("failed to write change: %w", err)
		}
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	return nil
}

func collectChangePaths(change ChangeRecordInput) []string {
	paths := make([]string, 0, len(change.Details))
	if len(change.Details) > 0 {
		for _, detail := range change.Details {
			if detail.FilePath != "" {
				paths = append(paths, detail.FilePath)
			}
		}
	}
	if len(paths) == 0 && change.FilePath != "" {
		paths = append(paths, change.FilePath)
	}
	return paths
}
