package navigation

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func buildGoSymbolSnapshot(pm *repomap.ProjectMap, rootPath, stateKey string) *goSymbolSnapshot {
	if pm == nil {
		return nil
	}

	root := strings.TrimSpace(rootPath)
	if root == "" {
		root = strings.TrimSpace(pm.RootPath)
	}
	if root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			root = abs
		}
	}

	snapshot := &goSymbolSnapshot{
		RootPath: root,
		StateKey: strings.TrimSpace(stateKey),
		ByName:   make(map[string][]goSymbolSnapshotEntry),
	}
	stableKeyCounts := make(map[string]int)
	for _, file := range pm.Files {
		if file == nil || !strings.HasSuffix(strings.ToLower(file.Path), ".go") {
			continue
		}
		relPath := filepath.Clean(filepath.ToSlash(file.Path))
		packageDir := filepath.Dir(relPath)
		for _, symbol := range file.Symbols {
			receiver := extractMethodReceiver(symbol.Signature)
			entry := goSymbolSnapshotEntry{
				Name:         symbol.Name,
				Kind:         symbol.Kind,
				File:         relPath,
				Line:         symbol.Line,
				EndLine:      symbol.EndLine,
				Signature:    symbol.Signature,
				Exported:     symbol.Exported,
				Receiver:     receiver,
				ReceiverNorm: canonicalReceiver(receiver),
				PackageDir:   packageDir,
				StableKey:    stableGoSymbolKey(packageDir, canonicalReceiver(receiver), symbol.Name, symbol.Kind, symbol.Signature),
			}
			snapshot.ByName[entry.Name] = append(snapshot.ByName[entry.Name], entry)
			stableKeyCounts[entry.StableKey]++
		}
	}
	for name := range snapshot.ByName {
		for i := range snapshot.ByName[name] {
			snapshot.ByName[name][i].Collision = stableKeyCounts[snapshot.ByName[name][i].StableKey] > 1
		}
		sort.SliceStable(snapshot.ByName[name], func(i, j int) bool {
			left := snapshot.ByName[name][i]
			right := snapshot.ByName[name][j]
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
	}
	return snapshot
}

func stableGoSymbolKey(packageDir, receiverNorm, name, kind, signature string) string {
	sigHash := sha256.Sum256([]byte(strings.TrimSpace(signature)))
	return strings.Join([]string{
		"go",
		filepath.ToSlash(filepath.Clean(packageDir)),
		strings.TrimSpace(receiverNorm),
		strings.TrimSpace(name),
		strings.TrimSpace(kind),
		hex.EncodeToString(sigHash[:8]),
	}, "|")
}
