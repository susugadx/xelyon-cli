package listtool

import (
	"os"
)

func populateListDirSectionEntries(section *listDirSection, tree *listDirVisibleNode, budget *listDirBudget, isRoot bool) {
	section.totalDirs = len(tree.dirs)
	section.totalFiles = len(tree.files)
	budget.remainingEntries -= len(tree.dirs) + len(tree.files)

	dirLimit, fileLimit, _ := listDirDisplayLimits(isRoot)
	section.dirs, section.moreDirs = summarizeDirNames(tree.dirs, dirLimit)
	section.files, section.moreFiles = summarizeFileNames(tree.files, fileLimit)
}

func summarizeDirNames(entries []listDirVisibleDir, limit int) ([]string, int) {
	shown := minInt(len(entries), limit)
	result := make([]string, 0, shown)
	for i := 0; i < shown; i++ {
		result = append(result, entries[i].name+"/")
	}
	if len(entries) <= shown {
		return result, 0
	}
	return result, len(entries) - shown
}

func summarizeFileNames(entries []os.DirEntry, limit int) ([]listDirFileSummary, int) {
	shown := minInt(len(entries), limit)
	result := make([]listDirFileSummary, 0, shown)
	for i := 0; i < shown; i++ {
		entry := entries[i]
		summary := listDirFileSummary{name: entry.Name()}
		if info, err := entry.Info(); err == nil {
			summary.size = info.Size()
		}
		result = append(result, summary)
	}
	if len(entries) <= shown {
		return result, 0
	}
	return result, len(entries) - shown
}
