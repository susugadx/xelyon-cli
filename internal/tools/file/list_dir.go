package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// デフォルト除外ディレクトリ
var defaultIgnoreDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	".next": true, "__pycache__": true, ".venv": true,
	"dist": true, "build": true, ".idea": true, ".vscode": true,
}

const maxEntries = 200

// ExecuteListDir はディレクトリ一覧を取得
func ExecuteListDir(path string, depth int) string {
	if depth <= 0 {
		depth = 1
	}
	if depth > 3 {
		depth = 3
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	cachePath := absPath + "::depth=" + strconv.Itoa(depth)

	// キャッシュチェック
	if tools.GlobalToolCache != nil {
		if cached, hit := tools.GlobalToolCache.GetDir(cachePath); hit {
			return cached
		}
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if !info.IsDir() {
		return fmt.Sprintf("Error: %s is not a directory", absPath)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("📂 %s", absPath))

	shown := 0
	total := 0
	truncated := false
	var walk func(dir string, prefix string, remain int)
	walk = func(dir string, prefix string, remain int) {
		rawEntries, readErr := os.ReadDir(dir)
		if readErr != nil {
			total++
			if shown < maxEntries {
				lines = append(lines, prefix+"[error reading directory]")
				shown++
			} else {
				truncated = true
			}
			return
		}

		var entries []os.DirEntry
		for _, entry := range rawEntries {
			if entry.IsDir() && defaultIgnoreDirs[entry.Name()] {
				continue
			}
			entries = append(entries, entry)
		}

		for i, entry := range entries {
			total++
			connector := "├── "
			nextPrefix := prefix + "│   "
			if i == len(entries)-1 {
				connector = "└── "
				nextPrefix = prefix + "    "
			}

			info, _ := entry.Info()
			size := ""
			icon := "📄 "
			if entry.IsDir() {
				icon = "📁 "
			} else if info != nil {
				size = fmt.Sprintf(" (%d bytes)", info.Size())
			}

			if shown < maxEntries {
				lines = append(lines, prefix+connector+icon+entry.Name()+size)
				shown++
			} else {
				truncated = true
			}

			if entry.IsDir() && remain > 1 {
				walk(filepath.Join(dir, entry.Name()), nextPrefix, remain-1)
			}
		}
	}

	walk(absPath, "", depth)

	if truncated && total > maxEntries {
		lines = append(lines, fmt.Sprintf("... (%d more entries, showing first %d)", total-maxEntries, maxEntries))
	}

	result := strings.Join(lines, "\n")

	// キャッシュに保存
	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetDir(cachePath, result)
	}

	return result
}
