package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const defaultSkillCatalogStoreMaxEntries = 32

type catalogCacheEntry struct {
	rootState   string
	discover    DiscoverResult
	fingerprint string
	catalog     SkillCatalog
	lastAccess  uint64
}

type discoverCatalogFunc func(DiscoverOptions) DiscoverResult
type buildCatalogFunc func(DiscoverResult) SkillCatalog
type buildCatalogWithContentFunc func(DiscoverResult, map[string][]byte) SkillCatalog

// SkillCatalogStore は discover 結果の fingerprint を用いて catalog を共有キャッシュする。
type SkillCatalogStore struct {
	mu                        sync.Mutex
	entries                   map[string]catalogCacheEntry
	max                       int
	clock                     uint64
	discoverFn                discoverCatalogFunc
	buildCatalogFn            buildCatalogFunc
	buildCatalogWithContentFn buildCatalogWithContentFunc
}

var (
	defaultCatalogStore = NewSkillCatalogStore()
)

// NewSkillCatalogStore は skill catalog store を初期化する。
func NewSkillCatalogStore() *SkillCatalogStore {
	return NewSkillCatalogStoreWithLimit(defaultSkillCatalogStoreMaxEntries)
}

// NewSkillCatalogStoreWithLimit は最大エントリ数を指定して skill catalog store を初期化する。
func NewSkillCatalogStoreWithLimit(maxEntries int) *SkillCatalogStore {
	return NewSkillCatalogStoreWithDeps(maxEntries, Discover, Catalog, CatalogWithContentCache)
}

// NewSkillCatalogStoreWithDeps は依存関数を注入して skill catalog store を初期化する。
func NewSkillCatalogStoreWithDeps(maxEntries int, discoverFn discoverCatalogFunc, buildFn buildCatalogFunc, buildWithContentFn buildCatalogWithContentFunc) *SkillCatalogStore {
	if maxEntries <= 0 {
		maxEntries = defaultSkillCatalogStoreMaxEntries
	}
	if discoverFn == nil {
		discoverFn = Discover
	}
	if buildFn == nil {
		buildFn = Catalog
	}
	return &SkillCatalogStore{
		entries:                   make(map[string]catalogCacheEntry),
		max:                       maxEntries,
		discoverFn:                discoverFn,
		buildCatalogFn:            buildFn,
		buildCatalogWithContentFn: buildWithContentFn,
	}
}

// Load は Discover -> fingerprint 判定 -> Catalog 構築/再利用を行う。
func (s *SkillCatalogStore) Load(opts DiscoverOptions) SkillCatalog {
	discoverRoots := resolveDiscoverRootsFromOptions(opts)
	cacheKey := buildRootsCacheKey(discoverRoots)
	rootState := buildRootsStateFingerprint(discoverRoots)

	if s == nil {
		discover := Discover(opts)
		return Catalog(discover)
	}

	discover := DiscoverResult{}
	cached := catalogCacheEntry{}
	hit := false

	s.mu.Lock()
	if s.entries == nil {
		s.entries = make(map[string]catalogCacheEntry)
	}
	cached, hit = s.entries[cacheKey]
	if hit && cached.rootState == rootState {
		discover = cloneDiscoverResult(cached.discover)
	} else {
		hit = false
	}
	s.mu.Unlock()

	if !hit {
		discover = s.resolveDiscoverFn()(opts)
	}

	fingerprint, skillContents := buildCatalogFingerprintWithContent(discover)
	if hit && cached.fingerprint == fingerprint {
		s.mu.Lock()
		s.touchEntryLocked(cacheKey, &cached)
		s.mu.Unlock()
		return cloneSkillCatalog(cached.catalog)
	}

	catalog := s.buildCatalog(discover, skillContents)

	s.mu.Lock()
	s.entries[cacheKey] = catalogCacheEntry{
		rootState:   rootState,
		discover:    cloneDiscoverResult(discover),
		fingerprint: fingerprint,
		catalog:     cloneSkillCatalog(catalog),
		lastAccess:  s.nextClockLocked(),
	}
	s.evictIfNeededLocked()
	s.mu.Unlock()

	return catalog
}

func (s *SkillCatalogStore) resolveDiscoverFn() discoverCatalogFunc {
	if s == nil || s.discoverFn == nil {
		return Discover
	}
	return s.discoverFn
}

func (s *SkillCatalogStore) buildCatalog(discover DiscoverResult, skillContents map[string][]byte) SkillCatalog {
	if s == nil {
		return CatalogWithContentCache(discover, skillContents)
	}
	if s.buildCatalogWithContentFn != nil {
		return s.buildCatalogWithContentFn(discover, skillContents)
	}
	if s.buildCatalogFn != nil {
		return s.buildCatalogFn(discover)
	}
	return CatalogWithContentCache(discover, skillContents)
}

// Clear は cache を破棄する。主にテスト用途。
func (s *SkillCatalogStore) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.entries = make(map[string]catalogCacheEntry)
	s.clock = 0
	s.mu.Unlock()
}

func (s *SkillCatalogStore) touchEntryLocked(key string, entry *catalogCacheEntry) {
	if s == nil || entry == nil {
		return
	}
	entry.lastAccess = s.nextClockLocked()
	s.entries[key] = *entry
}

func (s *SkillCatalogStore) nextClockLocked() uint64 {
	if s == nil {
		return 0
	}
	s.clock++
	return s.clock
}

