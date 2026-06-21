package readtool

import (
	"encoding/json"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func resolveReadTargets(execCtx tools.ExecutionContext, rawTargets, rawPaths string, detail readDetailMode) ([]readRequest, *locator.Registry, string, error) {
	reg := execCtx.EffectiveLocatorRegistry()
	if rawTargets != "" {
		locs := reg.ResolveMulti(rawTargets)
		if len(locs) == 0 {
			return nil, nil, fmt.Sprintf("Error: no valid locator IDs found in targets: %s", rawTargets), nil
		}

		requests := make([]readRequest, 0, len(locs))
		for _, loc := range locs {
			requests = append(requests, buildReadRequestFromLocator(execCtx, loc, detail))
		}
		return requests, reg, "", nil
	}

	var paths []string
	if rawPaths != "" {
		if err := json.Unmarshal([]byte(rawPaths), &paths); err != nil {
			return nil, nil, fmt.Sprintf("Error: invalid paths format: %v", err), nil
		}
	}
	if len(paths) == 0 {
		return nil, nil, "Error: either paths or targets is required", nil
	}

	return buildReadRequestsFromPaths(paths, detail), reg, "", nil
}
