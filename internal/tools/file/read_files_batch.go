package file

import (
	"fmt"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type readFileBatchResult struct {
	entry     string
	filePath  string
	startLine int
	endLine   int
	result    string
}

func validateReadFilesPaths(paths []string) string {
	if len(paths) == 0 {
		return "Error: paths is empty"
	}
	if len(paths) > MaxReadFilesPaths {
		return fmt.Sprintf("Error: too many paths (max %d), got %d", MaxReadFilesPaths, len(paths))
	}
	return ""
}

func resolveReadFilesBudget(budgetOverride int) int {
	if budgetOverride > 0 {
		return budgetOverride
	}
	return DefaultFullLines
}

func readFilesInParallel(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, paths []string, budget int) []readFileBatchResult {
	sem := make(chan struct{}, tools.MaxParallelTools)
	results := make([]readFileBatchResult, len(paths))
	var wg sync.WaitGroup

	for i, entry := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, rawEntry string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = executeReadBatchEntry(out, cfg, cache, rawEntry, budget)
		}(i, entry)
	}
	wg.Wait()
	return results
}

func executeReadBatchEntry(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, rawEntry string, budget int) readFileBatchResult {
	path, startLine, endLine := parsePath(rawEntry)

	result := ""
	if startLine > 0 || endLine > 0 {
		result = executeReadFileCore(out, cfg, cache, path, startLine, endLine, DefaultFullLines)
	} else {
		result = executeReadFileCore(out, cfg, cache, path, 0, 0, budget)
	}

	return readFileBatchResult{
		entry:     rawEntry,
		filePath:  path,
		startLine: startLine,
		endLine:   endLine,
		result:    result,
	}
}
