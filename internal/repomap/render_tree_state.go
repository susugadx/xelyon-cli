package repomap

import (
	"path/filepath"
	"sort"
)

type renderTreeState struct {
	dirs      []string
	grouped   map[string][]*FileEntry
	pathIndex map[string]int
}

func buildRenderTreeState(files []*FileEntry, options []renderOption) renderTreeState {
	state := renderTreeState{
		grouped:   make(map[string][]*FileEntry),
		pathIndex: make(map[string]int, len(files)),
	}

	for i, file := range files {
		if file != nil {
			state.pathIndex[file.Path] = i
		}
	}

	for i, file := range files {
		if file == nil || !options[i].include {
			continue
		}
		dir := renderDirectoryName(file.Path)
		if _, ok := state.grouped[dir]; !ok {
			state.dirs = append(state.dirs, dir)
		}
		state.grouped[dir] = append(state.grouped[dir], file)
	}

	sort.Strings(state.dirs)
	for _, dir := range state.dirs {
		filesInDir := state.grouped[dir]
		sort.Slice(filesInDir, func(i, j int) bool {
			return compareFileEntryPath(filesInDir[i].Path, filesInDir[j].Path)
		})
		state.grouped[dir] = filesInDir
	}

	return state
}

func renderDirectoryName(path string) string {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return "./"
	}
	return dir + "/"
}
