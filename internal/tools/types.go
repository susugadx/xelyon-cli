package tools

import "time"

// ToolCall はAIからのツール呼び出し
type ToolCall struct {
	Tool string            `json:"tool"`
	Args map[string]string `json:"args"`
}

// FileChange はファイル変更履歴
type FileChange struct {
	FilePath    string
	BackupPath  string
	Timestamp   time.Time
	Tool        string
	Description string
}
