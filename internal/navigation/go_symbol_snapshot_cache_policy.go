package navigation

import (
	"path/filepath"
	"strings"
)

func goSymbolSnapshotCacheKey(rootPath, stateKey string) string {
	rootPath = normalizeNavigationRootPath(rootPath)
	stateKey = strings.TrimSpace(stateKey)
	if rootPath == "" || stateKey == "" {
		return ""
	}
	return rootPath + "::" + stateKey
}

func trackGoSymbolSnapshotRootCacheKey(cacheKey string, snapshot *goSymbolSnapshot) {
	if snapshot == nil {
		return
	}

	rootKey := filepath.Clean(snapshot.RootPath)
	if rootKey == "" || rootKey == "." {
		return
	}
	if previous, ok := goSymbolSnapshotRootKeys.Load(rootKey); ok {
		if oldKey, _ := previous.(string); oldKey != "" && oldKey != cacheKey {
			goSymbolSnapshotCache.Delete(oldKey)
		}
	}
	goSymbolSnapshotRootKeys.Store(rootKey, cacheKey)
}
