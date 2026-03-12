package repomap

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const defaultMaxTokens = 4000

var defaultIgnoreDirs = []string{
	".git", "node_modules", "vendor",
	".next", "__pycache__", ".venv",
	"dist", "build", ".idea", ".vscode",
}

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
	Line      int    `json:"line"`
	Signature string `json:"signature"`
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

func (pm *ProjectMap) listFiles() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"--files"}
	for _, dir := range pm.ignoreDirs() {
		if dir == "" {
			continue
		}
		args = append(args,
			"--glob", "!"+dir+"/**",
			"--glob", "!**/"+dir+"/**",
		)
	}

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	cmd.Dir = pm.RootPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stdout.Len() == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("rg --files failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var paths []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		paths = append(paths, filepath.Clean(filepath.ToSlash(line)))
	}
	sort.Strings(paths)
	return paths, nil
}

func (pm *ProjectMap) buildFileStates(paths []string, cache *MapCache) ([]fileState, error) {
	states := make([]fileState, 0, len(paths))
	for _, relPath := range paths {
		absPath := filepath.Join(pm.RootPath, relPath)
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", relPath, err)
		}

		state := fileState{
			path:        relPath,
			absPath:     absPath,
			modTime:     info.ModTime().UTC(),
			supportsSym: supportsSymbols(relPath),
		}
		if cached, ok := cache.Files[relPath]; ok && cached != nil && state.modTime.Equal(cached.ModTime) {
			state.cached = cloneCacheFile(cached)
		}
		states = append(states, state)
	}
	return states, nil
}

func (pm *ProjectMap) scanSymbols(states []fileState) (map[string][]Symbol, error) {
	targetsByExt := make(map[string][]string)
	for _, state := range states {
		if state.cached != nil || !state.supportsSym {
			continue
		}
		ext := extensionForPath(state.path)
		if ext == "" {
			continue
		}
		targetsByExt[ext] = append(targetsByExt[ext], state.path)
	}
	if len(targetsByExt) == 0 {
		return map[string][]Symbol{}, nil
	}

	results := make(map[string][]Symbol)
	seen := make(map[string]map[int]struct{})

	for _, def := range defaultPatterns {
		var targets []string
		for _, ext := range def.Extensions {
			targets = append(targets, targetsByExt[ext]...)
		}
		if len(targets) == 0 {
			continue
		}

		symbols, err := pm.runRgAndParse(def, targets, seen)
		if err != nil {
			return nil, err
		}
		for path, syms := range symbols {
			results[path] = append(results[path], syms...)
		}
	}

	for path := range results {
		sort.Slice(results[path], func(i, j int) bool {
			return results[path][i].Line < results[path][j].Line
		})
	}
	return results, nil
}

func (pm *ProjectMap) runRgAndParse(def languagePattern, targets []string, seen map[string]map[int]struct{}) (map[string][]Symbol, error) {
	args := []string{"-n", "-H", "--color", "never"}
	for _, pattern := range def.Patterns {
		args = append(args, "-e", pattern)
	}
	for _, ext := range def.Extensions {
		args = append(args, "--glob", "*"+ext)
	}
	args = append(args, "--")
	args = append(args, targets...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	cmd.Dir = pm.RootPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stdout.Len() == 0 {
			return map[string][]Symbol{}, nil
		}
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("rg symbol scan failed: %s", strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("rg symbol scan failed: %w", err)
	}

	results := make(map[string][]Symbol)
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		path := filepath.Clean(filepath.ToSlash(parts[0]))
		lineNum, convErr := strconv.Atoi(parts[1])
		if convErr != nil {
			continue
		}
		content := parts[2]
		if !matchesSymbolPattern(path, content) {
			continue
		}

		if seen[path] == nil {
			seen[path] = make(map[int]struct{})
		}
		if _, ok := seen[path][lineNum]; ok {
			continue
		}
		seen[path][lineNum] = struct{}{}

		results[path] = append(results[path], Symbol{
			Line:      lineNum,
			Signature: normalizeSignature(content),
		})
	}

	return results, nil
}

