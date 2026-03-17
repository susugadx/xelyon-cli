package pathmatch

import (
	"path"
	"regexp"
	"strings"
)

var defaultIgnorePatterns = []string{
	".git",
	"node_modules",
	"vendor",
	".next",
	"__pycache__",
	".venv",
	"dist",
	"build",
	".idea",
	".vscode",
}

// Matcher は相対パスに対する glob/segment ベースのマッチャーを表す。
type Matcher struct {
	patterns []compiledPattern
}

type compiledPattern struct {
	raw   string
	regex *regexp.Regexp
}

// DefaultIgnorePatterns は共通のデフォルト ignore パターンを返す。
func DefaultIgnorePatterns() []string {
	patterns := make([]string, len(defaultIgnorePatterns))
	copy(patterns, defaultIgnorePatterns)
	return patterns
}

// NormalizePatterns はパターンを正規化し、空要素と重複を除去する。
func NormalizePatterns(patterns []string) []string {
	seen := make(map[string]struct{}, len(patterns))
	normalized := make([]string, 0, len(patterns))

	for _, pattern := range patterns {
		pattern = normalizePattern(pattern)
		if pattern == "" {
			continue
		}
		if _, ok := seen[pattern]; ok {
			continue
		}
		seen[pattern] = struct{}{}
		normalized = append(normalized, pattern)
	}

	return normalized
}

// NewMatcher は glob/segment パターンからマッチャーを構築する。
func NewMatcher(patterns []string) *Matcher {
	normalized := NormalizePatterns(patterns)
	compiled := make([]compiledPattern, 0, len(normalized))
	for _, pattern := range normalized {
		if regex := compilePattern(pattern); regex != nil {
			compiled = append(compiled, compiledPattern{
				raw:   pattern,
				regex: regex,
			})
		}
	}
	return &Matcher{patterns: compiled}
}

// Match は相対パスがいずれかのパターンに一致するかを返す。
func (m *Matcher) Match(relPath string, isDir bool) bool {
	if m == nil || len(m.patterns) == 0 {
		return false
	}

	target := normalizeTarget(relPath, isDir)
	if target == "" {
		return false
	}

	for _, pattern := range m.patterns {
		if pattern.regex.MatchString(target) {
			return true
		}
	}
	return false
}

// BuildRGIgnoreGlobs は ripgrep に渡す ignore glob の一覧を返す。
func BuildRGIgnoreGlobs(patterns []string) []string {
	normalized := NormalizePatterns(patterns)
	globs := make([]string, 0, len(normalized)*4)

	for _, pattern := range normalized {
		switch {
		case !HasWildcard(pattern) && !strings.Contains(pattern, "/"):
			globs = append(globs,
				"!"+pattern,
				"!"+pattern+"/**",
				"!**/"+pattern,
				"!**/"+pattern+"/**",
			)
		case !HasWildcard(pattern):
			globs = append(globs,
				"!"+pattern,
				"!"+pattern+"/**",
			)
		case strings.Contains(pattern, "/"):
			globs = append(globs, "!"+pattern)
		default:
			globs = append(globs,
				"!"+pattern,
				"!**/"+pattern,
			)
		}
	}

	return NormalizePatterns(globs)
}

// HasWildcard は `*`, `?`, `[` を含む glob パターンかどうかを返す。
func HasWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func normalizePattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	pattern = strings.TrimPrefix(pattern, "./")
	pattern = path.Clean(strings.ReplaceAll(pattern, "\\", "/"))
	if pattern == "." {
		return ""
	}
	pattern = strings.TrimSuffix(pattern, "/.")
	return strings.TrimPrefix(pattern, "/")
}

func normalizeTarget(relPath string, isDir bool) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return ""
	}
	relPath = strings.TrimPrefix(relPath, "./")
	relPath = strings.TrimPrefix(relPath, "/")
	relPath = path.Clean(strings.ReplaceAll(relPath, "\\", "/"))
	if relPath == "." {
		relPath = ""
	}
	if relPath == "" {
		return ""
	}
	if isDir && !strings.HasSuffix(relPath, "/") {
		relPath += "/"
	}
	return relPath
}

func compilePattern(pattern string) *regexp.Regexp {
	switch {
	case !HasWildcard(pattern) && !strings.Contains(pattern, "/"):
		return regexp.MustCompile(`(^|/)` + regexp.QuoteMeta(pattern) + `(/|$)`)
	case !HasWildcard(pattern):
		return regexp.MustCompile(`^` + regexp.QuoteMeta(strings.TrimSuffix(pattern, "/")) + `(/.*)?$`)
	case !strings.Contains(pattern, "/"):
		return regexp.MustCompile(`(^|/)` + globToRegex(pattern) + `$`)
	default:
		return regexp.MustCompile(`^` + globToRegex(pattern) + `$`)
	}
}

func globToRegex(pattern string) string {
	var b strings.Builder

	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
				continue
			}
			b.WriteString(`[^/]*`)
		case '?':
			b.WriteString(`[^/]`)
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteByte(ch)
		case '[':
			end := strings.IndexByte(pattern[i+1:], ']')
			if end < 0 {
				b.WriteString(`\[`)
				continue
			}
			b.WriteByte('[')
			b.WriteString(pattern[i+1 : i+1+end])
			b.WriteByte(']')
			i += end + 1
		default:
			b.WriteByte(ch)
		}
	}

	return b.String()
}
