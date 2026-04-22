package search

import (
	"path"
	"strings"
)

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
