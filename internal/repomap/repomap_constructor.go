package repomap

import "path/filepath"

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
