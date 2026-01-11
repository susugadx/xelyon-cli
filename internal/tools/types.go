package tools

import (
	"fmt"
	"time"
)

// ToolCall はAIからのツール呼び出し
type ToolCall struct {
	Tool    string         `json:"tool"`
	RawArgs map[string]any `json:"args"`
	Args    map[string]string
}

// NormalizeArgs はRawArgsをArgsに変換（数値→文字列）
func (tc *ToolCall) NormalizeArgs() {
	tc.Args = make(map[string]string)
	for k, v := range tc.RawArgs {
		switch val := v.(type) {
		case string:
			tc.Args[k] = val
		case float64:
			// JSONの数値はfloat64としてパースされる
			if val == float64(int64(val)) {
				tc.Args[k] = fmt.Sprintf("%d", int64(val))
			} else {
				tc.Args[k] = fmt.Sprintf("%g", val)
			}
		case int64:
			tc.Args[k] = fmt.Sprintf("%d", val)
		case bool:
			tc.Args[k] = fmt.Sprintf("%t", val)
		default:
			tc.Args[k] = fmt.Sprintf("%v", v)
		}
	}
}

// FileChange はファイル変更履歴
type FileChange struct {
	FilePath    string
	BackupPath  string
	Timestamp   time.Time
	Tool        string
	Description string
}
