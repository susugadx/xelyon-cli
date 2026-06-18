package listtool

import (
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy"
)

const listDirCacheKeyMetadataPrefix = "::depth="

// CachePhysicalPath returns the physical directory path behind a list_dir
// cache key. Legacy plain-path keys are preserved as-is.
func CachePhysicalPath(cacheKey string) string {
	path, _ := splitListDirCacheKey(cacheKey)
	return pathpolicy.NormalizeWorkspaceRoot(path)
}

// NormalizeCacheKey canonicalizes list_dir cache keys while preserving
// any scope metadata suffix.
func NormalizeCacheKey(cacheKey string) string {
	path, suffix := splitListDirCacheKey(cacheKey)
	path = pathpolicy.NormalizeWorkspaceRoot(path)
	if path == "" {
		return ""
	}
	return path + suffix
}

func buildListDirCacheKey(absPath string, depth int, fileFilter string, filterRoot string, projectMapStateKey string, ignoreMode listDirIgnoreMode) (string, bool) {
	absPath = pathpolicy.NormalizeWorkspaceRoot(absPath)
	filterRoot = pathpolicy.NormalizeWorkspaceRoot(filterRoot)
	if absPath == "" {
		return "", false
	}
	if ignoreMode == "" {
		ignoreMode = listDirApplyIgnores
	}

	key := absPath +
		listDirCacheKeyMetadataPrefix + strconv.Itoa(depth) +
		"::filter=" + fileFilter +
		"::filter_root=" + filterRoot +
		"::ignore=" + string(ignoreMode)
	if strings.TrimSpace(fileFilter) == "" {
		return key, true
	}

	projectMapStateKey = strings.TrimSpace(projectMapStateKey)
	if projectMapStateKey == "" {
		return "", false
	}
	return key + "::pm_state=" + projectMapStateKey, true
}

func splitListDirCacheKey(cacheKey string) (string, string) {
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" {
		return "", ""
	}
	if idx := strings.Index(cacheKey, listDirCacheKeyMetadataPrefix); idx >= 0 {
		return cacheKey[:idx], cacheKey[idx:]
	}
	return cacheKey, ""
}
