package agent

import (
	"encoding/json"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// extractToolFilePath はツール呼び出しから表示用ターゲットを抽出する。
func extractToolFilePath(tc *tools.ToolCall) string {
	if tc == nil {
		return ""
	}

	switch tc.Tool {
	case "gather_context":
		if query := tc.Args["query"]; query != "" {
			return query
		}
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "read_file", "write_file", "str_replace", "delete_file", "list_dir", "lint", "format":
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "search_code":
		if pattern := tc.Args["pattern"]; pattern != "" {
			return fmt.Sprintf("%q", pattern)
		}
		if path := tc.Args["path"]; path != "" {
			return path
		}
	case "bash":
		if cmd := tc.Args["command"]; cmd != "" {
			return truncateEventOutput(cmd, 40)
		}
	}

	if rawPaths := tc.Args["paths"]; rawPaths != "" {
		var paths []string
		if err := json.Unmarshal([]byte(rawPaths), &paths); err == nil && len(paths) > 0 {
			if len(paths) == 1 {
				return paths[0]
			}
			return fmt.Sprintf("%s (+%d files)", paths[0], len(paths)-1)
		}
	}
	if path := tc.Args["path"]; path != "" {
		return path
	}
	if pattern := tc.Args["pattern"]; pattern != "" {
		return pattern
	}
	if symbol := tc.Args["symbol"]; symbol != "" {
		return symbol
	}
	return ""
}

// truncateEventOutput はイベント出力を制限する。
func truncateEventOutput(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
