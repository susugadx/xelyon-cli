package file

import "path/filepath"

func summarizeListDirSubtrees(tree *listDirVisibleNode, remain int, budget *listDirBudget, isRoot bool) ([]*listDirSection, int) {
	subtrees := make([]*listDirSection, 0, len(tree.expandedDirs))
	for i := 0; i < len(tree.expandedDirs) && budget.remainingEntries > 0; i++ {
		subtrees = append(subtrees, summarizeVisibleListDirTree(tree.expandedDirs[i], remain-1, budget, false))
	}

	moreSubtree := 0
	if len(tree.dirs) > len(subtrees) {
		moreSubtree = len(tree.dirs) - len(subtrees)
	}
	return subtrees, moreSubtree
}

func listDirDisplayLimits(isRoot bool) (dirLimit, fileLimit, subtreeLimit int) {
	dirLimit = maxNestedDirsShown
	fileLimit = maxNestedFilesShown
	subtreeLimit = maxNestedSubtrees
	if isRoot {
		dirLimit = maxRootDirsShown
		fileLimit = maxRootFilesShown
		subtreeLimit = maxRootSubtreesShown
	}
	return dirLimit, fileLimit, subtreeLimit
}

func joinListDirRelPath(parent, name string) string {
	joined := filepath.ToSlash(filepath.Join(parent, name))
	if joined == "." || joined == "" {
		joined = name
	}
	return joined + "/"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
