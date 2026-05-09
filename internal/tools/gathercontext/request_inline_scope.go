package gathercontext

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filequery"
)

type inlineSearchScope struct {
	query string
	path  string
}

type inlineSearchScopePathSyntax string

const (
	inlineScopePathNone     inlineSearchScopePathSyntax = "none"
	inlineScopePathExplicit inlineSearchScopePathSyntax = "explicit"
	inlineScopePathSlash    inlineSearchScopePathSyntax = "slash"
	inlineScopePathBareFile inlineSearchScopePathSyntax = "bare_file"
)

func parseTrailingInlineSearchScope(query string) (inlineSearchScope, bool) {
	query = strings.TrimSpace(query)
	tokens := topLevelQueryTokens(query)
	if len(tokens) < 3 {
		return inlineSearchScope{}, false
	}

	keyword := tokens[len(tokens)-2]
	if !isInlineSearchScopeKeyword(keyword.text) {
		return inlineSearchScope{}, false
	}

	prefix := strings.TrimSpace(query[:keyword.start])
	if !inlineSearchScopePrefixHasSearchIntent(prefix, keyword.text) {
		return inlineSearchScope{}, false
	}

	scope, ok := normalizeInlineSearchScopePath(tokens[len(tokens)-1].text)
	if !ok {
		return inlineSearchScope{}, false
	}
	if prefix == "" || scope == "" {
		return inlineSearchScope{}, false
	}
	return inlineSearchScope{query: prefix, path: scope}, true
}

func isInlineSearchScopeKeyword(token string) bool {
	switch strings.ToLower(strings.Trim(token, `"'()[]{}:;,.`)) {
	case "in", "under":
		return true
	default:
		return false
	}
}

func normalizeInlineSearchScopePath(rawScope string) (string, bool) {
	scope := trimInlineSearchScopePathToken(rawScope)
	if scope == "" {
		return "", false
	}
	if classifyInlineSearchScopePathSyntax(scope) == inlineScopePathNone {
		return "", false
	}
	return strings.TrimRight(scope, `/\`), true
}

func trimInlineSearchScopePathToken(scope string) string {
	return trimQueryBoundaryQuotes(scope)
}

func classifyInlineSearchScopePathSyntax(scope string) inlineSearchScopePathSyntax {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return inlineScopePathNone
	}
	if filequery.HasExplicitDirectoryMarker(scope) ||
		filequery.HasExplicitRelativePrefix(scope) ||
		filequery.HasWindowsPathPrefix(scope) ||
		filepath.IsAbs(scope) {
		return inlineScopePathExplicit
	}
	if strings.ContainsAny(scope, `/\`) {
		return inlineScopePathSlash
	}
	if filequery.LooksLikeBareExtFileCandidate(scope) ||
		filequery.LooksLikeBareNamedFileCandidate(scope) {
		return inlineScopePathBareFile
	}
	return inlineScopePathNone
}

func inlineSearchScopePrefixHasSearchIntent(prefix, keyword string) bool {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return false
	}
	switch strings.ToLower(strings.Trim(keyword, `"'()[]{}:;,.`)) {
	case "in", "under":
		return filequery.ContainsNaturalLanguageSearchIntentMarker(prefix)
	default:
		return false
	}
}
