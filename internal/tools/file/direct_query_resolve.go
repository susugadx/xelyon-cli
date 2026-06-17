package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func resolveDirectQuery(execCtx tools.ExecutionContext, query string) (DirectQueryResolution, string) {
	input, ok := parseDirectQueryInput(query)
	if !ok {
		return DirectQueryResolution{}, "Error: direct query is empty"
	}
	return resolveDirectQueryInput(execCtx, input)
}

func resolveDirectQueryInput(execCtx tools.ExecutionContext, input directQueryInput) (DirectQueryResolution, string) {
	if len(input.Entries) == 0 {
		return DirectQueryResolution{}, "Error: direct query is empty"
	}

	targets := make([]DirectQueryTarget, 0, len(input.Entries))
	for _, entry := range input.Entries {
		target, errResult := resolveDirectQueryTarget(execCtx, entry)
		if errResult != "" {
			return DirectQueryResolution{}, errResult
		}
		targets = append(targets, target)
	}

	if len(targets) == 1 && targets[0].Kind == DirectQueryTargetDirectory {
		return DirectQueryResolution{
			Kind:    DirectQueryResolutionDirectory,
			Targets: targets,
		}, ""
	}

	for _, target := range targets {
		if target.Kind != DirectQueryTargetFile {
			return DirectQueryResolution{}, "Error: direct query cannot mix files and directories"
		}
	}

	return DirectQueryResolution{
		Kind:    DirectQueryResolutionFiles,
		Targets: targets,
	}, ""
}

func resolveDirectQueryTarget(execCtx tools.ExecutionContext, input directQueryEntryInput) (DirectQueryTarget, string) {
	if strings.TrimSpace(input.CleanedPath) == "" {
		return DirectQueryTarget{}, "Error: direct query target is empty"
	}

	resolvedPath, allowedRoots, info, errResult := resolveExistingDirectQueryPath(execCtx, input)
	if errResult != "" {
		return DirectQueryTarget{}, errResult
	}
	bypassIgnores := directQueryTargetBypassesIgnores(input)
	if info.IsDir() {
		if input.StartLine > 0 || input.EndLine > 0 {
			return DirectQueryTarget{}, "Error: direct directory query does not support line ranges"
		}
		return DirectQueryTarget{
			RawEntry:      input.CleanedPath,
			FilePath:      input.CleanedPath,
			ResolvedPath:  resolvedPath,
			AllowedRoots:  allowedRoots,
			WorkspaceRoot: directQueryWorkspaceRoot(resolvedPath, allowedRoots),
			BypassIgnores: bypassIgnores,
			Kind:          DirectQueryTargetDirectory,
		}, ""
	}

	return DirectQueryTarget{
		RawEntry:      normalizeDirectQueryRawEntry(input.CleanedPath, input.StartLine, input.EndLine),
		FilePath:      input.CleanedPath,
		ResolvedPath:  resolvedPath,
		AllowedRoots:  allowedRoots,
		WorkspaceRoot: directQueryWorkspaceRoot(resolvedPath, allowedRoots),
		BypassIgnores: bypassIgnores,
		StartLine:     input.StartLine,
		EndLine:       input.EndLine,
		Kind:          DirectQueryTargetFile,
	}, ""
}

func resolveExistingDirectQueryPath(execCtx tools.ExecutionContext, input directQueryEntryInput) (string, []string, os.FileInfo, string) {
	if strings.TrimSpace(input.CleanedPath) == "" {
		return "", nil, nil, "Error: direct query target is empty"
	}

	if input.ExplicitRelative {
		return resolveExistingExplicitRelativeDirectQueryPath(execCtx, input)
	}

	allowedRoots := directQueryAllowedRoots(execCtx)
	out := common.NewOutput(io.Discard, io.Discard)

	candidates := directQueryCandidates(input.CleanedPath, allowedRoots)
	if len(candidates) == 0 {
		candidates = []string{input.CleanedPath}
	}

	for _, candidate := range candidates {
		resolvedPath, errResult := resolveValidatedPathWithRoots(out, candidate, allowedRoots, "path is empty")
		if errResult != "" {
			continue
		}
		info, err := os.Stat(resolvedPath)
		if err != nil {
			continue
		}
		return resolvedPath, allowedRoots, info, ""
	}

	return "", nil, nil, "Error: direct path not found: " + input.RawEntry
}

