package listtool

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy"
)

// listDirVisibleNode owns request-local directory visibility. Immediate entries
// are read once, descendant visibility comes from the filtered index, and only
// subtrees that can actually be rendered are materialized.
type listDirVisibleNode struct {
	relPath      string
	dirs         []listDirVisibleDir
	expandedDirs []*listDirVisibleNode
	files        []os.DirEntry
	readErr      error
}

type listDirVisibleDir struct {
	name    string
	relPath string
	absPath string
}

type listDirReadDirFunc func(string) ([]os.DirEntry, error)

type listDirScanResult struct {
	dirs    []listDirVisibleDir
	files   []os.DirEntry
	readErr error
}

type listDirVisibleTreeBuilder struct {
	rootPath   string
	filterRoot string
	matcher    *pathmatch.Matcher
	fileFilter string
	filterIdx  *listDirFilterIndex
	readDir    listDirReadDirFunc
	scanMemo   map[string]*listDirScanResult
}

func buildVisibleListDirTree(dirPath, rootPath, filterRoot, relPath string, remain int, matcher *pathmatch.Matcher, fileFilter string, filterIdx *listDirFilterIndex, isRoot bool) *listDirVisibleNode {
	return buildVisibleListDirTreeWithReader(dirPath, rootPath, filterRoot, relPath, remain, matcher, fileFilter, filterIdx, isRoot, os.ReadDir)
}

func buildVisibleListDirTreeWithReader(dirPath, rootPath, filterRoot, relPath string, remain int, matcher *pathmatch.Matcher, fileFilter string, filterIdx *listDirFilterIndex, isRoot bool, readDir listDirReadDirFunc) *listDirVisibleNode {
	builder := listDirVisibleTreeBuilder{
		rootPath:   rootPath,
		filterRoot: filterRoot,
		matcher:    matcher,
		fileFilter: strings.TrimSpace(fileFilter),
		filterIdx:  filterIdx,
		readDir:    readDir,
		scanMemo:   make(map[string]*listDirScanResult),
	}
	return builder.build(dirPath, relPath, remain, isRoot)
}

func (b *listDirVisibleTreeBuilder) build(dirPath, relPath string, remain int, isRoot bool) *listDirVisibleNode {
	scan := b.scanImmediate(dirPath, relPath)
	node := &listDirVisibleNode{
		relPath: relPath,
		files:   scan.files,
		readErr: scan.readErr,
	}
	if scan.readErr != nil {
		return node
	}

	node.dirs = b.visibleDirs(scan.dirs)
	if remain <= 1 || len(node.dirs) == 0 {
		return node
	}

	_, _, subtreeLimit := listDirDisplayLimits(isRoot)
	expandCount := minInt(len(node.dirs), subtreeLimit)
	node.expandedDirs = make([]*listDirVisibleNode, 0, expandCount)
	for i := 0; i < expandCount; i++ {
		child := node.dirs[i]
		node.expandedDirs = append(node.expandedDirs, b.build(child.absPath, child.relPath, remain-1, false))
	}
	return node
}

func (b *listDirVisibleTreeBuilder) scanImmediate(dirPath, relPath string) *listDirScanResult {
	dirPath = pathpolicy.NormalizeWorkspaceRoot(dirPath)
	if cached, ok := b.scanMemo[dirPath]; ok {
		return cached
	}

	result := &listDirScanResult{}
	b.scanMemo[dirPath] = result

	rawEntries, err := b.readDir(dirPath)
	if err != nil {
		result.readErr = err
		return result
	}

	for _, entry := range rawEntries {
		childPath := filepath.Join(dirPath, entry.Name())
		if b.isIgnoredPath(childPath, entry.IsDir()) {
			continue
		}

		if entry.IsDir() {
			result.dirs = append(result.dirs, listDirVisibleDir{
				name:    entry.Name(),
				relPath: joinListDirRelPath(relPath, entry.Name()),
				absPath: childPath,
			})
			continue
		}

		if !b.matchesFileFilter(childPath) {
			continue
		}
		result.files = append(result.files, entry)
	}

	return result
}

func (b *listDirVisibleTreeBuilder) visibleDirs(dirs []listDirVisibleDir) []listDirVisibleDir {
	if b.fileFilter == "" {
		return append([]listDirVisibleDir(nil), dirs...)
	}

	visible := make([]listDirVisibleDir, 0, len(dirs))
	for _, dir := range dirs {
		if b.filterIdx != nil && b.filterIdx.hasVisibleDir(dir.absPath) {
			visible = append(visible, dir)
		}
	}
	return visible
}

func (b *listDirVisibleTreeBuilder) isIgnoredPath(path string, isDir bool) bool {
	if b.matcher == nil {
		return false
	}

	relPath, err := filepath.Rel(b.rootPath, path)
	if err != nil {
		return false
	}
	return b.matcher.Match(filepath.ToSlash(relPath), isDir)
}

func (b *listDirVisibleTreeBuilder) matchesFileFilter(path string) bool {
	return listDirFilterCandidateMatches(path, b.rootPath, b.filterRoot, nil, b.fileFilter)
}
