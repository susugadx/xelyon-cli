package repomap

import (
	"fmt"
	"path/filepath"
	"strings"
)

func writeRenderedDirectory(
	b *strings.Builder,
	dir string,
	files []*FileEntry,
	pathIndex map[string]int,
	options []renderOption,
) {
	b.WriteString("📂 ")
	b.WriteString(dir)
	b.WriteString("\n")

	for fileIndex, file := range files {
		connector := "├── "
		symbolPrefix := "│     "
		if fileIndex == len(files)-1 {
			connector = "└── "
			symbolPrefix = "      "
		}

		fmt.Fprintf(b, "%s📄 %s (%d lines)\n", connector, filepath.Base(file.Path), file.LineCount)
		idx, ok := pathIndex[file.Path]
		if !ok || !options[idx].showSymbols {
			continue
		}
		for _, symbol := range file.Symbols {
			writeRenderedSymbol(b, symbolPrefix, symbol)
		}
	}
}

func writeRenderedOmittedFiles(b *strings.Builder, renderedDirCount, omittedFiles int) {
	if omittedFiles <= 0 {
		return
	}
	if renderedDirCount > 0 {
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "... (%d more files, truncated to fit token budget)\n", omittedFiles)
}

func writeRenderedGitStatus(b *strings.Builder, renderedDirCount, omittedFiles int, gitStatus []GitChange) {
	if len(gitStatus) == 0 {
		return
	}
	if renderedDirCount > 0 || omittedFiles > 0 {
		b.WriteString("\n")
	}
	b.WriteString("## Uncommitted Changes\n")
	for _, change := range gitStatus {
		fmt.Fprintf(b, "  %s %s\n", change.Status, change.Path)
	}
}
