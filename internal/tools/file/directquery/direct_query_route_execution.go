package directquery

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/readtool"
)

// executeRoute executes the resolved direct route using file-package semantics.
func executeRoute(execCtx tools.ExecutionContext, route Route, detail string, depth int) string {
	output, _ := ExecuteWithObservation(execCtx, route, detail, depth)
	return output
}

// ExecuteWithObservation executes the resolved direct route and returns runtime observation facts.
func ExecuteWithObservation(execCtx tools.ExecutionContext, route Route, detail string, depth int) (string, *tools.RuntimeObservation) {
	switch route.Kind {
	case RouteRead:
		sections := executeDirectReadTargetsWithDetailSections(execCtx, route.targets, detail)
		return readtool.RenderReadExecutionSections(sections), readtool.MergeReadExecutionSectionObservations(sections)
	case RouteDirectory:
		if len(route.targets) == 0 {
			return "Error: path is not a directory", nil
		}
		return executeDirectListDirTarget(execCtx, route.targets[0], depth), nil
	default:
		return "", nil
	}
}
