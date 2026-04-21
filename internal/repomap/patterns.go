package repomap

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

type languagePattern struct {
	Extensions []string
	Patterns   []string
}

type languagePatternEngine struct {
	patternsByExtension map[string][]*regexp.Regexp
}

var defaultPatterns = []languagePattern{
	{
		Extensions: []string{".go"},
		Patterns: []string{
			`^func `,
			`^type [A-Za-z0-9_]+ (struct|interface)`,
			`^var [A-Za-z0-9_]+`,
			`^const [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs"},
		Patterns: []string{
			`^export (function|class|const|interface|type|enum|abstract class) `,
			`^export default (function|class) `,
			`^(async )?function [A-Za-z0-9_]+`,
			`^class [A-Za-z0-9_]+`,
			`^(const|let) [A-Za-z0-9_]+ = `,
			`^interface [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".py"},
		Patterns: []string{
			`^(async )?def [A-Za-z0-9_]+`,
			`^class [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".rs"},
		Patterns: []string{
			`^(pub )?(async )?fn [A-Za-z0-9_]+`,
			`^(pub )?struct [A-Za-z0-9_]+`,
			`^(pub )?enum [A-Za-z0-9_]+`,
			`^(pub )?trait [A-Za-z0-9_]+`,
			`^impl(<[^>]*>)? [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".java", ".kt", ".kts"},
		Patterns: []string{
			`^(public |private |protected )?(static )?(abstract )?(class|interface|enum|record) [A-Za-z0-9_]+`,
			`^(public |private |protected )?(static )?(abstract )?[A-Za-z0-9_<>,\[\]?]+\s+[A-Za-z0-9_]+\(`,
			`^(fun|suspend fun) [A-Za-z0-9_]+`,
			`^(data |sealed |value )?class [A-Za-z0-9_]+`,
			`^object [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".rb"},
		Patterns: []string{
			`^\s*def [A-Za-z0-9_]+`,
			`^\s*class [A-Za-z0-9_]+`,
			`^\s*module [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".php"},
		Patterns: []string{
			`^\s*(public |private |protected )?(static )?function [A-Za-z0-9_]+`,
			`^\s*(abstract )?(class|interface|trait|enum) [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".c", ".h", ".cpp", ".hpp", ".cc"},
		Patterns: []string{
			`^(typedef )?(struct|class|enum|union) [A-Za-z0-9_]+`,
			`^#define [A-Za-z0-9_]+`,
			`^namespace [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".swift"},
		Patterns: []string{
			`^\s*(public |private |internal |open )?(func|class|struct|enum|protocol|extension) [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".scala"},
		Patterns: []string{
			`^\s*(def|class|object|trait|case class|case object|sealed trait) [A-Za-z0-9_]+`,
		},
	},
	{
		Extensions: []string{".sh", ".bash", ".zsh"},
		Patterns: []string{
			`^([A-Za-z0-9_]+)\s*\(\)`,
			`^function [A-Za-z0-9_]+`,
		},
	},
}

var defaultPatternEngine = newLanguagePatternEngine(defaultPatterns)

func newLanguagePatternEngine(patternDefinitions []languagePattern) *languagePatternEngine {
	return &languagePatternEngine{
		patternsByExtension: compilePatternDefinitions(patternDefinitions),
	}
}

func compilePatternDefinitions(patternDefinitions []languagePattern) map[string][]*regexp.Regexp {
	compiled := make(map[string][]*regexp.Regexp)
	for _, def := range patternDefinitions {
		var regexps []*regexp.Regexp
		for _, pattern := range def.Patterns {
			regexps = append(regexps, regexp.MustCompile(pattern))
		}
		for _, ext := range def.Extensions {
			compiled[ext] = regexps
		}
	}
	return compiled
}

func (e *languagePatternEngine) supports(path string) bool {
	if e == nil {
		return false
	}
	_, ok := e.patternsByExtension[extensionForPath(path)]
	return ok
}

func (e *languagePatternEngine) matches(path, line string) bool {
	if e == nil {
		return false
	}
	regexps := e.patternsByExtension[extensionForPath(path)]
	if len(regexps) == 0 {
		return false
	}
	if isCommentLikeLine(path, line) {
		return false
	}
	for _, re := range regexps {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func extensionForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		return ext
	}
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "dockerfile":
		return "dockerfile"
	default:
		return ""
	}
}

func supportsSymbols(path string) bool {
	if ast.IsSupportedFile(path) {
		return true
	}
	return defaultPatternEngine.supports(path)
}

func matchesSymbolPattern(path, line string) bool {
	return defaultPatternEngine.matches(path, line)
}

func isCommentLikeLine(path, line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	switch extensionForPath(path) {
	case ".py", ".rb", ".sh", ".bash", ".zsh":
		return strings.HasPrefix(trimmed, "#")
	default:
		return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*")
	}
}
