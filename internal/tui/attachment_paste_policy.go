package tui

import "strings"

type pathCandidateContext struct {
	allowSingleBareRelative bool
}

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

func resolvePathCandidateToken(token string, ctx pathCandidateContext) (string, bool) {
	normalizedToken := trimPathQuotes(strings.TrimSpace(token))
	if normalizedToken == "" {
		return "", false
	}
	if looksLikePastedPathCandidate(normalizedToken) {
		normalized := normalizePastedPathToken(normalizedToken)
		if !normalized.isOK() {
			return "", false
		}
		return normalized.path, true
	}
	if !ctx.allowSingleBareRelative || !looksLikeSingleBareRelativePathCandidate(normalizedToken) {
		return "", false
	}
	normalized := normalizePastedPathToken(normalizedToken)
	if !normalized.isOK() || !isAttachablePath(normalized.path) {
		return "", false
	}
	return normalized.path, true
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

func looksLikeSingleBareRelativePathCandidate(token string) bool {
	if token == "." || token == ".." {
		return false
	}
	if strings.ContainsAny(token, `/\`) || strings.Contains(token, ".") {
		return false
	}
	if strings.Contains(token, "://") {
		return false
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return false
	}
	return true
}

func hasNonFileURIScheme(raw string) bool {
	sep := strings.Index(raw, "://")
	if sep <= 0 {
		return false
	}
	return !strings.EqualFold(raw[:sep], "file")
}
