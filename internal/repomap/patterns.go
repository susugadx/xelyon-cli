package repomap

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/ast"
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

type signaturePattern struct {
	re   *regexp.Regexp
	kind string
}

var signaturePatterns = []signaturePattern{
	{re: regexp.MustCompile(`^func\s+\([^)]*\)\s*([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]+\])?\s*\(`), kind: "method"},
	{re: regexp.MustCompile(`^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]+\])?\s*\(`), kind: "function"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\b`), kind: "struct"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+interface\b`), kind: "interface"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type"},
	{re: regexp.MustCompile(`^const\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const"},
	{re: regexp.MustCompile(`^var\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "var"},
	{re: regexp.MustCompile(`^export\s+default\s+function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^export\s+default\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^export\s+(?:abstract\s+class|class)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^export\s+function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^export\s+const\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const"},
	{re: regexp.MustCompile(`^export\s+interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface"},
	{re: regexp.MustCompile(`^export\s+type\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type"},
	{re: regexp.MustCompile(`^export\s+enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum"},
	{re: regexp.MustCompile(`^(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^(?:const|let)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "var"},
	{re: regexp.MustCompile(`^interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface"},
	{re: regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function"},
	{re: regexp.MustCompile(`^(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^(?:pub\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct"},
	{re: regexp.MustCompile(`^(?:pub\s+)?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum"},
	{re: regexp.MustCompile(`^(?:pub\s+)?trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait"},
	{re: regexp.MustCompile(`^impl(?:<[^>]*>)?\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "impl"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?(?:class|record) ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?interface ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?enum ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?[A-Za-z0-9_<>,\[\]?]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function"},
	{re: regexp.MustCompile(`^(?:suspend\s+)?fun\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^(?:data |sealed |value )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object"},
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*[!?=]?)\b`), kind: "function"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_:]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^\s*module\s+([A-Za-z_][A-Za-z0-9_:]*)\b`), kind: "module"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected )?(?:static )?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^\s*(?:abstract )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^\s*(?:abstract )?interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface"},
	{re: regexp.MustCompile(`^\s*trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait"},
	{re: regexp.MustCompile(`^\s*enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum"},
	{re: regexp.MustCompile(`^(?:typedef\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct"},
	{re: regexp.MustCompile(`^(?:typedef\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^(?:typedef\s+)?(?:enum|union)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type"},
	{re: regexp.MustCompile(`^#define\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const"},
	{re: regexp.MustCompile(`^namespace\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "namespace"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?func\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?protocol\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?extension\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "impl"},
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^\s*object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object"},
	{re: regexp.MustCompile(`^\s*trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait"},
	{re: regexp.MustCompile(`^\s*case\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class"},
	{re: regexp.MustCompile(`^\s*case\s+object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object"},
	{re: regexp.MustCompile(`^\s*sealed\s+trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait"},
	{re: regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\)`), kind: "function"},
	{re: regexp.MustCompile(`^function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function"},
}

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
	if ast.IsSupportedFile(path) {
		return true
	}
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

func signatureMetadataForPath(path, sig string) (string, string, bool) {
	name, kind, ok := extractSignatureMetadata(sig)
	if !ok {
		return "", "", false
	}
	exported := ast.IsSupportedFile(path) && isExportedName(name)
	return name, kind, exported
}

func extractSignatureMetadata(sig string) (string, string, bool) {
	for _, pattern := range signaturePatterns {
		matches := pattern.re.FindStringSubmatch(sig)
		if len(matches) != 2 {
			continue
		}
		name := strings.TrimSpace(matches[1])
		if name == "" {
			continue
		}
		return name, pattern.kind, true
	}
	return "", "", false
}

func isExportedName(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	return unicode.IsUpper(r)
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
