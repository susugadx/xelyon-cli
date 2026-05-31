package taskstate

import "github.com/susugadx/xelyon-cli/internal/tools"

// ChangedFiles は変更が観測されたファイルを初出順で保持する。
type ChangedFiles struct {
	files []fileFact
}

func (g ChangedFiles) clone() ChangedFiles {
	return ChangedFiles{files: cloneFileFacts(g.files)}
}

// Paths は記録済みファイルパスを防御コピーで返す。
func (g ChangedFiles) Paths() []string {
	return fileFactPaths(g.files)
}

// Len は記録済みファイルパス数を返す。
func (g ChangedFiles) Len() int {
	return len(g.files)
}

func (g *ChangedFiles) recordPath(path string) {
	if g == nil || path == "" || g.contains(path) {
		return
	}
	if len(g.files) >= maxRecordedFiles {
		return
	}
	g.files = append(g.files, fileFact{path: path})
}

func (g ChangedFiles) contains(path string) bool {
	for _, file := range g.files {
		if file.path == path {
			return true
		}
	}
	return false
}

// TouchedFiles は読み取りや参照で触れたファイルを初出順で保持する。
type TouchedFiles struct {
	files []fileFact
}

func (g TouchedFiles) clone() TouchedFiles {
	return TouchedFiles{files: cloneFileFacts(g.files)}
}

// Paths は記録済みファイルパスを防御コピーで返す。
func (g TouchedFiles) Paths() []string {
	return fileFactPaths(g.files)
}

// Len は記録済みファイルパス数を返す。
func (g TouchedFiles) Len() int {
	return len(g.files)
}

func (g *TouchedFiles) recordPath(path string) {
	if g == nil || path == "" || g.contains(path) {
		return
	}
	if len(g.files) >= maxRecordedFiles {
		return
	}
	g.files = append(g.files, fileFact{path: path})
}

func (g TouchedFiles) contains(path string) bool {
	for _, file := range g.files {
		if file.path == path {
			return true
		}
	}
	return false
}

type fileFact struct {
	path string
}

// Path は記録されたファイルパスを返す。
func (f fileFact) Path() string {
	return f.path
}

func cloneFileFacts(files []fileFact) []fileFact {
	if len(files) == 0 {
		return nil
	}
	cloned := make([]fileFact, len(files))
	copy(cloned, files)
	return cloned
}

func fileFactPaths(files []fileFact) []string {
	if len(files) == 0 {
		return nil
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.path)
	}
	return paths
}

func changedPathsFromFileChange(change tools.FileChange) []string {
	if len(change.Details) > 0 {
		paths := make([]string, 0, len(change.Details))
		for _, detail := range change.Details {
			paths = append(paths, detail.FilePath)
		}
		return paths
	}
	return []string{change.FilePath}
}
