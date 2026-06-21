package agent

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func projectInstructionInputPathsForAgent(agent *Agent, input string) []string {
	if agent == nil || strings.TrimSpace(input) == "" {
		return nil
	}
	rootPath, ok := projectInstructionInputRootPath(agent)
	if !ok {
		return nil
	}
	return extractProjectInstructionInputPaths(agent.invocationCWD(), rootPath, input)
}

func extractProjectInstructionInputPaths(cwd, rootPath, input string) []string {
	candidates := extractProjectMapPathsFromInput(input)
	if len(candidates) == 0 {
		return nil
	}
	paths := dedupeProjectMapPriorityPaths(projectInstructionPriorityPathsFromInput(cwd, rootPath, candidates, len(candidates)))
	if len(paths) == 0 {
		return nil
	}
	return paths
}

func projectInstructionPriorityPathsFromInput(cwd, rootPath string, candidates []string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	paths := make([]string, 0, min(len(candidates), limit))
	for _, candidate := range candidates {
		path, ok := resolveProjectInstructionInputCandidate(cwd, rootPath, candidate)
		if !ok {
			continue
		}
		paths = append(paths, path)
		if len(paths) >= limit {
			break
		}
	}
	return paths
}

func resolveProjectInstructionInputCandidate(cwd, rootPath, candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || strings.TrimSpace(rootPath) == "" {
		return "", false
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
		return "", false
	}
	normalized := filepath.FromSlash(candidate)
	if hasProjectInstructionParentTraversalSegment(normalized) {
		return "", false
	}

	if existingPath, ok := resolveProjectMapInputCandidate(cwd, rootPath, candidate); ok {
		return existingPath, true
	}
	if filepath.IsAbs(normalized) || isWindowsAbsoluteProjectMapPath(candidate) {
		return "", false
	}

	cleaned := filepath.Clean(normalized)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", false
	}
	return resolveMissingProjectInstructionInputCandidate(cwd, rootPath, cleaned)
}

func resolveMissingProjectInstructionInputCandidate(cwd, rootPath, cleaned string) (string, bool) {
	if strings.TrimSpace(cwd) == "" {
		return canonicalizeProjectMapPriorityPath(rootPath, filepath.Join(rootPath, cleaned))
	}
	sessionAbs := filepath.Clean(filepath.Join(cwd, cleaned))
	if path, ok := canonicalizeProjectMapPriorityPath(rootPath, sessionAbs); ok {
		return path, true
	}
	return canonicalizeProjectMapPriorityPath(rootPath, filepath.Join(rootPath, cleaned))
}

func hasProjectInstructionParentTraversalSegment(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func projectInstructionInputRootPath(agent *Agent) (string, bool) {
	if agent == nil {
		return "", false
	}
	if bundle := agent.projectInstructionBundleIfLoaded(); bundle != nil {
		if rootPath, ok := bundle.ProjectRootPath(); ok {
			return rootPath, true
		}
		if rootPath := strings.TrimSpace(bundle.RootPath); rootPath != "" {
			return rootPath, true
		}
	}
	cwd := strings.TrimSpace(agent.invocationCWD())
	if cwd == "" {
		return "", false
	}
	if rootPath, ok := config.ResolveProjectInstructionProjectRootForDir(agent.cfg(), cwd); ok {
		return rootPath, true
	}
	return cwd, true
}
