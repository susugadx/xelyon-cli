package repomap

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (pm *ProjectMap) render(options []renderOption, omittedFiles int) string {
	var b strings.Builder
	b.WriteString("## Project Map\n\n")

	pathIndex := make(map[string]int, len(pm.Files))
	for i, file := range pm.Files {
		if file != nil {
			pathIndex[file.Path] = i
		}
	}

	grouped := make(map[string][]*FileEntry)
	var dirs []string
	for i, file := range pm.Files {
		if file == nil || !options[i].include {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(file.Path))
		if dir == "." {
			dir = "./"
		} else {
			dir += "/"
		}
		if _, ok := grouped[dir]; !ok {
			dirs = append(dirs, dir)
		}
		grouped[dir] = append(grouped[dir], file)
	}
	sort.Strings(dirs)

	for dirIndex, dir := range dirs {
		if dirIndex > 0 {
			b.WriteString("\n")
		}
		b.WriteString("📂 ")
		b.WriteString(dir)
		b.WriteString("\n")

		files := grouped[dir]
		sort.Slice(files, func(i, j int) bool {
			return compareFileEntryPath(files[i].Path, files[j].Path)
		})

		for fileIndex, file := range files {
			connector := "├── "
			symbolPrefix := "│     "
			if fileIndex == len(files)-1 {
				connector = "└── "
				symbolPrefix = "      "
			}

			fmt.Fprintf(&b, "%s📄 %s (%d lines)\n", connector, filepath.Base(file.Path), file.LineCount)
			idx, ok := pathIndex[file.Path]
			if !ok || !options[idx].showSymbols {
				continue
			}
			for _, symbol := range file.Symbols {
				writeRenderedSymbol(&b, symbolPrefix, symbol)
			}
		}
	}

	if omittedFiles > 0 {
		if len(dirs) > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "... (%d more files, truncated to fit token budget)\n", omittedFiles)
	}

	if len(pm.GitStatus) > 0 {
		if len(dirs) > 0 || omittedFiles > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## Uncommitted Changes\n")
		for _, change := range pm.GitStatus {
			fmt.Fprintf(&b, "  %s %s\n", change.Status, change.Path)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

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

func writeRenderedSymbol(b *strings.Builder, symbolPrefix string, symbol Symbol) {
	location := strconv.Itoa(symbol.Line)
	if symbol.EndLine > 0 && symbol.EndLine != symbol.Line {
		location = fmt.Sprintf("%d-%d", symbol.Line, symbol.EndLine)
	}

	lines := strings.Split(symbol.Signature, "\n")
	if len(lines) == 0 {
		fmt.Fprintf(b, "%s%s:\n", symbolPrefix, location)
		return
	}

	fmt.Fprintf(b, "%s%s: %s\n", symbolPrefix, location, lines[0])
	if len(lines) > 1 {
		padding := strings.Repeat(" ", len(location)+2)
		for _, line := range lines[1:] {
			fmt.Fprintf(b, "%s%s%s\n", symbolPrefix, padding, line)
		}
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
