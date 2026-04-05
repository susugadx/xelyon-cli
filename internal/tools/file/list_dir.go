package file

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ExecuteListDir はディレクトリ一覧を取得する。
func ExecuteListDir(path string, depth int) string {
	return ExecuteListDirWithRuntime(config.DefaultConfig(), nil, path, depth)
}

// ExecuteListDirWithRuntime は runtime 設定を指定してディレクトリ一覧を取得する。
func ExecuteListDirWithRuntime(cfg *config.Config, cache tools.ToolCacheInterface, path string, depth int) string {
	req, errResult := resolveListDirRequest(cfg, path, depth)
	if errResult != "" {
		return errResult
	}

	if cached, hit := loadCachedListDir(cache, req); hit {
		return cached
	}

	section := summarizeListDir(req.absPath, req.rootPath, "", req.depth, req.matcher, &listDirBudget{remainingEntries: maxEntries}, true)
	result := strings.Join(renderListDirSummary(req.absPath, req.depth, section), "\n")
	storeCachedListDir(cache, req, result)
	return result
}

func normalizeListDirDepth(depth int) int {
	if depth <= 0 {
		return 1
	}
	if depth > 3 {
		return 3
	}
	return depth
}

func validateListDirTarget(absPath string) string {
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if !info.IsDir() {
		return fmt.Sprintf("Error: %s is not a directory", absPath)
	}
	return ""
}
