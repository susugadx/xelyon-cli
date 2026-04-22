package repomap

import "fmt"

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
