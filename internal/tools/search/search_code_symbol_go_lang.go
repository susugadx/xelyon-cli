package search

import (
	"os"
	"path/filepath"
	"strings"
)

// looksLikeIdentifier は文字列がシンボル名に見えるか判定する。
// 例: "NewAgent", "Config.Build", "(*Config).Build", "authenticate", "UserService"
func looksLikeIdentifier(s string) bool {
	if s == "" {
		return false
	}

	if containsRegexMeta(s) {
		return false
	}
	if strings.ContainsAny(s, " \t\n") {
		return false
	}
	if strings.Contains(s, ".*") || strings.Contains(s, ".+") {
		return false
	}
	if strings.Contains(s, "(") && !strings.HasPrefix(s, "(*") {
		return false
	}

	for _, r := range s {
		if !isIdentRune(r) && r != '.' && r != '*' && r != '(' && r != ')' {
			return false
		}
	}

	return true
}

func isIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// containsRegexMeta は regex メタ文字を含むか判定する。
// '.', '*', '(', ')' はシンボルで使われるため除外する。
func containsRegexMeta(s string) bool {
	return strings.ContainsAny(s, "+?[]{}|\\^$")
}

// isSymbolResolvableLanguage は指定言語でシンボル解決が可能か返す。
func isSymbolResolvableLanguage(fileType string) bool {
	switch fileType {
	case "go":
		return true
	case "py", "python":
		return true
	case "ts", "tsx", "js", "jsx", "mjs", "javascript":
		return true
	case "rs", "rust":
		return true
	case "java", "kt", "kts", "kotlin":
		return true
	case "cs", "csharp":
		return true
	case "rb", "ruby":
		return true
	case "php":
		return true
	case "c", "cpp", "cc", "h", "hpp":
		return true
	case "swift":
		return true
	case "scala":
		return true
	case "ex", "exs", "elixir":
		return true
	case "lua":
		return true
	case "sh", "bash", "zsh":
		return true
	default:
		return false
	}
}

// resolveLanguage は SearchOptions から正規化された言語名を返す。
func resolveLanguage(opts SearchOptions) string {
	if opts.FileType != "" {
		switch opts.FileType {
		case "go":
			return "go"
		case "py", "python":
			return "python"
		case "ts", "tsx", "js", "jsx", "mjs", "typescript", "javascript":
			return "js"
		case "rs", "rust":
			return "rust"
		case "java", "kt", "kts", "kotlin":
			return "java"
		case "cs", "csharp":
			return "csharp"
		case "php":
			return "php"
		case "rb", "ruby":
			return "ruby"
		case "c", "cpp", "cc", "h", "hpp":
			return "cpp"
		case "swift":
			return "swift"
		case "scala":
			return "scala"
		case "ex", "exs", "elixir":
			return "elixir"
		case "lua":
			return "lua"
		default:
			return ""
		}
	}

	if lang := resolveLanguageFromPath(opts.Path); lang != "" {
		return lang
	}

	switch opts.FilePattern {
	case "*.go":
		return "go"
	}

	return ""
}

func resolveLanguageFromPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	if ext := strings.TrimPrefix(filepath.Ext(trimmed), "."); ext != "" {
		switch ext {
		case "go":
			return "go"
		case "py":
			return "python"
		case "ts", "tsx", "js", "jsx":
			return "js"
		case "rs":
			return "rust"
		case "java", "kt":
			return "java"
		case "cs":
			return "csharp"
		case "php":
			return "php"
		case "rb":
			return "ruby"
		case "swift":
			return "swift"
		}
	}

	dir := trimmed
	if info, err := os.Stat(trimmed); err == nil && !info.IsDir() {
		dir = filepath.Dir(trimmed)
	}
	if dir == "" {
		dir = "."
	}
	if hasGoModuleMarkerUpward(dir) {
		return "go"
	}
	if hasGoSourceFileDirectly(dir) {
		return "go"
	}

	return ""
}

func hasGoModuleMarkerUpward(dir string) bool {
	start := dir
	if start == "" {
		start = "."
	}
	for {
		for _, marker := range []string{"go.mod", "go.work"} {
			if _, err := os.Stat(filepath.Join(start, marker)); err == nil {
				return true
			}
		}
		parent := filepath.Dir(start)
		if parent == start {
			return false
		}
		start = parent
	}
}

func hasGoSourceFileDirectly(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".go") {
			return true
		}
	}
	return false
}
