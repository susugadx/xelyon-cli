package file

import (
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

const (
	maxEntries           = 200
	maxRootDirsShown     = 8
	maxRootFilesShown    = 8
	maxRootSubtreesShown = 6
	maxNestedDirsShown   = 4
	maxNestedFilesShown  = 4
	maxNestedSubtrees    = 3
)

type listDirFileSummary struct {
	name string
	size int64
}

type listDirSection struct {
	relPath     string
	totalDirs   int
	totalFiles  int
	dirs        []string
	files       []listDirFileSummary
	moreDirs    int
	moreFiles   int
	subtrees    []*listDirSection
	moreSubtree int
	readErr     error
}

type listDirBudget struct {
	remainingEntries int
}

func summarizeListDir(dirPath, rootPath, filterRoot, relPath string, remain int, matcher *pathmatch.Matcher, fileFilter string, projectMap *repomap.ProjectMap, budget *listDirBudget, isRoot bool) *listDirSection {
	filterIdx := buildListDirFilterIndex(dirPath, rootPath, filterRoot, matcher, fileFilter, projectMap)
	tree := buildVisibleListDirTree(dirPath, rootPath, filterRoot, relPath, remain, matcher, fileFilter, filterIdx, isRoot)
	return summarizeVisibleListDirTree(tree, remain, budget, isRoot)
}

func summarizeVisibleListDirTree(tree *listDirVisibleNode, remain int, budget *listDirBudget, isRoot bool) *listDirSection {
	section := &listDirSection{relPath: tree.relPath, readErr: tree.readErr}
	if budget.remainingEntries <= 0 || tree.readErr != nil {
		return section
	}

	populateListDirSectionEntries(section, tree, budget, isRoot)

	if remain <= 1 || budget.remainingEntries <= 0 || len(tree.expandedDirs) == 0 {
		return section
	}

	section.subtrees, section.moreSubtree = summarizeListDirSubtrees(tree, remain, budget, isRoot)
	return section
}
