package listtool

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy"
)

// listDirFilterIndex owns request-local descendant visibility for filtered
// directory summaries. ProjectMap entries may seed likely-visible ancestors,
// but a filesystem walk remains authoritative because ProjectMap snapshots can
// be truncated under normal large-repo budgets.
type listDirFilterIndex struct {
	rootPath    string
	visibleDirs map[string]struct{}
}

func buildListDirFilterIndex(dirPath, rootPath, filterRoot string, matcher *pathmatch.Matcher, fileFilter string, projectMap *repomap.ProjectMap) *listDirFilterIndex {
	fileFilter = strings.TrimSpace(fileFilter)
	if fileFilter == "" {
		return nil
	}

	index := &listDirFilterIndex{
		rootPath:    pathpolicy.NormalizeWorkspaceRoot(dirPath),
		visibleDirs: make(map[string]struct{}),
	}
	if index.rootPath == "" {
		return index
	}

	seedListDirFilterIndexFromProjectMap(index, rootPath, filterRoot, matcher, fileFilter, projectMap)
	populateListDirFilterIndexByWalk(index, rootPath, filterRoot, matcher, fileFilter)
	return index
}

func (idx *listDirFilterIndex) hasVisibleDir(path string) bool {
	if idx == nil {
		return false
	}
	_, ok := idx.visibleDirs[pathpolicy.NormalizeWorkspaceRoot(path)]
	return ok
}

func seedListDirFilterIndexFromProjectMap(idx *listDirFilterIndex, rootPath, filterRoot string, matcher *pathmatch.Matcher, fileFilter string, projectMap *repomap.ProjectMap) {
	if idx == nil || projectMap == nil {
		return
	}

	projectRoot := pathpolicy.NormalizeWorkspaceRoot(projectMap.RootPath)
	if projectRoot == "" || len(projectMap.Files) == 0 {
		return
	}
	if !pathpolicy.IsPathWithinRoot(idx.rootPath, projectRoot) {
		return
	}

	for _, entry := range projectMap.Files {
		if entry == nil || strings.TrimSpace(entry.Path) == "" {
			continue
		}
		absPath := pathpolicy.NormalizeWorkspaceRoot(filepath.Join(projectRoot, filepath.FromSlash(entry.Path)))
		if !listDirFilterCandidateMatches(absPath, rootPath, filterRoot, matcher, fileFilter) {
			continue
		}
		idx.addAncestors(absPath)
	}
}

func populateListDirFilterIndexByWalk(idx *listDirFilterIndex, rootPath, filterRoot string, matcher *pathmatch.Matcher, fileFilter string) {
	if idx == nil {
		return
	}

	_ = filepath.WalkDir(idx.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path != idx.rootPath && listDirFilterPathIgnored(rootPath, matcher, path, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !listDirFilterCandidateMatches(path, rootPath, filterRoot, matcher, fileFilter) {
			return nil
		}
		idx.addAncestors(path)
		return nil
	})
}

func listDirFilterCandidateMatches(path, rootPath, filterRoot string, matcher *pathmatch.Matcher, fileFilter string) bool {
	path = pathpolicy.NormalizeWorkspaceRoot(path)
	if path == "" {
		return false
	}
	if listDirFilterPathIgnored(rootPath, matcher, path, false) {
		return false
	}
	if !filefilter.Matches(filefilter.WorkspaceRelativePath(path, filterRoot), fileFilter) {
		return false
	}
	return true
}

func listDirFilterPathIgnored(rootPath string, matcher *pathmatch.Matcher, path string, isDir bool) bool {
	if matcher == nil {
		return false
	}
	rootPath = pathpolicy.NormalizeWorkspaceRoot(rootPath)
	path = pathpolicy.NormalizeWorkspaceRoot(path)
	if rootPath == "" || path == "" {
		return false
	}
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		return false
	}
	return matcher.Match(filepath.ToSlash(relPath), isDir)
}

func (idx *listDirFilterIndex) addAncestors(filePath string) {
	if idx == nil {
		return
	}

	rootPath := pathpolicy.NormalizeWorkspaceRoot(idx.rootPath)
	filePath = pathpolicy.NormalizeWorkspaceRoot(filePath)
	if rootPath == "" || filePath == "" || !pathpolicy.IsPathWithinRoot(filePath, rootPath) {
		return
	}

	for dirPath := pathpolicy.NormalizeWorkspaceRoot(filepath.Dir(filePath)); dirPath != "" && dirPath != rootPath; dirPath = pathpolicy.NormalizeWorkspaceRoot(filepath.Dir(dirPath)) {
		idx.visibleDirs[dirPath] = struct{}{}
		if !pathpolicy.IsPathWithinRoot(dirPath, rootPath) {
			break
		}
	}
}
