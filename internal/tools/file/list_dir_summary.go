package file

import (
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
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

func summarizeListDir(dirPath, rootPath, relPath string, remain int, matcher *pathmatch.Matcher, budget *listDirBudget, isRoot bool) *listDirSection {
	section := &listDirSection{relPath: relPath}
	if budget.remainingEntries <= 0 {
		return section
	}

	dirs, files, err := readVisibleListDirEntries(dirPath, rootPath, matcher)
	if err != nil {
		section.readErr = err
		return section
	}

	populateListDirSectionEntries(section, dirs, files, budget, isRoot)

	if remain <= 1 || budget.remainingEntries <= 0 || len(dirs) == 0 {
		return section
	}

	section.subtrees, section.moreSubtree = summarizeListDirSubtrees(dirPath, rootPath, relPath, remain, matcher, budget, dirs)
	return section
}
