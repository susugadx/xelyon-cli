package search

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

type searchPathBasis struct {
	workdir   string
	target    string
	matchRoot string
}

type rawFileFilterSpec struct {
	primaryGlob string
	matchGlobs  []string
}

func newRawFileFilterSpec(primaryGlob string, extraGlobs ...string) rawFileFilterSpec {
	matchGlobs := make([]string, 0, 1+len(extraGlobs))
	matchGlobs = append(matchGlobs, primaryGlob)
	matchGlobs = append(matchGlobs, extraGlobs...)
	return rawFileFilterSpec{
		primaryGlob: primaryGlob,
		matchGlobs:  matchGlobs,
	}
}

var rawFileFilterSpecs = map[string]rawFileFilterSpec{
	"bash":       newRawFileFilterSpec("*.bash"),
	"c":          newRawFileFilterSpec("*.c", "*.h", "*.H", "*.[chH].in", "*.cats"),
	"cc":         newRawFileFilterSpec("*.cc"),
	"cjs":        newRawFileFilterSpec("*.cjs"),
	"cpp":        newRawFileFilterSpec("*.cpp", "*.cc", "*.cxx", "*.h", "*.hpp", "*.hxx"),
	"cs":         newRawFileFilterSpec("*.cs"),
	"csharp":     newRawFileFilterSpec("*.cs"),
	"css":        newRawFileFilterSpec("*.css"),
	"cxx":        newRawFileFilterSpec("*.cxx"),
	"elixir":     newRawFileFilterSpec("*.ex", "*.exs"),
	"env":        newRawFileFilterSpec("*.env"),
	"ex":         newRawFileFilterSpec("*.ex", "*.exs"),
	"exs":        newRawFileFilterSpec("*.exs"),
	"go":         newRawFileFilterSpec("*.go"),
	"h":          newRawFileFilterSpec("*.h"),
	"hpp":        newRawFileFilterSpec("*.hpp"),
	"html":       newRawFileFilterSpec("*.html"),
	"hxx":        newRawFileFilterSpec("*.hxx"),
	"java":       newRawFileFilterSpec("*.java", "*.jsp", "*.jspx", "*.properties"),
	"js":         newRawFileFilterSpec("*.js"),
	"javascript": newRawFileFilterSpec("*.js", "*.jsx", "*.mjs", "*.cjs"),
	"json":       newRawFileFilterSpec("*.json"),
	"jsx":        newRawFileFilterSpec("*.jsx"),
	"kt":         newRawFileFilterSpec("*.kt"),
	"kotlin":     newRawFileFilterSpec("*.kt", "*.kts"),
	"kts":        newRawFileFilterSpec("*.kts"),
	"lock":       newRawFileFilterSpec("*.lock"),
	"lua":        newRawFileFilterSpec("*.lua"),
	"md":         newRawFileFilterSpec("*.md"),
	"mjs":        newRawFileFilterSpec("*.mjs"),
	"mod":        newRawFileFilterSpec("*.mod"),
	"php":        newRawFileFilterSpec("*.php"),
	"proto":      newRawFileFilterSpec("*.proto"),
	"py":         newRawFileFilterSpec("*.py", "*.pyi"),
	"python":     newRawFileFilterSpec("*.py", "*.pyi"),
	"rb":         newRawFileFilterSpec("*.rb"),
	"rs":         newRawFileFilterSpec("*.rs"),
	"ruby":       newRawFileFilterSpec("*.rb"),
	"rust":       newRawFileFilterSpec("*.rs"),
	"sass":       newRawFileFilterSpec("*.sass"),
	"scala":      newRawFileFilterSpec("*.scala"),
	"scss":       newRawFileFilterSpec("*.scss"),
	"sh": newRawFileFilterSpec(
		"*.sh",
		"*.bash", "*.bashrc", "*.csh", "*.cshrc", "*.env", "*.ksh", "*.kshrc", "*.tcsh", "*.tcshrc", "*.zsh",
		".bash_login", ".bash_logout", ".bash_profile", ".bashrc", ".cshrc", ".env", ".kshrc", ".login", ".logout", ".profile",
		".tcshrc", ".zlogin", ".zlogout", ".zprofile", ".zshenv", ".zshrc",
		"bash_login", "bash_logout", "bash_profile", "bashrc", "profile", "zlogin", "zlogout", "zprofile", "zshenv", "zshrc",
	),
	"sum":        newRawFileFilterSpec("*.sum"),
	"swift":      newRawFileFilterSpec("*.swift"),
	"toml":       newRawFileFilterSpec("*.toml"),
	"ts":         newRawFileFilterSpec("*.ts"),
	"tsx":        newRawFileFilterSpec("*.tsx"),
	"txt":        newRawFileFilterSpec("*.txt"),
	"typescript": newRawFileFilterSpec("*.ts", "*.tsx"),
	"work":       newRawFileFilterSpec("*.work"),
	"xml":        newRawFileFilterSpec("*.xml"),
	"yaml":       newRawFileFilterSpec("*.yaml", "*.yml"),
	"yml":        newRawFileFilterSpec("*.yml"),
	"zsh":        newRawFileFilterSpec("*.zsh"),
}

