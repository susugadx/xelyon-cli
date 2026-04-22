package navigation

import (
	"os"
	"path/filepath"
	"strings"
)

func resolveSymbolCandidatesFromASTSource(query symbolQuery, pathHint string, runtime GoSymbolRuntime) []SymbolCandidate {
	goFiles := listGoFiles(pathHint)
	if len(goFiles) == 0 {
		return nil
	}

	rootPath := resolveRuntimeRootPath(runtime)
	var candidates []SymbolCandidate
	for _, file := range goFiles {
		symbols, err := extractASTSymbols(file)
		if err != nil {
			continue
		}
		for _, symbol := range symbols {
			if !symbolQueryMatches(query, symbol) {
				continue
			}
			candidates = append(candidates, newSymbolCandidateFromASTSymbol(symbol, file, rootPath))
		}
	}

	sortSymbolCandidates(candidates)
	return candidates
}

func resolveRuntimeRootPath(runtime GoSymbolRuntime) string {
	rootPath := strings.TrimSpace(runtime.InvocationCWD)
	if rootPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			rootPath = cwd
		}
	}
	if rootPath != "" {
		if absPath, err := filepath.Abs(rootPath); err == nil {
			return absPath
		}
	}
	return rootPath
}
