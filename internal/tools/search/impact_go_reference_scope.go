package search

import "strings"

func structuredGoImpactFallbackReferenceSearchPath(opts SearchOptions) string {
	opts = structuredImpactSemanticReferenceFilterOptions(opts)

	filePattern := cleanStructuredGoFilePattern(opts.FilePattern)
	if filePattern != "" {
		if pathHint, ok := structuredGoImpactPathHintForFilePattern(opts, filePattern); ok && pathHint != "" {
			return pathHint
		}
	}
	if target := structuredImpactSearchTargetPath(opts); target != "" {
		return target
	}
	return strings.TrimSpace(opts.Path)
}
