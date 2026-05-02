package agent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/cacheutil"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultProjectConfigStoreMaxEntries = 64

type projectConfigCacheEntry struct {
	path       string
	modTimeNS  int64
	size       int64
	projectCfg *config.ProjectConfig
}

// ProjectConfigStore は xelyon.yaml のロード結果を cwd 単位で共有する軽量キャッシュ。
type ProjectConfigStore struct {
	mu         sync.Mutex
	cache      *cacheutil.LRU[string, projectConfigCacheEntry]
	maxEntries int
}

var (
	defaultProjectConfigStore = NewProjectConfigStore()
	loadProjectConfigFromDisk = func() *config.ProjectConfig {
		return config.LoadProjectConfig()
	}
)

func NewProjectConfigStore() *ProjectConfigStore {
	return NewProjectConfigStoreWithLimit(defaultProjectConfigStoreMaxEntries)
}

func NewProjectConfigStoreWithLimit(maxEntries int) *ProjectConfigStore {
	if maxEntries <= 0 {
		maxEntries = defaultProjectConfigStoreMaxEntries
	}
	return &ProjectConfigStore{
		cache:      cacheutil.NewLRU[string, projectConfigCacheEntry](maxEntries),
		maxEntries: maxEntries,
	}
}

// Load は現在の cwd に対応する project config を返す。
func (s *ProjectConfigStore) Load() *config.ProjectConfig {
	cwd, err := os.Getwd()
	if err != nil {
		return loadProjectConfigFromDisk()
	}
	return s.LoadForCWD(cwd)
}

// LoadForCWD は指定 cwd に対応する project config を返す。
func (s *ProjectConfigStore) LoadForCWD(cwd string) *config.ProjectConfig {
	if strings.TrimSpace(cwd) == "" {
		return loadProjectConfigFromDisk()
	}
	return s.loadForCWD(cwd)
}

// Clear はキャッシュを破棄する。
func (s *ProjectConfigStore) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.cache = cacheutil.NewLRU[string, projectConfigCacheEntry](s.effectiveMaxEntries())
	s.mu.Unlock()
}

func (s *ProjectConfigStore) loadForCWD(cwd string) *config.ProjectConfig {
	if s == nil {
		return loadProjectConfigFromDisk()
	}

	cwd = cleanProjectPath(cwd)
	path, ok := findProjectConfigPath(cwd)
	if !ok {
		s.mu.Lock()
		s.ensureCacheLocked()
		s.cache.Delete(cwd)
		s.mu.Unlock()
		return nil
	}

	modTimeNS, size, statOK := projectConfigFileSignature(path)
	if statOK {
		s.mu.Lock()
		s.ensureCacheLocked()
		entry, hit := s.cache.Get(cwd)
		if hit && sameProjectPath(entry.path, path) && entry.modTimeNS == modTimeNS && entry.size == size {
			cached := cloneProjectConfig(entry.projectCfg)
			s.mu.Unlock()
			return cached
		}
		s.mu.Unlock()
	}

	loaded := loadProjectConfigFromDisk()
	if loaded == nil {
		s.mu.Lock()
		s.ensureCacheLocked()
		s.cache.Delete(cwd)
		s.mu.Unlock()
		return nil
	}

	modTimeNS, size, statOK = projectConfigFileSignature(loaded.FilePath)
	if !statOK {
		return cloneProjectConfig(loaded)
	}

	s.mu.Lock()
	s.ensureCacheLocked()
	s.cache.Set(cwd, projectConfigCacheEntry{
		path:       cleanProjectPath(loaded.FilePath),
		modTimeNS:  modTimeNS,
		size:       size,
		projectCfg: cloneProjectConfig(loaded),
	})
	s.mu.Unlock()
	return cloneProjectConfig(loaded)
}

func (s *ProjectConfigStore) ensureCacheLocked() {
	if s == nil || s.cache != nil {
		return
	}
	s.cache = cacheutil.NewLRU[string, projectConfigCacheEntry](s.effectiveMaxEntries())
}

func (s *ProjectConfigStore) effectiveMaxEntries() int {
	if s == nil || s.maxEntries <= 0 {
		return defaultProjectConfigStoreMaxEntries
	}
	return s.maxEntries
}

func findProjectConfigPath(cwd string) (string, bool) {
	cwd = cleanProjectPath(cwd)
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "xelyon.yaml")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return cleanProjectPath(candidate), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", false
}

func projectConfigFileSignature(path string) (modTimeNS int64, size int64, ok bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0, 0, false
	}
	return info.ModTime().UnixNano(), info.Size(), true
}

func cloneProjectConfig(pc *config.ProjectConfig) *config.ProjectConfig {
	if pc == nil {
		return nil
	}
	cloned := *pc
	cloned.Rules = append([]string(nil), pc.Rules...)
	cloned.Ignore.Patterns = append([]string(nil), pc.Ignore.Patterns...)
	cloned.Conditional = make([]config.ProjectConditionalBlock, len(pc.Conditional))
	for i := range pc.Conditional {
		cloned.Conditional[i] = pc.Conditional[i]
		cloned.Conditional[i].Paths = append([]string(nil), pc.Conditional[i].Paths...)
		cloned.Conditional[i].Rules = append([]string(nil), pc.Conditional[i].Rules...)
	}
	if pc.FinalChecks != nil {
		fc := *pc.FinalChecks
		fc.Commands = append([]string(nil), pc.FinalChecks.Commands...)
		cloned.FinalChecks = &fc
	}
	return &cloned
}

func cleanProjectPath(path string) string {
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func sameProjectPath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return cleanProjectPath(left) == cleanProjectPath(right)
}
