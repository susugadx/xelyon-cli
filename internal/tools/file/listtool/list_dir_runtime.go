package listtool

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy"
)

type listDirRequest struct {
	absPath            string
	depth              int
	cacheKey           string
	cacheable          bool
	ignoreMode         listDirIgnoreMode
	rootPath           string
	filterRoot         string
	matcher            *pathmatch.Matcher
	fileFilter         string
	projectMap         *repomap.ProjectMap
	projectMapStateKey string
}

type listDirIgnoreMode string

const (
	listDirApplyIgnores  listDirIgnoreMode = "apply"
	listDirBypassIgnores listDirIgnoreMode = "bypass"
)

func resolveListDirRequest(cfg *config.Config, path string, depth int, allowedRoots []string, fileFilter string, workspaceRoot string, projectMap *repomap.ProjectMap, projectMapStateKey string, ignoreMode listDirIgnoreMode) (listDirRequest, string) {
	if ignoreMode == "" {
		ignoreMode = listDirApplyIgnores
	}
	req := listDirRequest{
		depth:      normalizeListDirDepth(depth),
		ignoreMode: ignoreMode,
	}

	absPath, errResult := pathpolicy.ResolveValidatedPathWithRoots(common.NewOutput(io.Discard, io.Discard), path, allowedRoots, "path is empty")
	if errResult != "" {
		return listDirRequest{}, errResult
	}
	req.absPath = absPath

	if errResult := validateListDirTarget(req.absPath); errResult != "" {
		return listDirRequest{}, errResult
	}

	req.fileFilter = fileFilter
	req.rootPath, req.matcher = resolveListDirMatcher(cfg, req.absPath, req.ignoreMode)
	req.filterRoot = resolveListDirFilterRoot(req.absPath, req.rootPath, workspaceRoot)
	req.projectMap = projectMap
	req.projectMapStateKey = strings.TrimSpace(projectMapStateKey)
	req.cacheKey, req.cacheable = buildListDirCacheKey(req.absPath, req.depth, req.fileFilter, req.filterRoot, req.projectMapStateKey, req.ignoreMode)
	return req, ""
}

func loadCachedListDir(cache tools.ToolCacheInterface, req listDirRequest) (string, bool) {
	if cache == nil || !req.cacheable || req.cacheKey == "" {
		return "", false
	}
	return cache.GetDir(req.cacheKey)
}

func storeCachedListDir(cache tools.ToolCacheInterface, req listDirRequest, result string) {
	if cache == nil || !req.cacheable || req.cacheKey == "" {
		return
	}
	cache.SetDir(req.cacheKey, result)
}

func resolveListDirMatcher(cfg *config.Config, absPath string, ignoreMode listDirIgnoreMode) (string, *pathmatch.Matcher) {
	projectCfg := config.LoadProjectConfig()
	rootPath := absPath
	if projectCfg != nil && projectCfg.FilePath != "" {
		rootPath = filepath.Dir(projectCfg.FilePath)
	}
	if ignoreMode == listDirBypassIgnores {
		return rootPath, nil
	}

	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, projectCfg)
	return rootPath, pathmatch.NewMatcher(ignorePatterns)
}

func resolveListDirFilterRoot(absPath, matcherRoot, workspaceRoot string) string {
	if root := pathpolicy.NormalizeWorkspaceRoot(workspaceRoot); root != "" && pathpolicy.IsPathWithinRoot(absPath, root) {
		return root
	}
	if root := pathpolicy.NormalizeWorkspaceRoot(matcherRoot); root != "" && pathpolicy.IsPathWithinRoot(absPath, root) {
		return root
	}
	return pathpolicy.NormalizeWorkspaceRoot(absPath)
}
