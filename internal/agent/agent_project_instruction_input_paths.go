package agent

import (
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
	paths := dedupeProjectMapPriorityPaths(projectMapPriorityPathsFromInput(cwd, rootPath, candidates, len(candidates)))
	if len(paths) == 0 {
		return nil
	}
	return paths
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
