package repomap

import "fmt"

type buildMode int

const (
	buildModeFull buildMode = iota
	buildModeManifest
)

type buildResult struct {
	entries []*FileEntry
	cache   *MapCache
}

func (pm *ProjectMap) runBuild(mode buildMode) (*buildResult, error) {
	if err := pm.validateBuildPreconditions(); err != nil {
		return nil, err
	}

	switch mode {
	case buildModeFull:
		artifacts, err := pm.collectBuildArtifacts()
		if err != nil {
			return nil, err
		}
		return &buildResult{entries: artifacts.entries, cache: artifacts.cache}, nil
	case buildModeManifest:
		entries, err := pm.collectManifestEntries()
		if err != nil {
			return nil, err
		}
		return &buildResult{entries: entries}, nil
	default:
		return nil, fmt.Errorf("unknown build mode: %d", mode)
	}
}

func (pm *ProjectMap) collectManifestEntries() ([]*FileEntry, error) {
	paths, err := pm.listFiles()
	if err != nil {
		return nil, err
	}

	entries := make([]*FileEntry, 0, len(paths))
	for _, relPath := range paths {
		entries = append(entries, &FileEntry{Path: relPath})
	}
	return entries, nil
}

func (pm *ProjectMap) applyBuildResult(entries []*FileEntry) {
	pm.Files = entries
	pm.GitStatus = pm.loadGitStatus()
}
