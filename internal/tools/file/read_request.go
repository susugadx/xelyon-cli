package file

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

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
	RawEntry     string
	FilePath     string
	ResolvedPath string
	AllowedRoots []string
	StartLine    int
	EndLine      int
	Source       readRequestSource
	Locator      *locator.Location
	Detail       readDetailMode
	RangeEntry   string
}

type locatorReadAccess struct {
	resolvedPath string
	allowedRoots []string
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
	return fmt.Sprintf("%s\x00%d\x00%d", req.readPath(), req.StartLine, req.EndLine)
}

func (req readRequest) readPath() string {
	if strings.TrimSpace(req.ResolvedPath) != "" {
		return req.ResolvedPath
	}
	return req.FilePath
}

func buildReadRequestFromLocator(execCtx tools.ExecutionContext, loc locator.Location, detail readDetailMode) readRequest {
	access := resolveLocatorReadAccess(execCtx, loc)
	req := readRequest{
		RawEntry:     loc.FilePath,
		FilePath:     loc.FilePath,
		ResolvedPath: access.resolvedPath,
		AllowedRoots: access.allowedRoots,
		Source:       readRequestSourceLocator,
		Detail:       detail,
		RangeEntry:   loc.FilePath,
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
	return req
}

func resolveLocatorReadAccess(execCtx tools.ExecutionContext, loc locator.Location) locatorReadAccess {
	resolvedPath := strings.TrimSpace(loc.ResolvedPath)
	if resolvedPath != "" {
		resolvedPath = normalizeWorkspaceRoot(resolvedPath)
		return locatorReadAccess{
			resolvedPath: resolvedPath,
			allowedRoots: resolvedLocatorReadRoots(execCtx, loc, resolvedPath),
		}
	}

	filePath := loc.FilePath
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return locatorReadAccess{}
	}

	if filepath.IsAbs(filePath) {
		return locatorReadAccess{
			resolvedPath: filepath.Clean(filePath),
			allowedRoots: workspaceLocatorReadRoots(execCtx),
		}
	}

	if root := normalizeWorkspaceRoot(execCtx.ProjectMapRootPath); root != "" {
		return locatorReadAccess{
			resolvedPath: filepath.Clean(filepath.Join(root, filePath)),
			allowedRoots: []string{root},
		}
	}

	if cwd := normalizeWorkspaceRoot(execCtx.InvocationCWD); cwd != "" {
		return locatorReadAccess{
			resolvedPath: filepath.Clean(filepath.Join(cwd, filePath)),
			allowedRoots: []string{cwd},
		}
	}

	return locatorReadAccess{}
}

func workspaceLocatorReadRoots(execCtx tools.ExecutionContext) []string {
	roots := make([]string, 0, 2)
	for _, candidate := range []string{execCtx.ProjectMapRootPath, execCtx.InvocationCWD} {
		candidate = normalizeWorkspaceRoot(candidate)
		if candidate == "" {
			continue
		}
		roots = appendUniqueString(roots, candidate)
	}
	return roots
}

func resolvedLocatorReadRoots(execCtx tools.ExecutionContext, loc locator.Location, resolvedPath string) []string {
	roots := workspaceLocatorReadRoots(execCtx)
	inferredRoot := inferLocatorWorkspaceRoot(loc.FilePath, resolvedPath)
	if inferredRoot == "" || !isAllowedInferredLocatorRoot(roots, inferredRoot) {
		return roots
	}
	return appendUniqueString(roots, inferredRoot)
}

func inferLocatorWorkspaceRoot(displayPath, resolvedPath string) string {
	displayPath = strings.TrimSpace(displayPath)
	resolvedPath = normalizeWorkspaceRoot(resolvedPath)
	if displayPath == "" || resolvedPath == "" || filepath.IsAbs(displayPath) {
		return ""
	}
	relPath := filepath.Clean(filepath.FromSlash(displayPath))
	if relPath == "." || relPath == "" {
		return ""
	}
	root := resolvedPath
	for range strings.Split(relPath, string(filepath.Separator)) {
		root = filepath.Dir(root)
	}
	root = normalizeWorkspaceRoot(root)
	if root == "" {
		return ""
	}
	if filepath.Clean(filepath.Join(root, relPath)) != resolvedPath {
		return ""
	}
	return root
}

func isAllowedInferredLocatorRoot(currentRoots []string, inferredRoot string) bool {
	if inferredRoot == "" {
		return false
	}
	realInferred := evaluateWorkspaceRoot(inferredRoot)
	for _, root := range currentRoots {
		if isPathWithinRoot(root, inferredRoot) {
			return true
		}
		if realRoot := evaluateWorkspaceRoot(root); isPathWithinRoot(realRoot, realInferred) {
			return true
		}
	}
	return false
}
