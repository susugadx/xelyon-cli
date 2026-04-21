package agent

const projectMapFocusMaxPaths = 5

func extractProjectMapFocusPaths(cwd, rootPath, input string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	paths := dedupeProjectMapPriorityPaths(projectMapPriorityPathsFromInput(cwd, rootPath, extractProjectMapPathsFromInput(input), limit))
	if len(paths) == 0 {
		return nil
	}
	return paths
}
