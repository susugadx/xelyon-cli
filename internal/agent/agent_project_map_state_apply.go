package agent

import "github.com/susugadx/xelyon-cli/internal/repomap"

func applyProjectMapState(agent *Agent, pm *repomap.ProjectMap, rootPath string, ignorePatterns []string, ignoreKey string) {
	agent.projectMap = pm
	agent.projectMapRootPath = rootPath
	agent.projectMapIgnoreKey = ignoreKey
	agent.projectMapWatchDirs = resolveProjectMapWatchDirs(rootPath, ignorePatterns)
	agent.projectMapStateKey = currentProjectMapStateKey(agent, rootPath)
}

func resolveProjectMapWatchDirs(rootPath string, ignorePatterns []string) []string {
	if isGitProjectMapAvailable(rootPath) {
		return nil
	}
	return collectProjectMapWatchDirs(rootPath, ignorePatterns)
}
