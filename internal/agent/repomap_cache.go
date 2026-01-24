package agent

import (
	"os"
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/cache"
	"github.com/susugadx/xelyon-cli/internal/config"
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

// getMaxTokens はRepoMapのmax_tokensを返す（設定または自動計算）
func getMaxTokens(projectPath string) int {
	cfg := config.GetGlobalConfig()

	// 明示的に設定されていれば使う
	if cfg.RepoMap.MaxTokens > 0 {
		return cfg.RepoMap.MaxTokens
	}

	// 自動計算: ファイル数ベース
	fileCount := countSupportedFiles(projectPath)
	var tokens int
	switch {
	case fileCount < 50:
		tokens = 1000
	case fileCount < 100:
		tokens = 2000
	case fileCount < 200:
		tokens = 4000
	case fileCount < 400:
		tokens = 6000
	default:
		tokens = 8000
	}

	// 自動計算の上限: 10,000
	const maxAutoLimit = 10000
	if tokens > maxAutoLimit {
		tokens = maxAutoLimit
	}
	return tokens
}

// countSupportedFiles はサポートされているファイル数をカウント
func countSupportedFiles(projectPath string) int {
	count := 0
	_ = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if repomap.IsSupportedFile(path) {
			count++
		}
		return nil
	})
	return count
}