func resolveExistingExplicitRelativeDirectQueryPath(execCtx tools.ExecutionContext, input directQueryEntryInput) (string, []string, os.FileInfo, string) {
	baseRoot := directQueryExplicitRelativeBase(execCtx)
	if baseRoot == "" {
		return "", nil, nil, "Error: direct path not found: " + input.RawEntry
	}

	allowedRoots := directQueryExplicitRelativeAllowedRoots(execCtx)
	out := common.NewOutput(io.Discard, io.Discard)
	candidate := filepath.Clean(filepath.Join(baseRoot, input.CleanedPath))
	resolvedPath, errResult := resolveValidatedPathWithRoots(out, candidate, allowedRoots, "path is empty")
	if errResult != "" {
		return "", nil, nil, errResult
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", nil, nil, "Error: direct path not found: " + input.RawEntry
	}
	return resolvedPath, allowedRoots, info, ""
}

func resolveExistingDirectReadTargets(execCtx tools.ExecutionContext, input directQueryInput) ([]DirectQueryTarget, bool) {
	targets, errResult := resolveDirectReadTargets(execCtx, input)
	if errResult != "" {
		return nil, false
	}
	return targets, true
}

func resolveDirectReadTargets(execCtx tools.ExecutionContext, input directQueryInput) ([]DirectQueryTarget, string) {
	if len(input.Entries) == 0 {
		return nil, "Error: direct query is empty"
	}

	targets := make([]DirectQueryTarget, 0, len(input.Entries))
	for _, entry := range input.Entries {
		target, errResult := resolveDirectQueryTarget(execCtx, entry)
		if errResult != "" {
			return nil, errResult
		}
		if target.Kind != DirectQueryTargetFile {
			return nil, "Error: direct read query cannot include directories"
		}
		targets = append(targets, target)
	}
	return targets, ""
}

func directQueryRoots(execCtx tools.ExecutionContext) []string {
	roots := make([]string, 0, 2)
	for _, root := range []string{execCtx.InvocationCWD, execCtx.ProjectMapRootPath} {
		root = normalizeWorkspaceRoot(root)
		if root == "" {
			continue
		}
		roots = appendUniqueString(roots, root)
	}
	return roots
}

func directQueryAllowedRoots(execCtx tools.ExecutionContext) []string {
	return directQueryRoots(execCtx)
}

func directQueryExplicitRelativeBase(execCtx tools.ExecutionContext) string {
	if root := normalizeWorkspaceRoot(execCtx.InvocationCWD); root != "" {
		return root
	}
	return normalizeWorkspaceRoot(execCtx.ProjectMapRootPath)
}

func directQueryExplicitRelativeAllowedRoots(execCtx tools.ExecutionContext) []string {
	roots := make([]string, 0, 2)
	if root := normalizeWorkspaceRoot(execCtx.ProjectMapRootPath); root != "" {
		roots = appendUniqueString(roots, root)
	}
	if root := normalizeWorkspaceRoot(execCtx.InvocationCWD); root != "" {
		roots = appendUniqueString(roots, root)
	}
	if len(roots) == 0 {
		if root := directQueryExplicitRelativeBase(execCtx); root != "" {
			roots = append(roots, root)
		}
	}
	return roots
}

func directQueryCandidates(filePath string, roots []string) []string {
	if filepath.IsAbs(filePath) {
		return []string{filepath.Clean(filePath)}
	}
	if len(roots) == 0 {
		return nil
	}

	candidates := make([]string, 0, len(roots))
	for _, root := range roots {
		candidates = append(candidates, filepath.Clean(filepath.Join(root, filePath)))
	}
	return candidates
}

func normalizeDirectQueryRawEntry(filePath string, startLine, endLine int) string {
	return formatReadRangeEntry(filepath.Clean(filePath), startLine, endLine)
}

func directQueryWorkspaceRoot(resolvedPath string, allowedRoots []string) string {
	resolvedPath = normalizeWorkspaceRoot(resolvedPath)
	for _, root := range allowedRoots {
		root = normalizeWorkspaceRoot(root)
		if root != "" && isPathWithinRoot(resolvedPath, root) {
			return root
		}
	}
	for _, root := range allowedRoots {
		root = normalizeWorkspaceRoot(root)
		if root != "" {
			return root
		}
	}
	return ""
}

func directQueryTargetBypassesIgnores(input directQueryEntryInput) bool {
	switch input.Syntax {
	case directQuerySyntaxExplicitPath, directQuerySyntaxPathCandidate:
		return true
	default:
		return false
	}
}
