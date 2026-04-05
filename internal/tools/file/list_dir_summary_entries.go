package file

import (
	"os"
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func readVisibleListDirEntries(dirPath, rootPath string, matcher *pathmatch.Matcher) ([]os.DirEntry, []os.DirEntry, error) {
	rawEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, nil, err
	}

	var dirs []os.DirEntry
	var files []os.DirEntry
	for _, entry := range rawEntries {
		childPath := filepath.Join(dirPath, entry.Name())
		childRelPath, err := filepath.Rel(rootPath, childPath)
		if err == nil && matcher != nil && matcher.Match(filepath.ToSlash(childRelPath), entry.IsDir()) {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, entry)
			continue
		}
		files = append(files, entry)
	}
	return dirs, files, nil
}

func populateListDirSectionEntries(section *listDirSection, dirs, files []os.DirEntry, budget *listDirBudget, isRoot bool) {
	section.totalDirs = len(dirs)
	section.totalFiles = len(files)
	budget.remainingEntries -= len(dirs) + len(files)

	dirLimit, fileLimit, _ := listDirDisplayLimits(isRoot)
	section.dirs, section.moreDirs = summarizeDirNames(dirs, dirLimit)
	section.files, section.moreFiles = summarizeFileNames(files, fileLimit)
}

func summarizeDirNames(entries []os.DirEntry, limit int) ([]string, int) {
	shown := minInt(len(entries), limit)
	result := make([]string, 0, shown)
	for i := 0; i < shown; i++ {
		result = append(result, entries[i].Name()+"/")
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