var supportedBareFileExtensions = map[string]struct{}{
	"bash": {}, "c": {}, "cc": {}, "cjs": {}, "cpp": {}, "cs": {}, "css": {}, "cxx": {},
	"env": {}, "ex": {}, "exs": {}, "go": {}, "h": {}, "hpp": {}, "html": {}, "hxx": {},
	"java": {}, "js": {}, "json": {}, "jsx": {}, "kt": {}, "kts": {}, "lock": {}, "lua": {},
	"md": {}, "mjs": {}, "mod": {}, "php": {}, "proto": {}, "py": {}, "pyi": {}, "rb": {}, "rs": {},
	"sass": {}, "scala": {}, "scss": {}, "sh": {}, "sum": {}, "swift": {}, "toml": {}, "ts": {},
	"tsx": {}, "txt": {}, "work": {}, "xml": {}, "yaml": {}, "yml": {}, "zsh": {},
}

func ParseRawFileFilter(filter string) (string, string) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", ""
	}
	if containsGlobChar(filter) {
		return "", filter
	}
	return normalizeRawFileFilterToken(filter), ""
}

func MatchesRawFileFilter(filePath, filter string) bool {
	fileType, filePattern := ParseRawFileFilter(filter)
	return matchesFileFilterParts(filePath, fileType, filePattern)
}

func rawFileFilterToRipgrepArgs(fileType, filePattern string) []string {
	globs := rawFileFilterGlobs(fileType, filePattern)
	if len(globs) == 0 {
		return nil
	}

	args := make([]string, 0, len(globs)*2)
	for _, glob := range globs {
		args = append(args, "--glob", glob)
	}
	return args
}

func representativeRawFileFilterToken(fileType, filePattern string) string {
	if token := normalizeRawFileFilterToken(fileType); token != "" {
		return token
	}

	filePattern = strings.TrimSpace(filePattern)
	if filePattern == "" {
		return ""
	}

	ext := strings.TrimPrefix(path.Ext(path.Base(filePattern)), ".")
	return normalizeRawFileFilterToken(ext)
}

func SupportsBareFileExtension(ext string) bool {
	ext = normalizeRawFileFilterToken(ext)
	if ext == "" {
		return false
	}
	_, ok := supportedBareFileExtensions[ext]
	return ok
}

func matchesFileFilterParts(filePath, fileType, filePattern string) bool {
	cleanPath := cleanFileFilterPath(filePath)
	globs := rawFileFilterGlobs(fileType, filePattern)
	if len(globs) == 0 {
		return true
	}
	return matchesAnyFileFilterGlob(cleanPath, globs)
}

func fileTypeToGlob(fileType string) (string, bool) {
	globs, ok := fileTypeToGlobs(fileType)
	if !ok || len(globs) == 0 {
		return "", false
	}
	return globs[0], true
}

func fileTypeToGlobs(fileType string) ([]string, bool) {
	spec, ok := rawFileFilterSpecs[normalizeRawFileFilterToken(fileType)]
	if !ok {
		return nil, false
	}
	globs := append([]string(nil), spec.matchGlobs...)
	return globs, true
}

func matchGlobsForRawFileType(fileType string) []string {
	if spec, ok := rawFileFilterSpecs[normalizeRawFileFilterToken(fileType)]; ok {
		return spec.matchGlobs
	}

	token := normalizeRawFileFilterToken(fileType)
	if token == "" {
		return nil
	}
	return []string{"*." + token}
}

func rawFileFilterGlobs(fileType, filePattern string) []string {
	fileType = strings.TrimSpace(fileType)
	if fileType != "" {
		return matchGlobsForRawFileType(fileType)
	}

	filePattern = strings.TrimSpace(filePattern)
	if filePattern == "" {
		return nil
	}
	return []string{filePattern}
}

func searchFileFilterMatchPath(filePath string, searchPath string) string {
	return searchFileFilterMatchPathWithWorkspace(filePath, searchPath, "")
}

