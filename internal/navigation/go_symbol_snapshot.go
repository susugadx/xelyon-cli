package navigation

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// GoSymbolRuntime carries nil-safe runtime state for snapshot-backed Go symbol resolution.
type GoSymbolRuntime struct {
	ProjectMap         *repomap.ProjectMap
	ProjectMapRootPath string
	ProjectMapStateKey string
	InvocationCWD      string
}

type goSymbolSnapshot struct {
	RootPath string
	StateKey string
	ByName   map[string][]goSymbolSnapshotEntry
}

type goSymbolSnapshotEntry struct {
	Name         string
	Kind         string
	File         string
	Line         int
	EndLine      int
	Signature    string
	Exported     bool
	Receiver     string
	ReceiverNorm string
	PackageDir   string
	StableKey    string
	Collision    bool
}

var (
	goSymbolSnapshotCache    sync.Map
	goSymbolSnapshotRootKeys sync.Map
)

func init() {
	tools.AddSearchCacheLifecycleHooks(clearGoSymbolSnapshotCache, clearGoSymbolSnapshotCacheWithKeys, clearGoSymbolSnapshotCacheWithKeys)
}

func resolveSymbolCandidatesWithRuntime(symbol, pathHint string, runtime GoSymbolRuntime) []SymbolCandidate {
	query := parseSymbolQuery(symbol)
	if snapshot := loadGoSymbolSnapshot(runtime); snapshot != nil {
		if candidates := resolveSymbolCandidatesFromSnapshot(query, pathHint, snapshot, runtime); len(candidates) > 0 {
			return candidates
		}
	}
	return resolveSymbolCandidatesFromASTWithRuntime(query, pathHint, runtime)
}

func loadGoSymbolSnapshot(runtime GoSymbolRuntime) *goSymbolSnapshot {
	cacheKey := goSymbolSnapshotCacheKey(runtime.ProjectMapRootPath, runtime.ProjectMapStateKey)
	if cacheKey != "" {
		if snapshot := lookupGoSymbolSnapshot(cacheKey); snapshot != nil {
			return snapshot
		}
	}
	if runtime.ProjectMap != nil {
		snapshot := buildGoSymbolSnapshot(runtime.ProjectMap, runtime.ProjectMapRootPath, runtime.ProjectMapStateKey)
		if snapshot != nil {
			storeGoSymbolSnapshot(cacheKey, snapshot)
		}
		return snapshot
	}
	return nil
}

func goSymbolSnapshotCacheKey(rootPath, stateKey string) string {
	rootPath = strings.TrimSpace(rootPath)
	stateKey = strings.TrimSpace(stateKey)
	if rootPath == "" || stateKey == "" {
		return ""
	}
	return filepath.Clean(rootPath) + "::" + stateKey
}

func lookupGoSymbolSnapshot(cacheKey string) *goSymbolSnapshot {
	if strings.TrimSpace(cacheKey) == "" {
		return nil
	}
	value, ok := goSymbolSnapshotCache.Load(cacheKey)
	if !ok {
		return nil
	}
	snapshot, _ := value.(*goSymbolSnapshot)
	return snapshot
}

func storeGoSymbolSnapshot(cacheKey string, snapshot *goSymbolSnapshot) {
	if snapshot == nil || strings.TrimSpace(cacheKey) == "" {
		return
	}
	goSymbolSnapshotCache.Store(cacheKey, snapshot)

	rootKey := filepath.Clean(snapshot.RootPath)
	if rootKey == "" || rootKey == "." {
		return
	}
	if previous, ok := goSymbolSnapshotRootKeys.Load(rootKey); ok {
		if oldKey, _ := previous.(string); oldKey != "" && oldKey != cacheKey {
			goSymbolSnapshotCache.Delete(oldKey)
		}
	}
	goSymbolSnapshotRootKeys.Store(rootKey, cacheKey)
}

func clearGoSymbolSnapshotCache() {
	goSymbolSnapshotCache.Range(func(key, value any) bool {
		goSymbolSnapshotCache.Delete(key)
		return true
	})
	goSymbolSnapshotRootKeys.Range(func(key, value any) bool {
		goSymbolSnapshotRootKeys.Delete(key)
		return true
	})
}

func clearGoSymbolSnapshotCacheWithKeys(_ []string) {
	clearGoSymbolSnapshotCache()
}

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
