package file

import "github.com/susugadx/xelyon-cli/internal/tools"

func resolveImplicitDirectFileQuery(execCtx tools.ExecutionContext, query string) ([]DirectQueryTarget, bool) {
	input, ok := parseDirectQueryInput(query)
	if !ok || !inputHasOnlyNamedBareFileCandidates(input) {
		return nil, false
	}
	return resolveExistingDirectReadTargets(execCtx, input)
}
