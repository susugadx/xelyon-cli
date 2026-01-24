package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ExecuteListDir はディレクトリ一覧を取得
func ExecuteListDir(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// キャッシュチェック
	if tools.GlobalToolCache != nil {
		if cached, hit := tools.GlobalToolCache.GetDir(absPath); hit {
			return cached
		}
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading directory: %v", err)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("📂 %s", absPath))

	for _, entry := range entries {
		prefix := "  📄 "
		if entry.IsDir() {
			prefix = "  📁 "
		}
		info, _ := entry.Info()
		size := ""
		if info != nil && !entry.IsDir() {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		lines = append(lines, prefix+entry.Name()+size)
	}

	result := strings.Join(lines, "\n")

	// キャッシュに保存
	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetDir(absPath, result)
	}

	return result
}
