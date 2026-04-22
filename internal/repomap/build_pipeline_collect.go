package repomap

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