func (pm *ProjectMap) loadGitStatus() []GitChange {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = pm.RootPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil
	}

	var changes []GitChange
	for _, line := range strings.Split(stdout.String(), "\n") {
		if len(line) < 3 {
			continue
		}
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		if idx := strings.LastIndex(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		if status == "" || path == "" {
			continue
		}
		changes = append(changes, GitChange{
			Status: status,
			Path:   filepath.ToSlash(path),
		})
	}
	return changes
}

func (pm *ProjectMap) render(options []renderOption, omittedFiles int) string {
	var b strings.Builder
	b.WriteString("## Project Map\n\n")

	pathIndex := make(map[string]int, len(pm.Files))
	for i, file := range pm.Files {
		if file != nil {
			pathIndex[file.Path] = i
		}
	}

	grouped := make(map[string][]*FileEntry)
	var dirs []string
	for i, file := range pm.Files {
		if file == nil || !options[i].include {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(file.Path))
		if dir == "." {
			dir = "./"
		} else {
			dir += "/"
		}
		if _, ok := grouped[dir]; !ok {
			dirs = append(dirs, dir)
		}
		grouped[dir] = append(grouped[dir], file)
	}
	sort.Strings(dirs)

	for dirIndex, dir := range dirs {
		if dirIndex > 0 {
			b.WriteString("\n")
		}
		b.WriteString("📂 ")
		b.WriteString(dir)
		b.WriteString("\n")

		files := grouped[dir]
		sort.Slice(files, func(i, j int) bool {
			return compareFileEntryPath(files[i].Path, files[j].Path)
		})

		for fileIndex, file := range files {
			connector := "├── "
			symbolPrefix := "│     "
			if fileIndex == len(files)-1 {
				connector = "└── "
				symbolPrefix = "      "
			}

			fmt.Fprintf(&b, "%s📄 %s (%d lines)\n", connector, filepath.Base(file.Path), file.LineCount)
			idx, ok := pathIndex[file.Path]
			if !ok || !options[idx].showSymbols {
				continue
			}
			for _, symbol := range file.Symbols {
				fmt.Fprintf(&b, "%s%d: %s\n", symbolPrefix, symbol.Line, symbol.Signature)
			}
		}
	}

	if omittedFiles > 0 {
		if len(dirs) > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "... (%d more files, truncated to fit token budget)\n", omittedFiles)
	}

	if len(pm.GitStatus) > 0 {
		if len(dirs) > 0 || omittedFiles > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## Uncommitted Changes\n")
		for _, change := range pm.GitStatus {
			fmt.Fprintf(&b, "  %s %s\n", change.Status, change.Path)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func (pm *ProjectMap) fitsBudget(text string) bool {
	maxTokens := pm.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	return token.EstimateTokenCount(text) <= maxTokens
}

func (pm *ProjectMap) ignoreDirs() []string {
	seen := make(map[string]struct{})
	var dirs []string
	for _, dir := range defaultIgnoreDirs {
		dir = normalizeIgnoreDir(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	for _, dir := range pm.additionalIgnoreDirs {
		dir = normalizeIgnoreDir(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	return dirs
}

func normalizeIgnoreDir(dir string) string {
	dir = filepath.ToSlash(strings.TrimSpace(dir))
	dir = strings.Trim(dir, "/")
	return dir
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	reader := bufio.NewReaderSize(f, 32*1024)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return count, nil
			}
			return 0, err
		}
		if b == '\n' {
			count++
		}
	}
}

func compareFileEntryPath(leftPath, rightPath string) bool {
	leftName := filepath.Base(leftPath)
	rightName := filepath.Base(rightPath)
	leftBase := testSortBase(leftName)
	rightBase := testSortBase(rightName)
	if leftBase != rightBase {
		return leftBase < rightBase
	}
	leftTest := isTestFile(leftName)
	rightTest := isTestFile(rightName)
	if leftTest != rightTest {
		return !leftTest
	}
	return strings.ToLower(leftName) < strings.ToLower(rightName)
}

func sortFileEntries(entries []*FileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		leftDir := filepath.ToSlash(filepath.Dir(entries[i].Path))
		rightDir := filepath.ToSlash(filepath.Dir(entries[j].Path))
		if leftDir != rightDir {
			return leftDir < rightDir
		}
		return compareFileEntryPath(entries[i].Path, entries[j].Path)
	})
}

func directoryDepth(path string) int {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return 0
	}
	return strings.Count(dir, "/") + 1
}
