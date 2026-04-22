package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func ensureProjectMap(agent *Agent, rootPath string, ignorePatterns []string, ignoreKey string) (*repomap.ProjectMap, bool) {
	if agent == nil {
		return nil, false
	}

	if canReuseExistingProjectMap(agent, rootPath, ignoreKey) && isProjectMapStateUnchanged(agent, rootPath) {
		return agent.projectMap, false
	}

	pm, err := buildProjectMap(rootPath, ignorePatterns)
	if err != nil {
		yellow.Fprintf(agent.output(), "⚠️ ProjectMap build failed: %v\n", err)
		return nil, false
	}

	applyProjectMapState(agent, pm, rootPath, ignorePatterns, ignoreKey)
	return pm, true
}

func canReuseExistingProjectMap(agent *Agent, rootPath, ignoreKey string) bool {
	return !agent.projectMapDirty &&
		agent.projectMap != nil &&
		agent.projectMapRootPath == rootPath &&
		agent.projectMapIgnoreKey == ignoreKey
}

func isProjectMapStateUnchanged(agent *Agent, rootPath string) bool {
	stateKey := currentProjectMapStateKey(agent, rootPath)
	return stateKey != "" && agent.projectMapStateKey == stateKey
}

func buildProjectMap(rootPath string, ignorePatterns []string) (*repomap.ProjectMap, error) {
	pm := repomap.NewProjectMap(rootPath, 0, ignorePatterns...)
	if err := pm.Build(); err != nil {
		return nil, err
	}
	return pm, nil
}

func currentProjectMapStateKey(agent *Agent, rootPath string) string {
	if gitKey := resolveProjectMapStateKeyFromGit(rootPath); gitKey != "" {
		return gitKey
	}

	if watchKey := resolveProjectMapStateKeyFromWatch(agent, rootPath); watchKey != "" {
		return watchKey
	}

	return resolveProjectMapStateKeyFromRootStat(rootPath)
}

func projectMapWatchDirs(agent *Agent) []string {
	if agent == nil || len(agent.projectMapWatchDirs) == 0 {
		return []string{"."}
	}

	dirs := make([]string, len(agent.projectMapWatchDirs))
	copy(dirs, agent.projectMapWatchDirs)
	return dirs
}

func projectMapIgnorePatterns(agent *Agent) []string {
	if agent == nil || agent.projectMapIgnoreKey == "" {
		return nil
	}
	return pathmatch.NormalizePatterns(strings.Split(agent.projectMapIgnoreKey, "\x00"))
}
