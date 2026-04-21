package navigation

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

var (
	goSymbolSnapshotCache    sync.Map
	goSymbolSnapshotRootKeys sync.Map
)

func init() {
	tools.AddSearchCacheLifecycleHooks(clearGoSymbolSnapshotCache, clearGoSymbolSnapshotCacheWithKeys, clearGoSymbolSnapshotCacheWithKeys)
}

func loadGoSymbolSnapshot(runtime GoSymbolRuntime) *goSymbolSnapshot {
	cacheKey := goSymbolSnapshotCacheKey(runtime.ProjectMapRootPath, runtime.ProjectMapStateKey)
	if cacheKey != "" {
		if snapshot := lookupGoSymbolSnapshot(cacheKey); snapshot != nil {
			return snapshot
		}
	}
	if runtime.ProjectMap != nil {
		snapshot := buildGoSymbolSnapshot(runtime.ProjectMap, runtime.ProjectMapRootPath, runtime.ProjectMapStateKey)
		if snapshot != nil {
			storeGoSymbolSnapshot(cacheKey, snapshot)
		}
		return snapshot
	}
	return nil
}

func goSymbolSnapshotCacheKey(rootPath, stateKey string) string {
	rootPath = strings.TrimSpace(rootPath)
	stateKey = strings.TrimSpace(stateKey)
	if rootPath == "" || stateKey == "" {
		return ""
	}
	return filepath.Clean(rootPath) + "::" + stateKey
}

func lookupGoSymbolSnapshot(cacheKey string) *goSymbolSnapshot {
	if strings.TrimSpace(cacheKey) == "" {
		return nil
	}
	value, ok := goSymbolSnapshotCache.Load(cacheKey)
	if !ok {
		return nil
	}
	snapshot, _ := value.(*goSymbolSnapshot)
	return snapshot
}

func storeGoSymbolSnapshot(cacheKey string, snapshot *goSymbolSnapshot) {
	if snapshot == nil || strings.TrimSpace(cacheKey) == "" {
		return
	}
	goSymbolSnapshotCache.Store(cacheKey, snapshot)

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

func clearGoSymbolSnapshotCache() {
	goSymbolSnapshotCache.Range(func(key, value any) bool {
		goSymbolSnapshotCache.Delete(key)
		return true
	})
	goSymbolSnapshotRootKeys.Range(func(key, value any) bool {
		goSymbolSnapshotRootKeys.Delete(key)
		return true
	})
}

func clearGoSymbolSnapshotCacheWithKeys(_ []string) {
	clearGoSymbolSnapshotCache()
}
