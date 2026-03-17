package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const (
	maxEntries           = 200
	maxRootDirsShown     = 8
	maxRootFilesShown    = 8
	maxRootSubtreesShown = 6
	maxNestedDirsShown   = 4
	maxNestedFilesShown  = 4
	maxNestedSubtrees    = 3
)

type listDirFileSummary struct {
	name string
	size int64
}

type listDirSection struct {
	relPath     string
	totalDirs   int
	totalFiles  int
	dirs        []string
	files       []listDirFileSummary
	moreDirs    int
	moreFiles   int
	subtrees    []*listDirSection
	moreSubtree int
	readErr     error
}

// ExecuteListDir はディレクトリ一覧を取得
func ExecuteListDir(path string, depth int) string {
	return ExecuteListDirWithRuntime(config.DefaultConfig(), nil, path, depth)
}

// ExecuteListDirWithRuntime は runtime 設定を指定してディレクトリ一覧を取得する。
func ExecuteListDirWithRuntime(cfg *config.Config, cache tools.ToolCacheInterface, path string, depth int) string {
	if depth <= 0 {
		depth = 1
	}
	if depth > 3 {
		depth = 3
	}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	cachePath := absPath + "::depth=" + strconv.Itoa(depth)

	projectCfg := config.LoadProjectConfig()
	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, projectCfg)
	matcher := pathmatch.NewMatcher(ignorePatterns)

	rootPath := absPath
	if projectCfg != nil && projectCfg.FilePath != "" {
		rootPath = filepath.Dir(projectCfg.FilePath)
	}

	if cache != nil {
		if cached, hit := cache.GetDir(cachePath); hit {
			return cached
		}
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if !info.IsDir() {
		return fmt.Sprintf("Error: %s is not a directory", absPath)
	}

	section := summarizeListDir(absPath, rootPath, "", depth, matcher, &listDirBudget{remainingEntries: maxEntries}, true)
	lines := renderListDirSummary(absPath, depth, section)
	result := strings.Join(lines, "\n")

	if cache != nil {
		cache.SetDir(cachePath, result)
	}

	return result
}

type listDirBudget struct {
	remainingEntries int
}

func summarizeListDir(dirPath, rootPath, relPath string, remain int, matcher *pathmatch.Matcher, budget *listDirBudget, isRoot bool) *listDirSection {
	section := &listDirSection{relPath: relPath}
	if budget.remainingEntries <= 0 {
		return section
	}

	rawEntries, err := os.ReadDir(dirPath)
	if err != nil {
		section.readErr = err
		return section
	}

	var dirs []os.DirEntry
	var files []os.DirEntry
	for _, entry := range rawEntries {
		childPath := filepath.Join(dirPath, entry.Name())
		childRelPath, err := filepath.Rel(rootPath, childPath)
		if err == nil && matcher != nil && matcher.Match(filepath.ToSlash(childRelPath), entry.IsDir()) {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, entry)
			continue
		}
		files = append(files, entry)
	}

	section.totalDirs = len(dirs)
	section.totalFiles = len(files)
	budget.remainingEntries -= len(dirs) + len(files)

	dirLimit := maxNestedDirsShown
	fileLimit := maxNestedFilesShown
	subtreeLimit := maxNestedSubtrees
	if isRoot {
		dirLimit = maxRootDirsShown
		fileLimit = maxRootFilesShown
		subtreeLimit = maxRootSubtreesShown
	}

	section.dirs, section.moreDirs = summarizeDirNames(dirs, dirLimit)
	section.files, section.moreFiles = summarizeFileNames(files, fileLimit)

	if remain <= 1 || budget.remainingEntries <= 0 || len(dirs) == 0 {
		return section
	}

	expandCount := minInt(len(dirs), subtreeLimit)
	for i := 0; i < expandCount && budget.remainingEntries > 0; i++ {
		dirName := dirs[i].Name()
		childRelPath := joinListDirRelPath(relPath, dirName)
		childPath := filepath.Join(dirPath, dirName)
		section.subtrees = append(section.subtrees, summarizeListDir(childPath, rootPath, childRelPath, remain-1, matcher, budget, false))
	}
	if len(dirs) > len(section.subtrees) {
		section.moreSubtree = len(dirs) - len(section.subtrees)
	}

	return section
}

