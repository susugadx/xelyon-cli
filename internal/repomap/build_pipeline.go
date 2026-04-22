package repomap

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type buildArtifacts struct {
	entries []*FileEntry
	cache   *MapCache
}

func (pm *ProjectMap) validateBuildPreconditions() error {
	if pm == nil {
		return fmt.Errorf("project map is nil")
	}
	if !common.IsRipgrepAvailable() {
		return fmt.Errorf("ripgrep (rg) is required")
	}
	return nil
}

func (pm *ProjectMap) collectBuildArtifacts() (*buildArtifacts, error) {
	states, symbolsByFile, err := pm.collectBuildInputs()
	if err != nil {
		return nil, err
	}

	entries, cache, err := pm.buildEntriesAndCache(states, symbolsByFile)
	if err != nil {
		return nil, err
	}

	return &buildArtifacts{
		entries: entries,
		cache:   cache,
	}, nil
}

func (pm *ProjectMap) collectBuildInputs() ([]fileState, map[string][]Symbol, error) {
	paths, err := pm.listFiles()
	if err != nil {
		return nil, nil, err
	}

	cache := loadBuildInputCache(pm.RootPath)
	states, err := pm.buildFileStates(paths)
	if err != nil {
		return nil, nil, err
	}
	states = applyCachePolicyToStates(states, cache)

	symbolsByFile, err := pm.scanSymbols(states)
	if err != nil {
		return nil, nil, err
	}
	return states, symbolsByFile, nil
}

func (pm *ProjectMap) buildEntriesAndCache(states []fileState, symbolsByFile map[string][]Symbol) ([]*FileEntry, *MapCache, error) {
	entries := make([]*FileEntry, 0, len(states))
	cache := newEmptyMapCache(pm.RootPath)
	cache.Files = make(map[string]*CacheFile, len(states))

	for _, state := range states {
		entry, cacheFile, err := buildEntryAndCacheFile(state, symbolsByFile[state.path])
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, entry)
		cache.Files[state.path] = cacheFile
	}

	sortFileEntries(entries)
	return entries, cache, nil
}

func buildEntryAndCacheFile(state fileState, scannedSymbols []Symbol) (*FileEntry, *CacheFile, error) {
	if cached, ok := reusableCacheForState(state); ok {
		entry := &FileEntry{
			Path:      state.path,
			LineCount: cached.LineCount,
		}
		if len(cached.Symbols) > 0 {
			entry.Symbols = append([]Symbol(nil), cached.Symbols...)
		}
		return entry, cached, nil
	}

	lineCount, err := countLines(state.absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("count lines %s: %w", state.path, err)
	}

	entry := &FileEntry{
		Path:      state.path,
		LineCount: lineCount,
		Symbols:   scannedSymbols,
	}

	cacheFile := &CacheFile{
		ModTime:   state.modTime,
		LineCount: lineCount,
		Symbols:   append([]Symbol(nil), scannedSymbols...),
	}
	return entry, cacheFile, nil
}

func reusableCacheForState(state fileState) (*CacheFile, bool) {
	if state.cached == nil {
		return nil, false
	}
	return cloneCacheFile(state.cached), true
}
