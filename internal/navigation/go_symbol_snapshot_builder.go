package navigation

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func buildGoSymbolSnapshot(pm *repomap.ProjectMap, rootPath, stateKey string) *goSymbolSnapshot {
	if pm == nil {
		return nil
	}

	root := normalizeGoSymbolSnapshotRootPath(rootPath, pm)

	snapshot := &goSymbolSnapshot{
		RootPath: root,
		StateKey: strings.TrimSpace(stateKey),
		ByName:   make(map[string][]goSymbolSnapshotEntry),
	}
	stableKeyCounts := make(map[string]int)
	for _, file := range pm.Files {
		appendGoSymbolSnapshotEntriesFromFile(snapshot, stableKeyCounts, file)
	}
	finalizeGoSymbolSnapshotEntries(snapshot, stableKeyCounts)
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
