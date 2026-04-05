package file

import (
	"encoding/json"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type readRequestSource string

const (
	readRequestSourcePathWhole readRequestSource = "pathWhole"
	readRequestSourcePathRange readRequestSource = "pathRange"
	readRequestSourceLocator   readRequestSource = "locator"
)

type readRequest struct {
	RawEntry   string
	FilePath   string
	StartLine  int
	EndLine    int
	Source     readRequestSource
	Locator    *locator.Location
	Detail     readDetailMode
	RangeEntry string
}

func buildReadRequestsFromPaths(paths []string, detail readDetailMode) []readRequest {
	requests := make([]readRequest, 0, len(paths))
	for _, entry := range paths {
		path, startLine, endLine := parsePath(entry)
		source := readRequestSourcePathWhole
		if startLine > 0 || endLine > 0 {
			source = readRequestSourcePathRange
		}
		requests = append(requests, readRequest{
			RawEntry:  entry,
			FilePath:  path,
			StartLine: startLine,
			EndLine:   endLine,
			Source:    source,
			Detail:    detail,
		})
	}
	return requests
}

func resolveReadTargets(execCtx tools.ExecutionContext, rawTargets, rawPaths string, detail readDetailMode) ([]readRequest, *locator.Registry, string, error) {
	reg := execCtx.EffectiveLocatorRegistry()
	if rawTargets != "" {
		locs := reg.ResolveMulti(rawTargets)
		if len(locs) == 0 {
			return nil, nil, fmt.Sprintf("Error: no valid locator IDs found in targets: %s", rawTargets), nil
		}

		requests := make([]readRequest, 0, len(locs))
		for _, loc := range locs {
			req := readRequest{
				RawEntry:   loc.FilePath,
				FilePath:   loc.FilePath,
				Source:     readRequestSourceLocator,
				Detail:     detail,
				RangeEntry: loc.FilePath,
			}
			locCopy := loc
			req.Locator = &locCopy
			if loc.Line > 0 {
				req.StartLine = loc.Line
				if loc.EndLine > 0 {
					req.EndLine = loc.EndLine
				} else {
					req.EndLine = loc.Line
				}
			}
			requests = append(requests, req)
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

func validateReadRequests(requests []readRequest) string {
	return validateReadRequestCount(len(requests))
}

func formatReadRangeEntry(path string, startLine, endLine int) string {
	switch {
	case startLine > 0 && endLine > 0:
		return fmt.Sprintf("%s:%d-%d", path, startLine, endLine)
	case startLine > 0:
		return fmt.Sprintf("%s:%d", path, startLine)
	default:
		return path
	}
}

func dedupeReadRequestKey(req readRequest) string {
	return fmt.Sprintf("%s\x00%d\x00%d", req.FilePath, req.StartLine, req.EndLine)
}
