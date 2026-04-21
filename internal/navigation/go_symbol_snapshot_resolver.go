package navigation

import (
	"os"
	"path/filepath"
	"strings"
)

func resolveSymbolCandidatesWithRuntime(symbol, pathHint string, runtime GoSymbolRuntime) []SymbolCandidate {
	query := parseSymbolQuery(symbol)
	if snapshot := loadGoSymbolSnapshot(runtime); snapshot != nil {
		if candidates := resolveSymbolCandidatesFromSnapshot(query, pathHint, snapshot, runtime); len(candidates) > 0 {
			return candidates
		}
	}
	return resolveSymbolCandidatesFromASTWithRuntime(query, pathHint, runtime)
}

func resolveSymbolCandidatesFromSnapshot(query symbolQuery, pathHint string, snapshot *goSymbolSnapshot, runtime GoSymbolRuntime) []SymbolCandidate {
	if snapshot == nil || query.BaseName == "" {
		return nil
	}

	matchesPath := buildSnapshotPathMatcher(snapshot.RootPath, runtime.InvocationCWD, pathHint)
	entries := snapshot.ByName[query.BaseName]
	candidates := make([]SymbolCandidate, 0, len(entries))
	for _, entry := range entries {
		if query.Receiver != "" {
			if entry.Kind != "method" || entry.ReceiverNorm != query.Receiver {
				continue
			}
		}
		if matchesPath != nil && !matchesPath(entry.File) {
			continue
		}
		candidates = append(candidates, SymbolCandidate{
			Name:               entry.Name,
			Kind:               entry.Kind,
			File:               entry.File,
			Line:               entry.Line,
			EndLine:            entry.EndLine,
			Receiver:           entry.Receiver,
			ReceiverNorm:       entry.ReceiverNorm,
			Signature:          entry.Signature,
			Exported:           entry.Exported,
			PackageDir:         entry.PackageDir,
			StableKey:          entry.StableKey,
			StableKeyCollision: entry.Collision,
			RootPath:           snapshot.RootPath,
		})
	}

	sortSymbolCandidates(candidates)
	return candidates
}

func buildSnapshotPathMatcher(rootPath, invocationCWD, pathHint string) func(string) bool {
	pathHint = strings.TrimSpace(pathHint)
	if pathHint == "" {
		return nil
	}

	root := strings.TrimSpace(rootPath)
	if root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			root = abs
		}
	}

	normalizedHint := filepath.Clean(pathHint)
	absHint := normalizedHint
	if !filepath.IsAbs(absHint) {
		baseCWD := strings.TrimSpace(invocationCWD)
		if baseCWD == "" {
			if cwd, err := os.Getwd(); err == nil {
				baseCWD = cwd
			}
		}
		if baseCWD != "" {
			absHint = filepath.Join(baseCWD, normalizedHint)
		} else if abs, err := filepath.Abs(normalizedHint); err == nil {
			absHint = abs
		}
	}

	isDir := false
	if info, err := os.Stat(absHint); err == nil {
		isDir = info.IsDir()
	} else if filepath.Ext(normalizedHint) == "" {
		isDir = true
	}

	if root != "" {
		if isDir && filepath.Clean(absHint) == filepath.Clean(root) {
			return nil
		}
		if rel, ok := absoluteToSnapshotRel(root, absHint); ok {
			rel = filepath.Clean(filepath.ToSlash(rel))
			if isDir {
				return func(candidate string) bool {
					candidate = filepath.Clean(filepath.ToSlash(candidate))
					return candidate == rel || strings.HasPrefix(candidate, rel+"/")
				}
			}
			return func(candidate string) bool {
				return filepath.Clean(filepath.ToSlash(candidate)) == rel
			}
		}
	}

	cleanHint := filepath.Clean(filepath.ToSlash(normalizedHint))
	if isDir {
		return func(candidate string) bool {
			candidate = filepath.Clean(filepath.ToSlash(candidate))
			return candidate == cleanHint || strings.HasPrefix(candidate, cleanHint+"/")
		}
	}
	return func(candidate string) bool {
		return filepath.Clean(filepath.ToSlash(candidate)) == cleanHint
	}
}

func absoluteToSnapshotRel(rootPath, absPath string) (string, bool) {
	if rootPath == "" || absPath == "" {
		return "", false
	}
	root := filepath.Clean(rootPath)
	abs := filepath.Clean(absPath)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", false
	}
	rel = filepath.Clean(filepath.ToSlash(rel))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func findAmbiguousFilesWithRuntime(symbol string, cand SymbolCandidate, runtime GoSymbolRuntime) map[string]bool {
	ambiguous := make(map[string]bool)

	if snapshot := loadGoSymbolSnapshot(runtime); snapshot != nil {
		for _, entry := range snapshot.ByName[strings.TrimSpace(symbol)] {
			if entry.File == "" || entry.File == cand.File {
				continue
			}
			ambiguous[entry.File] = true
		}
		if len(ambiguous) > 0 {
			return ambiguous
		}
	}

	allCandidates := resolveSymbolCandidatesWithRuntime(symbol, "", runtime)
	for _, candidate := range allCandidates {
		if candidate.File == "" || candidate.File == cand.File {
			continue
		}
		ambiguous[candidate.File] = true
	}
	return ambiguous
}

func resolveSymbolCandidatesFromASTWithRuntime(query symbolQuery, pathHint string, runtime GoSymbolRuntime) []SymbolCandidate {
	goFiles := listGoFiles(pathHint)
	if len(goFiles) == 0 {
		return nil
	}

	rootPath := strings.TrimSpace(runtime.InvocationCWD)
	if rootPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			rootPath = cwd
		}
	}
	if rootPath != "" {
		if abs, err := filepath.Abs(rootPath); err == nil {
			rootPath = abs
		}
	}

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
			receiver := extractMethodReceiver(symbol.Signature)
			relPath := toRelativePath(file)
			candidates = append(candidates, SymbolCandidate{
				Name:         symbol.Name,
				Kind:         string(symbol.Kind),
				File:         relPath,
				Line:         symbol.Line,
				EndLine:      symbol.EndLine,
				Receiver:     receiver,
				ReceiverNorm: canonicalReceiver(receiver),
				Signature:    symbol.Signature,
				Exported:     symbol.Exported,
				PackageDir:   filepath.Dir(relPath),
				StableKey:    stableGoSymbolKey(filepath.Dir(relPath), canonicalReceiver(receiver), symbol.Name, string(symbol.Kind), symbol.Signature),
				RootPath:     rootPath,
			})
		}
	}

	sortSymbolCandidates(candidates)
	return candidates
}
