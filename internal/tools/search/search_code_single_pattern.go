package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func executeSinglePattern(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) string {
	return executeSinglePatternDetailed(cache, pattern, opts).Output
}

func executeSinglePatternWithTrace(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) (string, searchRouteTrace) {
	result := executeSinglePatternDetailed(cache, pattern, opts)
	return result.Output, result.Route
}

func executeSinglePatternDetailed(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) singlePatternExecution {
	route := planSearchRoute(pattern, opts)
	cacheKey := buildSearchCacheKeyWithRoute(opts, route.cacheSignature())
	if cache != nil {
		if cached, ok := cache.GetSearch(pattern, cacheKey); ok {
			bundle := loadSinglePatternBundle(pattern, cacheKey)
			affectedFiles := loadSinglePatternAffectedFiles(pattern, cacheKey)
			if len(affectedFiles) == 0 {
				affectedFiles = deriveAffectedFilesFromCachedResult(bundle, cached, opts)
			}
			return singlePatternExecution{
				Pattern:       pattern,
				Output:        cached,
				Route:         route,
				Bundle:        bundle,
				AffectedFiles: affectedFiles,
			}
		}
	}

	if route.InitialLane == searchLaneSymbol {
		route.SymbolAttempted = true
		resolver := resolverForLanguage(route.Language)
		if resolver != nil {
			resolved := resolver.Resolve(route.SymbolQuery, opts)
			switch resolved.Status {
			case symbolResolveSingle:
				route.FinalLane = searchLaneSymbol
				route.SymbolResolved = true
				resolved.Bundle = attachBundleRoute(resolved.Bundle, route)
				affectedFiles := collectSymbolBundleAffectedFiles(resolved.Bundle, opts)
				if cache != nil {
					cache.SetSearch(pattern, cacheKey, resolved.Output, affectedFiles)
					storeSinglePatternBundle(pattern, cacheKey, resolved.Bundle)
					storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
				}
				return singlePatternExecution{
					Pattern:       pattern,
					Output:        resolved.Output,
					Route:         route,
					Bundle:        resolved.Bundle,
					AffectedFiles: affectedFiles,
				}
			case symbolResolveMultiple:
				route.FinalLane = searchLaneSymbol
				route.SymbolResolved = true
				affectedFiles := resolved.AffectedFiles
				if len(affectedFiles) == 0 {
					affectedFiles = deriveAffectedFilesFromCachedResult(nil, resolved.Output, opts)
				}
				if cache != nil {
					cache.SetSearch(pattern, cacheKey, resolved.Output, affectedFiles)
					storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
				}
				return singlePatternExecution{
					Pattern:       pattern,
					Output:        resolved.Output,
					Route:         route,
					AffectedFiles: affectedFiles,
				}
			case symbolResolveNone:
				route.SymbolResolved = false
			}
		}
		if route.FallbackLane != "" {
			route.FallbackUsed = true
			route.FinalLane = route.FallbackLane
		}
	}
	if route.FinalLane == "" {
		route.FinalLane = route.InitialLane
	}

	textOpts := opts
	textOpts.IsRegex = route.textIsRegex()
	output, useRipgrep, warnings, err := executeSearch(pattern, textOpts)
	if err != nil {
		return singlePatternExecution{Pattern: pattern, Output: fmt.Sprintf("Error: %v", err), Route: route}
	}

	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output, 0)
	} else {
		results = parseGrepOutput(output, 0)
	}
	results = filterResultsByOptions(results, textOpts)
	reclassifyWithAST(results, pattern, textOpts.IsRegex)

	if len(results) == 0 {
		if len(warnings) > 0 {
			return singlePatternExecution{Pattern: pattern, Output: strings.Join(warnings, "\n") + "\nNo matches found", Route: route}
		}
		return singlePatternExecution{Pattern: pattern, Output: "No matches found", Route: route}
	}

	if opts.OutputMode == "manifest" {
		sortResultsByPriority(results)
		detectBlocksWithCache(cache, results)
		formatted := formatManifestResults(results, opts.LocatorRegistry)
		finalOutput := formatted
		if len(warnings) > 0 {
			finalOutput = strings.Join(warnings, "\n") + "\n" + formatted
		}

		if cache != nil {
			affectedFiles := collectFilePaths(results, opts)
			cache.SetSearch(pattern, cacheKey, finalOutput, affectedFiles)
			storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
			return singlePatternExecution{Pattern: pattern, Output: finalOutput, Route: route, AffectedFiles: affectedFiles}
		}
		return singlePatternExecution{Pattern: pattern, Output: finalOutput, Route: route, AffectedFiles: collectFilePaths(results, opts)}
	}

	results = mergeContextLines(results)
	sortResultsByPriority(results)

	results, truncated := truncateToTokenBudget(results, opts.TokenBudget, false)

	detectBlocksWithCache(cache, results)

	formatted := formatSearchResults(results, truncated, opts.TokenBudget, opts.LocatorRegistry)
	finalOutput := formatted
	if len(warnings) > 0 {
		finalOutput = strings.Join(warnings, "\n") + "\n" + formatted
	}
	finalOutput += lineRangeHint

	affectedFiles := collectFilePaths(results, opts)
	if cache != nil {
		cache.SetSearch(pattern, cacheKey, finalOutput, affectedFiles)
		storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
	}

	return singlePatternExecution{Pattern: pattern, Output: finalOutput, Route: route, AffectedFiles: affectedFiles}
}
