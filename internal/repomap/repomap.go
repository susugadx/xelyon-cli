package repomap

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
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
	if pm == nil {
		return fmt.Errorf("project map is nil")
	}
	if !common.IsRipgrepAvailable() {
		return fmt.Errorf("ripgrep (rg) is required")
	}
	paths, err := pm.listFiles()
	if err != nil {
		return err
	}

	cache, err := loadMapCache(pm.RootPath)
	if err != nil {
		cache = &MapCache{RootPath: pm.RootPath, Files: map[string]*CacheFile{}}
	}
	if cache.Files == nil {
		cache.Files = map[string]*CacheFile{}
	}

	states, err := pm.buildFileStates(paths, cache)
	if err != nil {
		return err
	}

	symbolsByFile, err := pm.scanSymbols(states)
	if err != nil {
		return err
	}

	newCache := &MapCache{
		RootPath: pm.RootPath,
		Files:    make(map[string]*CacheFile, len(states)),
	}

	entries := make([]*FileEntry, 0, len(states))
	for _, state := range states {
		if state.cached != nil && state.modTime.Equal(state.cached.ModTime) {
			entry := &FileEntry{
				Path:      state.path,
				LineCount: state.cached.LineCount,
			}
			if len(state.cached.Symbols) > 0 {
				entry.Symbols = append([]Symbol(nil), state.cached.Symbols...)
			}
			entries = append(entries, entry)
			newCache.Files[state.path] = cloneCacheFile(state.cached)
			continue
		}

		lineCount, err := countLines(state.absPath)
		if err != nil {
			return fmt.Errorf("count lines %s: %w", state.path, err)
		}

		entry := &FileEntry{
			Path:      state.path,
			LineCount: lineCount,
			Symbols:   symbolsByFile[state.path],
		}
		entries = append(entries, entry)
		newCache.Files[state.path] = &CacheFile{
			ModTime:   state.modTime,
			LineCount: lineCount,
			Symbols:   append([]Symbol(nil), entry.Symbols...),
		}
	}

	sortFileEntries(entries)
	pm.Files = entries
	pm.GitStatus = pm.loadGitStatus()

	_ = saveMapCache(pm.RootPath, newCache)
	return nil
}

// BuildManifest は軽量な manifest 用にファイル一覧と git status のみを構築する。
func (pm *ProjectMap) BuildManifest() error {
	if pm == nil {
		return fmt.Errorf("project map is nil")
	}
	if !common.IsRipgrepAvailable() {
		return fmt.Errorf("ripgrep (rg) is required")
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

	options := make([]renderOption, len(pm.Files))
	for i := range pm.Files {
		options[i] = renderOption{include: true, showSymbols: true}
	}

	result := pm.render(options, 0)
	if pm.fitsBudget(result) {
		return result
	}

	for i, file := range pm.Files {
		if isTestFile(file.Path) && len(file.Symbols) > 0 {
			options[i].showSymbols = false
		}
	}
	result = pm.render(options, 0)
	if pm.fitsBudget(result) {
		return result
	}

	var order []int
	for i, file := range pm.Files {
		if !options[i].include {
			continue
		}
		order = append(order, i)
		if file == nil {
			continue
		}
	}
	sort.Slice(order, func(i, j int) bool {
		left := pm.Files[order[i]]
		right := pm.Files[order[j]]
		leftDepth := directoryDepth(left.Path)
		rightDepth := directoryDepth(right.Path)
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return left.Path > right.Path
	})

	omitted := 0
	for _, idx := range order {
		options[idx].include = false
		omitted++
		result = pm.render(options, omitted)
		if pm.fitsBudget(result) {
			return result
		}
	}

	return pm.render(options, omitted)
}

// GenerateManifest は root manifest 寄りの軽量な Project Map を生成する。
func (pm *ProjectMap) GenerateManifest(prioritizedPaths []string) string {
	if pm == nil || len(pm.Files) == 0 {
		return ""
	}

	const (
		maxTopLevelDirs  = 8
		maxTopLevelFiles = 8
		maxPriorityFiles = 10
	)

	topDirs, topFiles := pm.collectTopLevelEntries()
	priorityFiles := pm.collectPriorityFiles(prioritizedPaths)
	dirLimit := minInt(len(topDirs), maxTopLevelDirs)
	fileLimit := minInt(len(topFiles), maxTopLevelFiles)
	priorityLimit := minInt(len(priorityFiles), maxPriorityFiles)
	changeLimit := len(pm.GitStatus)

	for {
		result := renderManifest(topDirs, dirLimit, topFiles, fileLimit, priorityFiles, priorityLimit, pm.GitStatus, changeLimit)
		if result == "" || pm.fitsBudget(result) {
			return result
		}

		switch {
		case priorityLimit > 0:
			priorityLimit--
		case changeLimit > 0:
			changeLimit--
		case fileLimit > 0:
			fileLimit--
		case dirLimit > 0:
			dirLimit--
		default:
			return pm.generateManifestFallback(len(pm.Files), len(pm.GitStatus))
		}
	}
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
