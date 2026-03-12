package repomap

import (
	"path/filepath"
	"regexp"
	"strings"
)

type languagePattern struct {
	Extensions []string
	Patterns   []string
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

var compiledPatterns = compilePatterns()

func compilePatterns() map[string][]*regexp.Regexp {
	compiled := make(map[string][]*regexp.Regexp)
	for _, def := range defaultPatterns {
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
	_, ok := compiledPatterns[extensionForPath(path)]
	return ok
}

func matchesSymbolPattern(path, line string) bool {
	regexps := compiledPatterns[extensionForPath(path)]
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

func normalizeSignature(line string) string {
	sig := strings.TrimSpace(line)
	sig = strings.TrimSuffix(sig, "{}")
	sig = strings.TrimSuffix(sig, " {}")
	sig = strings.TrimSuffix(sig, "{")
	sig = strings.TrimSuffix(sig, " {")
	sig = strings.TrimSpace(sig)
	return sig
}

func isTestFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(base))
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.HasPrefix(base, "test_") && ext == ".py":
		return true
	case strings.HasSuffix(base, ".test.ts"), strings.HasSuffix(base, ".test.tsx"),
		strings.HasSuffix(base, ".test.js"), strings.HasSuffix(base, ".test.jsx"),
		strings.HasSuffix(base, ".spec.ts"), strings.HasSuffix(base, ".spec.tsx"),
		strings.HasSuffix(base, ".spec.js"), strings.HasSuffix(base, ".spec.jsx"):
		return true
	default:
		return false
	}
}

func testSortBase(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "_test.go"):
		return strings.TrimSuffix(lower, "_test.go") + ".go"
	case strings.HasPrefix(lower, "test_") && strings.HasSuffix(lower, ".py"):
		return strings.TrimPrefix(lower, "test_")
	case strings.HasSuffix(lower, ".test.ts"):
		return strings.TrimSuffix(lower, ".test.ts") + ".ts"
	case strings.HasSuffix(lower, ".test.tsx"):
		return strings.TrimSuffix(lower, ".test.tsx") + ".tsx"
	case strings.HasSuffix(lower, ".test.js"):
		return strings.TrimSuffix(lower, ".test.js") + ".js"
	case strings.HasSuffix(lower, ".test.jsx"):
		return strings.TrimSuffix(lower, ".test.jsx") + ".jsx"
	case strings.HasSuffix(lower, ".spec.ts"):
		return strings.TrimSuffix(lower, ".spec.ts") + ".ts"
	case strings.HasSuffix(lower, ".spec.tsx"):
		return strings.TrimSuffix(lower, ".spec.tsx") + ".tsx"
	case strings.HasSuffix(lower, ".spec.js"):
		return strings.TrimSuffix(lower, ".spec.js") + ".js"
	case strings.HasSuffix(lower, ".spec.jsx"):
		return strings.TrimSuffix(lower, ".spec.jsx") + ".jsx"
	default:
		return lower
	}
}
