package file

import "github.com/susugadx/xelyon-cli/internal/tools"

// ExecuteGatherContextDirectRoute executes the resolved direct route using file-package semantics.
func ExecuteGatherContextDirectRoute(execCtx tools.ExecutionContext, route GatherContextDirectRoute, detail string, depth int) string {
	output, _ := ExecuteGatherContextDirectRouteWithObservation(execCtx, route, detail, depth)
	return output
}

// ExecuteGatherContextDirectRouteWithObservation executes the resolved direct route and returns runtime observation facts.
func ExecuteGatherContextDirectRouteWithObservation(execCtx tools.ExecutionContext, route GatherContextDirectRoute, detail string, depth int) (string, *tools.RuntimeObservation) {
	switch route.Kind {
	case GatherContextDirectRouteRead:
		sections := ExecuteDirectReadTargetsWithDetailSections(execCtx, route.targets, detail)
		return RenderReadExecutionSections(sections), MergeReadExecutionSectionObservations(sections)
	case GatherContextDirectRouteDirectory:
		if len(route.targets) == 0 {
			return "Error: path is not a directory", nil
		}
		return ExecuteDirectListDirTarget(execCtx, route.targets[0], depth), nil
	default:
		return "", nil
	}
}
