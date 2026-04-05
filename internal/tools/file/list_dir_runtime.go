package file

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type listDirRequest struct {
	absPath  string
	depth    int
	cacheKey string
	rootPath string
	matcher  *pathmatch.Matcher
}

func resolveListDirRequest(cfg *config.Config, path string, depth int) (listDirRequest, string) {
	req := listDirRequest{depth: normalizeListDirDepth(depth)}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return listDirRequest{}, fmt.Sprintf("Error: %v", err)
	}
	req.absPath = absPath

	if errResult := validateListDirTarget(req.absPath); errResult != "" {
		return listDirRequest{}, errResult
	}

	req.cacheKey = buildListDirCacheKey(req.absPath, req.depth)
	req.rootPath, req.matcher = resolveListDirMatcher(cfg, req.absPath)
	return req, ""
}

func buildListDirCacheKey(absPath string, depth int) string {
	return absPath + "::depth=" + strconv.Itoa(depth)
}

func loadCachedListDir(cache tools.ToolCacheInterface, req listDirRequest) (string, bool) {
	if cache == nil {
		return "", false
	}
	return cache.GetDir(req.cacheKey)
}

func storeCachedListDir(cache tools.ToolCacheInterface, req listDirRequest, result string) {
	if cache == nil {
		return
	}
	cache.SetDir(req.cacheKey, result)
}

func resolveListDirMatcher(cfg *config.Config, absPath string) (string, *pathmatch.Matcher) {
	projectCfg := config.LoadProjectConfig()
	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, projectCfg)
	matcher := pathmatch.NewMatcher(ignorePatterns)

	rootPath := absPath
	if projectCfg != nil && projectCfg.FilePath != "" {
		rootPath = filepath.Dir(projectCfg.FilePath)
	}
	return rootPath, matcher
}
