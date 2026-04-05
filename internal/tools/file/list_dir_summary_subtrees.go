package file

import (
	"os"
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func summarizeListDirSubtrees(dirPath, rootPath, relPath string, remain int, matcher *pathmatch.Matcher, budget *listDirBudget, dirs []os.DirEntry) ([]*listDirSection, int) {
	_, _, subtreeLimit := listDirDisplayLimits(relPath == "")
	expandCount := minInt(len(dirs), subtreeLimit)

	subtrees := make([]*listDirSection, 0, expandCount)
	for i := 0; i < expandCount && budget.remainingEntries > 0; i++ {
		dirName := dirs[i].Name()
		childRelPath := joinListDirRelPath(relPath, dirName)
		childPath := filepath.Join(dirPath, dirName)
		subtrees = append(subtrees, summarizeListDir(childPath, rootPath, childRelPath, remain-1, matcher, budget, false))
	}

	moreSubtree := 0
	if len(dirs) > len(subtrees) {
		moreSubtree = len(dirs) - len(subtrees)
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