func (s *SkillCatalogStore) evictIfNeededLocked() {
	if s == nil || s.max <= 0 || len(s.entries) <= s.max {
		return
	}

	var oldestKey string
	var oldestAccess uint64
	first := true
	for key, entry := range s.entries {
		if first || entry.lastAccess < oldestAccess {
			oldestKey = key
			oldestAccess = entry.lastAccess
			first = false
		}
	}
	if oldestKey != "" {
		delete(s.entries, oldestKey)
	}
}

func buildRootsCacheKey(roots []discoverRoot) string {
	if len(roots) == 0 {
		return "(no-roots)"
	}
	keys := make([]string, 0, len(roots))
	for _, root := range roots {
		keys = append(keys, cleanAbsPathOrFallback(root.Path))
	}
	return strings.Join(keys, "\x00")
}

func buildCatalogFingerprint(discover DiscoverResult) string {
	fingerprint, _ := buildCatalogFingerprintWithContent(discover)
	return fingerprint
}

func buildCatalogFingerprintWithContent(discover DiscoverResult) (string, map[string][]byte) {
	hasher := sha256.New()
	skillContents := make(map[string][]byte, len(discover.Skills))
	for _, root := range discover.Roots {
		_, _ = hasher.Write([]byte("root:" + cleanAbsPathOrFallback(root) + "\n"))
	}
	for _, skill := range discover.Skills {
		writeCatalogFingerprintEntry(hasher, "skill", cleanAbsPathOrFallback(skill.SkillPath), skillContents)
		for _, group := range skillResourceGroupOrder {
			writeCatalogResourceListingFingerprint(hasher, cleanAbsPathOrFallback(skill.Directory), group.String())
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), skillContents
}

func writeCatalogFingerprintEntry(hasher interface{ Write([]byte) (int, error) }, kind, path string, skillContents map[string][]byte) {
	if hasher == nil {
		return
	}
	_, _ = hasher.Write([]byte(kind + ":" + path))
	data, err := os.ReadFile(path)
	if err != nil {
		_, _ = hasher.Write([]byte("|err=" + err.Error() + "\n"))
		return
	}
	sum := sha256.Sum256(data)
	if skillContents != nil {
		skillContents[path] = append([]byte(nil), data...)
	}
	_, _ = hasher.Write([]byte("|sha256=" + hex.EncodeToString(sum[:]) + "\n"))
}

func writeCatalogResourceListingFingerprint(hasher interface{ Write([]byte) (int, error) }, skillDir, group string) {
	if hasher == nil {
		return
	}
	target := filepath.Join(skillDir, group)
	entries, err := os.ReadDir(target)
	if err != nil {
		_, _ = hasher.Write([]byte("list:" + target + "|err=" + err.Error() + "\n"))
		return
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	_, _ = hasher.Write([]byte("list:" + target + "|files=" + strings.Join(files, ",") + "\n"))
}

func resolveDiscoverRootsFromOptions(opts DiscoverOptions) []discoverRoot {
	cwd := strings.TrimSpace(opts.InvocationCWD)
	if cwd == "" {
		if current, err := os.Getwd(); err == nil {
			cwd = current
		}
	}
	if cwd == "" {
		cwd = "."
	}

	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}

	return resolveDiscoverRoots(cwd, home)
}

func buildRootsStateFingerprint(roots []discoverRoot) string {
	hasher := sha256.New()
	for _, root := range roots {
		path := cleanAbsPathOrFallback(root.Path)
		_, _ = hasher.Write([]byte("root:" + path))
		info, err := os.Stat(path)
		if err != nil {
			_, _ = hasher.Write([]byte("|err=" + err.Error() + "\n"))
			continue
		}
		_, _ = hasher.Write([]byte(fmt.Sprintf("|mtime=%d|size=%d\n", info.ModTime().UnixNano(), info.Size())))

		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			_, _ = hasher.Write([]byte("|entries_err=" + readErr.Error() + "\n"))
			continue
		}

		childDirs := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				childDirs = append(childDirs, entry.Name())
			}
		}
		sort.Strings(childDirs)
		_, _ = hasher.Write([]byte("|child_dirs=" + strings.Join(childDirs, ",") + "\n"))
		for _, child := range childDirs {
			childPath := filepath.Join(path, child)
			childInfo, childErr := os.Stat(childPath)
			if childErr != nil {
				_, _ = hasher.Write([]byte("|child:" + child + "|err=" + childErr.Error() + "\n"))
				continue
			}
			_, _ = hasher.Write([]byte(fmt.Sprintf("|child:%s|mtime=%d|size=%d\n", child, childInfo.ModTime().UnixNano(), childInfo.Size())))
		}
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func cloneDiscoverResult(discover DiscoverResult) DiscoverResult {
	cloned := DiscoverResult{
		Roots:       append([]string(nil), discover.Roots...),
		Skills:      make([]DiscoveredSkill, len(discover.Skills)),
		Diagnostics: append([]Diagnostic(nil), discover.Diagnostics...),
	}
	copy(cloned.Skills, discover.Skills)
	return cloned
}

func cloneSkillCatalog(catalog SkillCatalog) SkillCatalog {
	cloned := SkillCatalog{
		Skills:      make([]ParsedSkill, len(catalog.Skills)),
		Diagnostics: append([]Diagnostic(nil), catalog.Diagnostics...),
	}
	for i, skill := range catalog.Skills {
		cloned.Skills[i] = cloneParsedSkill(skill)
	}
	return cloned
}

func cloneParsedSkill(skill ParsedSkill) ParsedSkill {
	cloned := skill
	for _, group := range skillResourceGroupOrder {
		items := append([]string(nil), skillResourceItems(skill, group)...)
		setSkillResourceItems(&cloned, group, items)
	}
	return cloned
}
