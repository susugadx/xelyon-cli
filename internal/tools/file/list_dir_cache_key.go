package file

import (
	"strconv"
	"strings"
)

const listDirCacheKeyMetadataPrefix = "::depth="

// ListDirCachePhysicalPath returns the physical directory path behind a list_dir
// cache key. Legacy plain-path keys are preserved as-is.
func ListDirCachePhysicalPath(cacheKey string) string {
	path, _ := splitListDirCacheKey(cacheKey)
	return normalizeWorkspaceRoot(path)
}

// NormalizeListDirCacheKey canonicalizes list_dir cache keys while preserving
// any scope metadata suffix.
func NormalizeListDirCacheKey(cacheKey string) string {
	path, suffix := splitListDirCacheKey(cacheKey)
	path = normalizeWorkspaceRoot(path)
	if path == "" {
		return ""
	}
	return path + suffix
}

func buildListDirCacheKey(absPath string, depth int, fileFilter string, filterRoot string, projectMapStateKey string, ignoreMode listDirIgnoreMode) (string, bool) {
	absPath = normalizeWorkspaceRoot(absPath)
	filterRoot = normalizeWorkspaceRoot(filterRoot)
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
