package gathercontext

import "github.com/susugadx/xelyon-cli/internal/tools"

func mergePrefetchObservations(observations []*tools.RuntimeObservation) *tools.RuntimeObservation {
	return tools.MergeRuntimeObservations(observations...)
}
