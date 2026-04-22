package navigation

import "github.com/susugadx/xelyon-cli/internal/tools"

func init() {
	tools.AddSearchCacheLifecycleHooks(
		clearGoSymbolSnapshotCache,
		clearGoSymbolSnapshotCacheWithKeys,
		clearGoSymbolSnapshotCacheWithKeys,
	)
}
