package repomap

import (
	"path/filepath"
	"sort"
	"strings"
)

func sortSymbolsByLocation(results map[string][]Symbol) {
	for path := range results {
		sort.Slice(results[path], func(i, j int) bool {
			left := results[path][i]
			right := results[path][j]
			if left.Line != right.Line {
				return left.Line < right.Line
			}
			if left.EndLine != right.EndLine {
				return left.EndLine < right.EndLine
			}
			if left.Name != right.Name {
				return left.Name < right.Name
			}
			return left.Signature < right.Signature
		})
	}
}

func compareFileEntryPath(leftPath, rightPath string) bool {
	leftName := filepath.Base(leftPath)
	rightName := filepath.Base(rightPath)
	leftBase := testSortBase(leftName)
	rightBase := testSortBase(rightName)
	if leftBase != rightBase {
		return leftBase < rightBase
	}
	leftTest := isTestFile(leftName)
	rightTest := isTestFile(rightName)
	if leftTest != rightTest {
		return !leftTest
	}
	return strings.ToLower(leftName) < strings.ToLower(rightName)
}

func sortFileEntries(entries []*FileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		leftDir := filepath.ToSlash(filepath.Dir(entries[i].Path))
		rightDir := filepath.ToSlash(filepath.Dir(entries[j].Path))
		if leftDir != rightDir {
			return leftDir < rightDir
		}
		return compareFileEntryPath(entries[i].Path, entries[j].Path)
	})
}

func directoryDepth(path string) int {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return 0
	}
	return strings.Count(dir, "/") + 1
}
