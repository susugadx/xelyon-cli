package tui

import "strings"

// looksLikePastedPathCandidate は paste 文字列を path 解釈するかどうかの policy owner。
func looksLikePastedPathCandidate(token string) bool {
	trimmed := trimPathQuotes(strings.TrimSpace(token))
	if trimmed == "" {
		return false
	}

	lower := strings.ToLower(trimmed)
	if hasNonFileURIScheme(trimmed) {
		return false
	}
	switch {
	case strings.HasPrefix(lower, "file://"):
		return true
	case strings.HasPrefix(trimmed, "~/"):
		return true
	case strings.HasPrefix(trimmed, "./"), strings.HasPrefix(trimmed, "../"):
		return true
	case strings.HasPrefix(trimmed, "/"):
		return true
	case strings.HasPrefix(trimmed, `\\`):
		return true
	case looksLikeWindowsPath(trimmed):
		return true
	case looksLikeBareRelativeFilename(trimmed):
		return true
	default:
		return false
	}
}

func looksLikeBareRelativeFilename(token string) bool {
	if token == "." || token == ".." {
		return false
	}
	if strings.ContainsAny(token, `/\`) {
		return false
	}
	return strings.Contains(token, ".")
}

func hasNonFileURIScheme(raw string) bool {
	sep := strings.Index(raw, "://")
	if sep <= 0 {
		return false
	}
	return !strings.EqualFold(raw[:sep], "file")
}
