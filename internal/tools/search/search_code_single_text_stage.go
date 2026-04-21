package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type textSearchStageResult struct {
	Results  []SearchResult
	Warnings []string
}

func executeSinglePatternTextStage(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext) singlePatternExecution {
	route := finalizeSinglePatternRoute(ctx.Route)
	textOpts := ctx.Opts
	textOpts.IsRegex = route.textIsRegex()

	stageResult, err := runTextSearchStage(ctx.Pattern, textOpts)
	if err != nil {
		return singlePatternExecution{Pattern: ctx.Pattern, Output: fmt.Sprintf("Error: %v", err), Route: route}
	}

	if len(stageResult.Results) == 0 {
		return singlePatternExecution{
			Pattern: ctx.Pattern,
			Output:  formatNoMatchOutput(stageResult.Warnings),
			Route:   route,
		}
	}

	return formatSinglePatternTextStage(cache, ctx, route, textOpts, stageResult)
}

func finalizeSinglePatternRoute(route searchRouteTrace) searchRouteTrace {
	if route.FinalLane == "" {
		route.FinalLane = route.InitialLane
	}
	return route
}

func runTextSearchStage(pattern string, textOpts SearchOptions) (textSearchStageResult, error) {
	output, useRipgrep, warnings, err := executeSearch(pattern, textOpts)
	if err != nil {
		return textSearchStageResult{}, err
	}

	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output, 0)
	} else {
		results = parseGrepOutput(output, 0)
	}
	results = filterResultsByOptions(results, textOpts)
	reclassifyWithAST(results, pattern, textOpts.IsRegex, textOpts)
	return textSearchStageResult{Results: results, Warnings: warnings}, nil
}

func formatSinglePatternTextStage(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, route searchRouteTrace, textOpts SearchOptions, stage textSearchStageResult) singlePatternExecution {
	results := stage.Results
	if ctx.Opts.OutputMode == "manifest" {
		sortResultsByPriority(results)
		detectBlocksWithCache(cache, results, textOpts)
		finalOutput := withSearchWarnings(formatManifestResultsWithOptions(results, ctx.Opts.LocatorRegistry, ctx.Opts), stage.Warnings)
		affectedFiles := collectFilePaths(results, ctx.Opts)
		writeSinglePatternSearchCache(cache, ctx, finalOutput, affectedFiles)
		return singlePatternExecution{
			Pattern:       ctx.Pattern,
			Output:        finalOutput,
			Route:         route,
			AffectedFiles: affectedFiles,
		}
	}

	results = mergeContextLines(results)
	sortResultsByPriority(results)
	results, truncated := truncateToTokenBudget(results, ctx.Opts.TokenBudget, false)
	detectBlocksWithCache(cache, results, textOpts)

	formatted := formatSearchResultsWithOptions(results, truncated, ctx.Opts.TokenBudget, ctx.Opts.LocatorRegistry, ctx.Opts)
	finalOutput := withSearchWarnings(formatted, stage.Warnings) + lineRangeHint
	affectedFiles := collectFilePaths(results, ctx.Opts)
	writeSinglePatternSearchCache(cache, ctx, finalOutput, affectedFiles)

	return singlePatternExecution{
		Pattern:       ctx.Pattern,
		Output:        finalOutput,
		Route:         route,
		AffectedFiles: affectedFiles,
	}
}

func formatNoMatchOutput(warnings []string) string {
	if len(warnings) == 0 {
		return "No matches found"
	}
	return strings.Join(warnings, "\n") + "\nNo matches found"
}

func withSearchWarnings(output string, warnings []string) string {
	if len(warnings) == 0 {
		return output
	}
	return strings.Join(warnings, "\n") + "\n" + output
}
