package repomap

import (
	"path/filepath"
	"time"
)

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

// NewProjectMap はプロジェクト構造マップを作成する。
func NewProjectMap(rootPath string, maxTokens int, additionalIgnoreDirs ...string) *ProjectMap {
	if rootPath == "" {
		rootPath = "."
	}
	if abs, err := filepath.Abs(rootPath); err == nil {
		rootPath = abs
	}

	pm := &ProjectMap{
		RootPath:  rootPath,
		MaxTokens: maxTokens,
	}
	pm.additionalIgnoreDirs = append(pm.additionalIgnoreDirs, additionalIgnoreDirs...)
	return pm
}

// Build はプロジェクトのファイル一覧とシンボル一覧を構築する。
func (pm *ProjectMap) Build() error {
	if err := pm.validateBuildPreconditions(); err != nil {
		return err
	}

	artifacts, err := pm.collectBuildArtifacts()
	if err != nil {
		return err
	}

	pm.Files = artifacts.entries
	pm.GitStatus = pm.loadGitStatus()

	_ = saveMapCache(pm.RootPath, artifacts.cache)
	return nil
}

// BuildManifest は軽量な manifest 用にファイル一覧と git status のみを構築する。
func (pm *ProjectMap) BuildManifest() error {
	if err := pm.validateBuildPreconditions(); err != nil {
		return err
	}

	paths, err := pm.listFiles()
	if err != nil {
		return err
	}

	entries := make([]*FileEntry, 0, len(paths))
	for _, relPath := range paths {
		entries = append(entries, &FileEntry{Path: relPath})
	}

	pm.Files = entries
	pm.GitStatus = pm.loadGitStatus()
	return nil
}

// Generate はプロジェクト構造マップを文字列に変換する。
func (pm *ProjectMap) Generate() string {
	if pm == nil {
		return ""
	}
	if len(pm.Files) == 0 && len(pm.GitStatus) == 0 {
		return ""
	}
	return newProjectMapBudgetReducer(pm).reduce()
}

// GenerateManifest は root manifest 寄りの軽量な Project Map を生成する。
func (pm *ProjectMap) GenerateManifest(prioritizedPaths []string) string {
	if pm == nil || len(pm.Files) == 0 {
		return ""
	}

	topDirs, topFiles := pm.collectTopLevelEntries()
	priorityFiles := pm.collectPriorityFiles(prioritizedPaths)
	return newProjectMapManifestBudgetReducer(pm, topDirs, topFiles, priorityFiles).reduce()
}

// GetSymbolCount は保持しているシンボル数を返す。
func (pm *ProjectMap) GetSymbolCount() int {
	if pm == nil {
		return 0
	}
	total := 0
	for _, file := range pm.Files {
		total += len(file.Symbols)
	}
	return total
}

// GetFileCount は保持しているファイル数を返す。
func (pm *ProjectMap) GetFileCount() int {
	if pm == nil {
		return 0
	}
	return len(pm.Files)
}