func summarizeDirNames(entries []os.DirEntry, limit int) ([]string, int) {
	shown := minInt(len(entries), limit)
	result := make([]string, 0, shown)
	for i := 0; i < shown; i++ {
		result = append(result, entries[i].Name()+"/")
	}
	if len(entries) <= shown {
		return result, 0
	}
	return result, len(entries) - shown
}

func summarizeFileNames(entries []os.DirEntry, limit int) ([]listDirFileSummary, int) {
	shown := minInt(len(entries), limit)
	result := make([]listDirFileSummary, 0, shown)
	for i := 0; i < shown; i++ {
		entry := entries[i]
		summary := listDirFileSummary{name: entry.Name()}
		if info, err := entry.Info(); err == nil {
			summary.size = info.Size()
		}
		result = append(result, summary)
	}
	if len(entries) <= shown {
		return result, 0
	}
	return result, len(entries) - shown
}

func renderListDirSummary(absPath string, depth int, section *listDirSection) []string {
	lines := []string{
		fmt.Sprintf("📂 %s", absPath),
		fmt.Sprintf("summary: depth=%d, dirs=%d, files=%d", depth, section.totalDirs, section.totalFiles),
	}
	if section.readErr != nil {
		return append(lines, "Error: failed to read directory")
	}
	appendSectionSummary(&lines, "", section, depth > 1)
	return lines
}

func appendSectionSummary(lines *[]string, indent string, section *listDirSection, includeSubtrees bool) {
	if len(section.dirs) > 0 {
		*lines = append(*lines, indent+"dirs: "+formatListDirNames(section.dirs, section.moreDirs))
	}
	if len(section.files) > 0 {
		*lines = append(*lines, indent+"files: "+formatListDirFiles(section.files, section.moreFiles))
	}
	if !includeSubtrees || len(section.subtrees) == 0 {
		return
	}

	subtreeSummary := fmt.Sprintf("subtrees: %d shown", len(section.subtrees))
	if section.moreSubtree > 0 {
		subtreeSummary = fmt.Sprintf("subtrees: %d shown (+%d more)", len(section.subtrees), section.moreSubtree)
	}
	*lines = append(*lines, indent+subtreeSummary)
	for _, child := range section.subtrees {
		renderListDirSection(lines, indent, child)
	}
}

func renderListDirSection(lines *[]string, indent string, section *listDirSection) {
	prefix := indent + "- "
	if section.readErr != nil {
		*lines = append(*lines, fmt.Sprintf("%s%s -> [error reading directory]", prefix, section.relPath))
		return
	}

	*lines = append(*lines, fmt.Sprintf("%s%s -> dirs=%d, files=%d", prefix, section.relPath, section.totalDirs, section.totalFiles))
	appendSectionSummary(lines, indent+"  ", section, len(section.subtrees) > 0)
}

func formatListDirNames(names []string, more int) string {
	if more == 0 {
		return strings.Join(names, ", ")
	}
	return strings.Join(names, ", ") + fmt.Sprintf(", (+%d more)", more)
}

func formatListDirFiles(files []listDirFileSummary, more int) string {
	parts := make([]string, 0, len(files)+1)
	for _, file := range files {
		parts = append(parts, fmt.Sprintf("%s (%d bytes)", file.name, file.size))
	}
	if more > 0 {
		parts = append(parts, fmt.Sprintf("(+%d more)", more))
	}
	return strings.Join(parts, ", ")
}

func joinListDirRelPath(parent, name string) string {
	joined := filepath.ToSlash(filepath.Join(parent, name))
	if joined == "." || joined == "" {
		joined = name
	}
	return joined + "/"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
