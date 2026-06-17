package search

import "github.com/susugadx/xelyon-cli/internal/tools"

func tryStructuredImpactSearchResultForIntent(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	for _, spec := range structuredImpactLanguageSpecs() {
		if result, ok := spec.trySearchResultForIntent(cache, opts); ok {
			return result, true
		}
	}
	return structuredImpactExecutionResult{}, false
}
