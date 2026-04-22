package navigation

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func normalizeGoSymbolSnapshotRootPath(rootPath string, pm *repomap.ProjectMap) string {
	root := strings.TrimSpace(rootPath)
	if root == "" && pm != nil {
		root = strings.TrimSpace(pm.RootPath)
	}
	return normalizeNavigationRootPath(root)
}

func appendGoSymbolSnapshotEntriesFromFile(snapshot *goSymbolSnapshot, stableKeyCounts map[string]int, file *repomap.FileEntry) {
	if snapshot == nil || file == nil {
		return
	}
	if !strings.HasSuffix(strings.ToLower(file.Path), ".go") {
		return
	}

	relPath := filepath.Clean(filepath.ToSlash(file.Path))
	for _, symbol := range file.Symbols {
		entry := newGoSymbolSnapshotEntry(relPath, symbol)
		snapshot.ByName[entry.Name] = append(snapshot.ByName[entry.Name], entry)
		stableKeyCounts[entry.StableKey]++
	}
}

func newGoSymbolSnapshotEntry(relPath string, symbol repomap.Symbol) goSymbolSnapshotEntry {
	receiver := extractMethodReceiver(symbol.Signature)
	receiverNorm := canonicalReceiver(receiver)
	packageDir := filepath.Dir(relPath)
	return goSymbolSnapshotEntry{
		Name:         symbol.Name,
		Kind:         symbol.Kind,
		File:         relPath,
		Line:         symbol.Line,
		EndLine:      symbol.EndLine,
		Signature:    symbol.Signature,
		Exported:     symbol.Exported,
		Receiver:     receiver,
		ReceiverNorm: receiverNorm,
		PackageDir:   packageDir,
		StableKey:    stableGoSymbolKey(packageDir, receiverNorm, symbol.Name, symbol.Kind, symbol.Signature),
	}
}

func finalizeGoSymbolSnapshotEntries(snapshot *goSymbolSnapshot, stableKeyCounts map[string]int) {
	if snapshot == nil {
		return
	}
	for name, entries := range snapshot.ByName {
		snapshot.ByName[name] = markAndSortGoSymbolSnapshotEntries(entries, stableKeyCounts)
	}
}

func markAndSortGoSymbolSnapshotEntries(entries []goSymbolSnapshotEntry, stableKeyCounts map[string]int) []goSymbolSnapshotEntry {
	for i := range entries {
		entries[i].Collision = stableKeyCounts[entries[i].StableKey] > 1
	}
	sort.SliceStable(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.EndLine != right.EndLine {
			return left.EndLine < right.EndLine
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.Signature < right.Signature
	})
	return entries
}
