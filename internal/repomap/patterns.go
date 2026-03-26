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
	lang string // 対象言語（"go","js","py","rs","java","rb","php","c","swift","scala","sh"、"" で全言語）
}

var signaturePatterns = []signaturePattern{
	// Go
	{re: regexp.MustCompile(`^func\s+\([^)]*\)\s*([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]+\])?\s*\(`), kind: "method", lang: "go"},
	{re: regexp.MustCompile(`^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]+\])?\s*\(`), kind: "function", lang: "go"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\b`), kind: "struct", lang: "go"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+interface\b`), kind: "interface", lang: "go"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type", lang: "go"},
	{re: regexp.MustCompile(`^const\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const", lang: "go"},
	{re: regexp.MustCompile(`^var\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "var", lang: "go"},
	// JS/TS
	{re: regexp.MustCompile(`^export\s+default\s+(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^export\s+default\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "js"},
	{re: regexp.MustCompile(`^export\s+(?:abstract\s+class|class)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "js"},
	{re: regexp.MustCompile(`^export\s+(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^export\s+const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s+)?\([^)]*\)(?:\s*:\s*.+)?\s*=>`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^export\s+const\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const", lang: "js"},
	{re: regexp.MustCompile(`^export\s+interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "js"},
	{re: regexp.MustCompile(`^export\s+type\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type", lang: "js"},
	{re: regexp.MustCompile(`^export\s+enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "js"},
	{re: regexp.MustCompile(`^(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "js"},
	{re: regexp.MustCompile(`^(?:const|let)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s+)?\([^)]*\)(?:\s*:\s*.+)?\s*=>`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^(?:const|let)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "var", lang: "js"},
	{re: regexp.MustCompile(`^interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "js"},
	// Python
	{re: regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function", lang: "py"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "py"},
	// Rust
	{re: regexp.MustCompile(`^(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "rs"},
	{re: regexp.MustCompile(`^(?:pub\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "rs"},
	{re: regexp.MustCompile(`^(?:pub\s+)?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "rs"},
	{re: regexp.MustCompile(`^(?:pub\s+)?trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "rs"},
	{re: regexp.MustCompile(`^impl(?:<[^>]*>)?\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "impl", lang: "rs"},
	// Java/Kotlin
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?(?:class|record) ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "java"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?interface ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "java"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?enum ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "java"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?[A-Za-z0-9_<>,\[\]?]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function", lang: "java"},
	{re: regexp.MustCompile(`^(?:suspend\s+)?fun\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "java"},
	{re: regexp.MustCompile(`^(?:data |sealed |value )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "java"},
	{re: regexp.MustCompile(`^object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object", lang: "java"},
	// Ruby
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*[!?=]?)\b`), kind: "function", lang: "rb"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_:]*)\b`), kind: "class", lang: "rb"},
	{re: regexp.MustCompile(`^\s*module\s+([A-Za-z_][A-Za-z0-9_:]*)\b`), kind: "module", lang: "rb"},
	// PHP
	{re: regexp.MustCompile(`^\s*(?:public |private |protected )?(?:static )?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "php"},
	{re: regexp.MustCompile(`^\s*(?:abstract )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "php"},
	{re: regexp.MustCompile(`^\s*(?:abstract )?interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "php"},
	{re: regexp.MustCompile(`^\s*trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "php"},
	{re: regexp.MustCompile(`^\s*enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "php"},
	// C#
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:(?:static |abstract |sealed |partial )*)?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:static )?interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:readonly )?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:(?:static |virtual |abstract |override |async )*)?[A-Za-z0-9_<>,\[\]?]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function", lang: "csharp"},
	// C/C++
	{re: regexp.MustCompile(`^(?:typedef\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "c"},
	{re: regexp.MustCompile(`^(?:typedef\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "c"},
	{re: regexp.MustCompile(`^(?:typedef\s+)?(?:enum|union)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type", lang: "c"},
	{re: regexp.MustCompile(`^#define\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const", lang: "c"},
	{re: regexp.MustCompile(`^namespace\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "namespace", lang: "c"},
	// Swift
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?func\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?protocol\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?extension\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "impl", lang: "swift"},
	// Scala
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "scala"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "scala"},
	{re: regexp.MustCompile(`^\s*object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object", lang: "scala"},
	{re: regexp.MustCompile(`^\s*trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "scala"},
	{re: regexp.MustCompile(`^\s*case\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "scala"},
	{re: regexp.MustCompile(`^\s*case\s+object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object", lang: "scala"},
	{re: regexp.MustCompile(`^\s*sealed\s+trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "scala"},
	// Elixir
	{re: regexp.MustCompile(`^\s*defmodule\s+([A-Za-z_][A-Za-z0-9_.]*)\b`), kind: "module", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_!?]*)\b`), kind: "function", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*defp\s+([A-Za-z_][A-Za-z0-9_!?]*)\b`), kind: "function", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*defprotocol\s+([A-Za-z_][A-Za-z0-9_.]*)\b`), kind: "protocol", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*defmacro\s+([A-Za-z_][A-Za-z0-9_!?]*)\b`), kind: "macro", lang: "elixir"},
	// Lua
	{re: regexp.MustCompile(`^(?:local\s+)?function\s+([A-Za-z_][A-Za-z0-9_.]*)\s*\(`), kind: "function", lang: "lua"},
	{re: regexp.MustCompile(`^(?:local\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*function\s*\(`), kind: "function", lang: "lua"},
	// Shell
	{re: regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\)`), kind: "function", lang: "sh"},
	{re: regexp.MustCompile(`^function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "sh"},
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
	name, kind, ok := extractSignatureMetadataForLang(sig, patternLangForPath(path))
	if !ok {
		return "", "", false
	}
	exported := ast.IsSupportedFile(path) && isExportedName(name)
	return name, kind, exported
}

func extractSignatureMetadata(sig string) (string, string, bool) {
	if strings.Contains(sig, "=>") {
		if name, kind, ok := extractSignatureMetadataForLang(sig, "js"); ok {
			return name, kind, true
		}
	}
	return extractSignatureMetadataForLang(sig, "")
}

func extractSignatureMetadataForLang(sig, lang string) (string, string, bool) {
	if lang == "" || lang == "js" {
		if name, kind, ok := extractJSArrowFunctionMetadata(sig); ok {
			return name, kind, true
		}
	}

	for _, pattern := range signaturePatterns {
		if lang != "" && pattern.lang != "" && pattern.lang != lang {
			continue
		}
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

func extractJSArrowFunctionMetadata(sig string) (string, string, bool) {
	trimmed := strings.TrimSpace(sig)
	for _, prefix := range []string{"export const ", "const ", "let "} {
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}

		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		nameEnd := 0
		for i, r := range rest {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) || r == '_' {
				nameEnd = i + 1
				continue
			}
			break
		}
		if nameEnd == 0 {
			return "", "", false
		}

		name := rest[:nameEnd]
		rest = strings.TrimSpace(rest[nameEnd:])
		if !strings.HasPrefix(rest, "=") {
			return "", "", false
		}

		rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
		if strings.HasPrefix(rest, "async ") {
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "async "))
		}
		if !strings.HasPrefix(rest, "(") {
			return "", "", false
		}

		closeIdx := strings.Index(rest, ")")
		if closeIdx < 0 {
			return "", "", false
		}
		if !strings.Contains(rest[closeIdx+1:], "=>") {
			return "", "", false
		}

		return name, "function", true
	}

	return "", "", false
}

func patternLangForPath(path string) string {
	switch extensionForPath(path) {
	case ".go":
		return "go"
	case ".py":
		return "py"
	case ".ts", ".tsx", ".js", ".jsx", ".mjs":
		return "js"
	case ".rs":
		return "rs"
	case ".java", ".kt", ".kts":
		return "java"
	case ".rb":
		return "rb"
	case ".php":
		return "php"
	case ".c", ".cpp", ".cc", ".h", ".hpp":
		return "c"
	case ".swift":
		return "swift"
	case ".scala":
		return "scala"
	case ".ex", ".exs":
		return "elixir"
	case ".lua":
		return "lua"
	case ".sh", ".bash", ".zsh":
		return "sh"
	default:
		return ""
	}
}

func isExportedName(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	return unicode.IsUpper(r)
}

func isTestFile(path string) bool {
	origBase := filepath.Base(path)
	base := strings.ToLower(origBase)
	ext := strings.ToLower(filepath.Ext(base))
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.HasPrefix(base, "test_") && ext == ".py":
		return true
	case strings.HasSuffix(base, "_test.py"):
		return true
	case base == "conftest.py":
		return true
	case strings.HasSuffix(base, ".test.ts"), strings.HasSuffix(base, ".test.tsx"),
		strings.HasSuffix(base, ".test.js"), strings.HasSuffix(base, ".test.jsx"),
		strings.HasSuffix(base, ".spec.ts"), strings.HasSuffix(base, ".spec.tsx"),
		strings.HasSuffix(base, ".spec.js"), strings.HasSuffix(base, ".spec.jsx"):
		return true
	case isInTestsDir(path):
		return true
	case ext == ".java" && isTestSuffixName(origBase):
		return true
	case ext == ".kt" && isTestSuffixName(origBase):
		return true
	case ext == ".cs" && isTestSuffixName(origBase):
		return true
	case ext == ".swift" && isTestSuffixName(origBase):
		return true
	case ext == ".scala" && isTestSuffixName(origBase):
		return true
	case ext == ".php" && isTestSuffixName(origBase):
		return true
	case ext == ".rb" && (strings.HasSuffix(base, "_spec.rb") || strings.HasSuffix(base, "_test.rb")):
		return true
	case ext == ".exs" && strings.HasSuffix(base, "_test.exs"):
		return true
	case ext == ".lua" && (strings.HasSuffix(base, "_test.lua") || strings.HasSuffix(base, "_spec.lua")):
		return true
	case (ext == ".c" || ext == ".cpp" || ext == ".cc") && isTestSuffixName(origBase):
		return true
	default:
		return false
	}
}

// isTestSuffixName は *Test.ext / *Tests.ext / *Spec.ext のパターンを判定する。
// origBase は大文字小文字を保持した元のファイル名（PascalCase 判定に必要）。
func isTestSuffixName(origBase string) bool {
	ext := filepath.Ext(origBase)
	nameNoExt := strings.TrimSuffix(origBase, ext)
	// PascalCase: UserServiceTest, UserServiceTests, UserServiceSpec
	if strings.HasSuffix(nameNoExt, "Test") || strings.HasSuffix(nameNoExt, "Tests") || strings.HasSuffix(nameNoExt, "Spec") {
		return true
	}
	// snake_case: user_service_test, user_service_spec
	lower := strings.ToLower(nameNoExt)
	return strings.HasSuffix(lower, "_test") || strings.HasSuffix(lower, "_tests") || strings.HasSuffix(lower, "_spec")
}

// isInTestsDir はファイルパスが tests/ または test/ ディレクトリ配下かどうか判定する。
func isInTestsDir(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.HasPrefix(normalized, "tests/") ||
		strings.HasPrefix(normalized, "test/") ||
		strings.Contains(normalized, "/tests/") ||
		strings.Contains(normalized, "/test/")
}

// ExtractSignatureMetadata は行テキストからシンボル名と種別を抽出する（全言語）。
// Project Map 生成用。
func ExtractSignatureMetadata(sig string) (string, string, bool) {
	return extractSignatureMetadata(sig)
}

// ExtractSignatureMetadataForLang は指定言語に限定してシンボル名と種別を抽出する。
// search_code の多言語シンボル解決用。lang が空の場合は全言語パターンを適用する。
func ExtractSignatureMetadataForLang(sig, lang string) (string, string, bool) {
	return extractSignatureMetadataForLang(sig, lang)
}

// IsTestFile はテストファイルかどうかを返す。
func IsTestFile(path string) bool {
	return isTestFile(path)
}

func testSortBase(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "_test.go"):
		return strings.TrimSuffix(lower, "_test.go") + ".go"
	case strings.HasPrefix(lower, "test_") && strings.HasSuffix(lower, ".py"):
		return strings.TrimPrefix(lower, "test_")
	case strings.HasSuffix(lower, "_test.py"):
		return strings.TrimSuffix(lower, "_test.py") + ".py"
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
		ext := filepath.Ext(name)
		switch strings.ToLower(ext) {
		case ".java", ".kt", ".cs", ".php", ".swift", ".scala", ".c", ".cpp", ".cc":
			if base, ok := stripTestSuffixName(strings.TrimSuffix(name, ext)); ok {
				return strings.ToLower(base) + strings.ToLower(ext)
			}
		}
		return lower
	}
}

func stripTestSuffixName(name string) (string, bool) {
	switch {
	case strings.HasSuffix(name, "Tests"):
		return strings.TrimSuffix(name, "Tests"), true
	case strings.HasSuffix(name, "Test"):
		return strings.TrimSuffix(name, "Test"), true
	case strings.HasSuffix(name, "Spec"):
		return strings.TrimSuffix(name, "Spec"), true
	}

	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "_tests"):
		return name[:len(name)-len("_tests")], true
	case strings.HasSuffix(lower, "_test"):
		return name[:len(name)-len("_test")], true
	case strings.HasSuffix(lower, "_spec"):
		return name[:len(name)-len("_spec")], true
	default:
		return "", false
	}
}
