package file

import (
	"fmt"
	"io"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type readFileBatchResult struct {
	entry        string
	filePath     string
	resolvedPath string
	startLine    int
	endLine      int
	locatorName  string
	result       string
}

func validateReadFilesPaths(paths []string) string {
	return validateReadRequestCount(len(paths))
}

func validateReadRequestCount(count int) string {
	if count == 0 {
		return "Error: paths is empty"
	}
	if count > MaxReadFilesPaths {
		return fmt.Sprintf("Error: too many paths (max %d), got %d", MaxReadFilesPaths, count)
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
	return readRequestsInParallel(out, cfg, cache, buildReadRequestsFromPaths(paths, readDetailAuto), budget)
}

func readRequestsInParallel(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, requests []readRequest, budget int) []readFileBatchResult {
	sem := make(chan struct{}, tools.MaxParallelTools)
	results := make([]readFileBatchResult, len(requests))
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, request readRequest) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = executeReadBatchRequest(out, cfg, cache, request, budget)
		}(i, req)
	}
	wg.Wait()
	return results
}

func executeReadBatchEntry(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, rawEntry string, budget int) readFileBatchResult {
	requests := buildReadRequestsFromPaths([]string{rawEntry}, readDetailAuto)
	return executeReadBatchRequest(out, cfg, cache, requests[0], budget)
}

func executeReadBatchRequest(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, req readRequest, budget int) readFileBatchResult {
	req = normalizeReadRequestForExecution(req)

	entry := req.RawEntry
	if req.Source == readRequestSourceLocator {
		entry = req.RangeEntry
		if entry == "" {
			entry = formatReadRangeEntry(req.FilePath, req.StartLine, req.EndLine)
		}
	}

	locatorName := ""
	if req.Locator != nil {
		locatorName = req.Locator.Name
	}

	return newReadFileBatchResult(req, entry, locatorName, executeReadFileRequest(out, cfg, cache, req, budget))
}

func newReadFileBatchResult(req readRequest, entry, locatorName, result string) readFileBatchResult {
	return readFileBatchResult{
		entry:        entry,
		filePath:     req.FilePath,
		resolvedPath: resolvedReadResultPath(req),
		startLine:    req.StartLine,
		endLine:      req.EndLine,
		locatorName:  locatorName,
		result:       result,
	}
}

func resolvedReadResultPath(req readRequest) string {
	if req.ResolvedPath != "" {
		return req.ResolvedPath
	}
	return resolveReadResultPath(req)
}

func resolveReadResultPath(req readRequest) string {
	out := common.NewOutput(io.Discard, io.Discard)
	absPath, errResult := resolveValidatedPathWithRoots(out, req.readPath(), req.AllowedRoots, "path is empty")
	if errResult != "" {
		return ""
	}
	return absPath
}
