package skills

import (
	"sync"

	"github.com/susugadx/xelyon-cli/internal/cacheutil"
)

const defaultSkillCatalogStoreMaxEntries = 32

type catalogCacheEntry struct {
	rootState   string
	discover    DiscoverResult
	fingerprint string
	catalog     SkillCatalog
}

type discoverCatalogFunc func(DiscoverOptions) DiscoverResult
type buildCatalogFunc func(DiscoverResult) SkillCatalog
type buildCatalogWithContentFunc func(DiscoverResult, map[string][]byte) SkillCatalog

// SkillCatalogStore は discover 結果の fingerprint を用いて catalog を共有キャッシュする。
type SkillCatalogStore struct {
	mu                        sync.Mutex
	cache                     *cacheutil.LRU[string, catalogCacheEntry]
	maxEntries                int
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
		cache:                     cacheutil.NewLRU[string, catalogCacheEntry](maxEntries),
		maxEntries:                maxEntries,
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
	var cached catalogCacheEntry
	hit := false

	s.mu.Lock()
	s.ensureCacheLocked()
	cached, hit = s.cache.Peek(cacheKey)
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
		s.ensureCacheLocked()
		s.cache.Set(cacheKey, cached)
		s.mu.Unlock()
		return cloneSkillCatalog(cached.catalog)
	}

	catalog := s.buildCatalog(discover, skillContents)
	cached = catalogCacheEntry{
		rootState:   rootState,
		discover:    cloneDiscoverResult(discover),
		fingerprint: fingerprint,
		catalog:     cloneSkillCatalog(catalog),
	}

	s.mu.Lock()
	s.ensureCacheLocked()
	s.cache.Set(cacheKey, cached)
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
	s.cache = cacheutil.NewLRU[string, catalogCacheEntry](s.effectiveMaxEntries())
	s.mu.Unlock()
}

func (s *SkillCatalogStore) ensureCacheLocked() {
	if s == nil || s.cache != nil {
		return
	}
	s.cache = cacheutil.NewLRU[string, catalogCacheEntry](s.effectiveMaxEntries())
}

func (s *SkillCatalogStore) effectiveMaxEntries() int {
	if s == nil || s.maxEntries <= 0 {
		return defaultSkillCatalogStoreMaxEntries
	}
	return s.maxEntries
}
