package listtool

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ExecuteListDir はディレクトリ一覧を取得する。
func ExecuteListDir(path string, depth int) string {
	return ExecuteListDirWithRuntime(config.DefaultConfig(), nil, path, depth)
}

// ExecuteListDirWithRuntime は runtime 設定を指定してディレクトリ一覧を取得する。
func ExecuteListDirWithRuntime(cfg *config.Config, cache tools.ToolCacheInterface, path string, depth int) string {
	return executeListDirWithRuntime(cfg, cache, path, depth, nil, "", "", nil, "")
}

// ExecuteWithContext は ExecutionContext に紐づく list_dir runtime で directory を読む。
func ExecuteWithContext(execCtx tools.ExecutionContext, path string, depth int) string {
	return executeListDirWithRuntime(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), path, depth, nil, "", "", execCtx.ProjectMap, execCtx.ProjectMapStateKey)
}

// ResolvedTarget は caller が既に解決した list_dir directory target を表す。
type ResolvedTarget struct {
	Path          string
	AllowedRoots  []string
	FileFilter    string
	WorkspaceRoot string
	BypassIgnores bool
}

// ExecuteResolvedTarget は解決済み directory target を list_dir と同じ runtime で読む。
func ExecuteResolvedTarget(execCtx tools.ExecutionContext, target ResolvedTarget, depth int) string {
	ignoreMode := listDirApplyIgnores
	if target.BypassIgnores {
		ignoreMode = listDirBypassIgnores
	}
	return executeListDirWithRuntimeMode(
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		target.Path,
		depth,
		target.AllowedRoots,
		target.FileFilter,
		target.WorkspaceRoot,
		execCtx.ProjectMap,
		execCtx.ProjectMapStateKey,
		ignoreMode,
	)
}

func executeListDirWithRuntime(cfg *config.Config, cache tools.ToolCacheInterface, path string, depth int, allowedRoots []string, fileFilter string, workspaceRoot string, projectMap *repomap.ProjectMap, projectMapStateKey string) string {
	return executeListDirWithRuntimeMode(cfg, cache, path, depth, allowedRoots, fileFilter, workspaceRoot, projectMap, projectMapStateKey, listDirApplyIgnores)
}

func executeListDirWithRuntimeMode(cfg *config.Config, cache tools.ToolCacheInterface, path string, depth int, allowedRoots []string, fileFilter string, workspaceRoot string, projectMap *repomap.ProjectMap, projectMapStateKey string, ignoreMode listDirIgnoreMode) string {
	req, errResult := resolveListDirRequest(cfg, path, depth, allowedRoots, fileFilter, workspaceRoot, projectMap, projectMapStateKey, ignoreMode)
	if errResult != "" {
		return errResult
	}

	if cached, hit := loadCachedListDir(cache, req); hit {
		return cached
	}

	section := summarizeListDir(req.absPath, req.rootPath, req.filterRoot, "", req.depth, req.matcher, req.fileFilter, req.projectMap, &listDirBudget{remainingEntries: maxEntries}, true)
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
