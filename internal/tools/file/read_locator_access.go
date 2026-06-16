package file

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type locatorReadAccess struct {
	resolvedPath string
	allowedRoots []string
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