// searchFileFilterMatchPathWithWorkspace normalizes candidate paths onto the
// shared workspace-relative basis used by direct directory filters and search
// post-filters.
func searchFileFilterMatchPathWithWorkspace(filePath string, searchPath string, workspaceRoot string) string {
	return WorkspaceRelativeFileFilterPath(filePath, searchFileFilterMatchRootWithWorkspace(searchPath, workspaceRoot))
}

func searchFileFilterMatchRootWithWorkspace(searchPath string, workspaceRoot string) string {
	return resolveSearchPathBasisWithWorkspace(searchPath, workspaceRoot).matchRoot
}

// WorkspaceRelativeFileFilterPath converts file paths to the shared file_filter
// matching basis. Absolute paths are relativized to the workspace root when
// possible; relative paths are preserved as workspace-relative display paths.
func WorkspaceRelativeFileFilterPath(filePath string, workspaceRoot string) string {
	cleanPath := cleanFileFilterPath(filePath)
	if cleanPath == "" {
		return ""
	}
	if !filepath.IsAbs(filepath.Clean(cleanPath)) {
		return cleanPath
	}

	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return cleanPath
	}

	relPath, ok := relativeSearchFileFilterPath(workspaceRoot, cleanPath)
	if !ok {
		return cleanPath
	}
	return relPath
}

func relativeSearchFileFilterPath(rootPath, filePath string) (string, bool) {
	rootPath = filepath.Clean(rootPath)
	filePath = filepath.Clean(filePath)

	relPath, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		return "", false
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." || relPath == "" || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(relPath), true
}

func resolveSearchPathBasis(searchPath string) searchPathBasis {
	return resolveSearchPathBasisWithWorkspace(searchPath, "")
}

func resolveSearchPathBasisForOptions(opts SearchOptions) searchPathBasis {
	return resolveSearchPathBasisWithWorkspace(opts.Path, resolveSearchWorkspaceRoot(opts))
}

// resolveSearchPathBasisWithWorkspace keeps ripgrep/grep execution scoped to the
// requested path while preserving workspace-relative file_filter semantics when
// the path is an absolute path inside the current workspace.
func resolveSearchPathBasisWithWorkspace(searchPath string, workspaceRoot string) searchPathBasis {
	searchPath = strings.TrimSpace(searchPath)
	workspaceRoot = normalizeSearchWorkspaceRoot(workspaceRoot)
	if searchPath == "" {
		return searchPathBasis{target: "."}
	}

	cleanPath := filepath.Clean(searchPath)
	if !filepath.IsAbs(cleanPath) {
		return searchPathBasis{target: cleanPath}
	}

	if workspaceRoot != "" {
		if cleanPath == workspaceRoot {
			return searchPathBasis{
				workdir:   workspaceRoot,
				target:    ".",
				matchRoot: workspaceRoot,
			}
		}
		if relPath, ok := relativeSearchFileFilterPath(workspaceRoot, cleanPath); ok {
			return searchPathBasis{
				workdir:   workspaceRoot,
				target:    relPath,
				matchRoot: workspaceRoot,
			}
		}
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return searchPathBasis{target: cleanPath}
	}
	if info.IsDir() {
		return searchPathBasis{
			workdir:   cleanPath,
			target:    ".",
			matchRoot: cleanPath,
		}
	}

	root := filepath.Dir(cleanPath)
	return searchPathBasis{
		workdir:   root,
		target:    filepath.Base(cleanPath),
		matchRoot: root,
	}
}

func resolveSearchWorkspaceRoot(opts SearchOptions) string {
	for _, candidate := range []string{
		opts.ProjectMapRootPath,
		opts.InvocationCWD,
	} {
		if root := normalizeSearchWorkspaceRoot(candidate); root != "" {
			return root
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		return normalizeSearchWorkspaceRoot(cwd)
	}
	return ""
}

func normalizeSearchWorkspaceRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if absPath, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(path)
}

func normalizeRawFileFilterToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, ".")
	return strings.ToLower(token)
}

func cleanFileFilterPath(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filePath))
}

func matchesAnyFileFilterGlob(cleanPath string, globs []string) bool {
	if cleanPath == "" {
		return false
	}
	baseName := path.Base(cleanPath)
	for _, glob := range globs {
		glob = filepath.ToSlash(strings.TrimSpace(glob))
		if glob == "" {
			continue
		}
		if matched, err := path.Match(glob, baseName); err == nil && matched {
			return true
		}
		if matched, err := path.Match(glob, cleanPath); err == nil && matched {
			return true
		}
	}
	return false
}
