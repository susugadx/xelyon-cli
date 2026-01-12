package agent

import (
	"github.com/susugadx/xelyon-cli/internal/cache"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

// loadRepoMapForProject builds repo map string with a persistent cache.
//
// Cache key: project path (cwd)
// Cache invalidation: project fingerprint (max modTime under the project directory)
//
// This is best-effort. On any cache error, it falls back to generating repo map.
func loadRepoMapForProject(projectPath string, maxTokens int) (repoMapStr string, symbols int, files int, fromCache bool) {
	fingerprint, err := cache.ComputeProjectFingerprint(projectPath)
	if err == nil {
		if cached, err := cache.LoadRepoMapFromDisk(projectPath, fingerprint); err == nil {
			return cached, 0, 0, true
		}
	}

	rm := repomap.NewRepoMap(projectPath, maxTokens)
	if err := rm.Build(); err != nil || rm.GetSymbolCount() == 0 {
		return "", 0, 0, false
	}

	repoMapStr = rm.Generate()
	_ = cache.SaveRepoMapToDisk(projectPath, fingerprint, repoMapStr)
	return repoMapStr, rm.GetSymbolCount(), len(rm.Files), false
}
