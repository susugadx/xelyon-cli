package repomap

import "time"

// ProjectMap はプロジェクトの構造マップ。
type ProjectMap struct {
	RootPath  string
	MaxTokens int
	Files     []*FileEntry
	GitStatus []GitChange

	additionalIgnoreDirs []string
}

// FileEntry はファイルのシンボル一覧。
type FileEntry struct {
	Path      string
	LineCount int
	Symbols   []Symbol
}

// Symbol はコード内の定義シンボル。
type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind,omitempty"`
	Line      int    `json:"line"`
	EndLine   int    `json:"end_line,omitempty"`
	Signature string `json:"signature"`
	Exported  bool   `json:"exported,omitempty"`
}

// GitChange は git status の変更ファイル。
type GitChange struct {
	Status string
	Path   string
}

type fileState struct {
	path        string
	absPath     string
	modTime     time.Time
	cached      *CacheFile
	supportsSym bool
}

type renderOption struct {
	include     bool
	showSymbols bool
}
