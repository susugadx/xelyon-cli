package navigation

import "github.com/susugadx/xelyon-cli/internal/searchcache"

func init() {
	searchcache.AddSearchCacheLifecycleHooks(
		clearGoSymbolSnapshotCache,
		clearGoSymbolSnapshotCacheWithKeys,
		clearGoSymbolSnapshotCacheWithKeys,
	)
}
