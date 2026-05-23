package jsast

import (
	"regexp"
	"strings"
)

var fallbackTypeBodyMethodPattern = regexp.MustCompile(`^(?:(?:public|private|protected|static|async|abstract|override|readonly|get|set)\s+)*(` + fallbackIdentifierPattern + `)\s*(?:<[^>{}]*>)?\s*\(`)

func fallbackMethodSymbolFromLine(parsed *ParsedFile, line string, trimmed string, lineNo int, lineStart int) (Symbol, bool) {
	if strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "interface ") {
		return Symbol{}, false
	}
	match := fallbackTypeBodyMethodPattern.FindStringSubmatch(trimmed)
	if len(match) == 0 {
		return Symbol{}, false
	}
	name := strings.TrimSpace(match[1])
	if name == "" || fallbackMethodNameIsKeyword(name) {
		return Symbol{}, false
	}
	return fallbackSymbol(parsed, line, trimmed, lineNo, lineStart, name, "method", false), true
}

func fallbackMethodNameIsKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "catch", "function", "return":
		return true
	default:
		return false
	}
}

type fallbackTypeBodyScope struct {
	depth int
	kind  string
}

func fallbackOpeningTypeBodyScope(trimmed string, depth int) (fallbackTypeBodyScope, bool) {
	if !strings.Contains(trimmed, "{") {
		return fallbackTypeBodyScope{}, false
	}
	switch {
	case fallbackClassDeclarationPattern.MatchString(trimmed):
		return fallbackTypeBodyScope{depth: depth, kind: "class"}, true
	case fallbackInterfaceDeclarationPattern.MatchString(trimmed):
		return fallbackTypeBodyScope{depth: depth, kind: "interface"}, true
	default:
		return fallbackTypeBodyScope{}, false
	}
}

func fallbackDirectTypeBodyScope(braceDepth int, scopes []fallbackTypeBodyScope) (fallbackTypeBodyScope, bool) {
	if len(scopes) == 0 {
		return fallbackTypeBodyScope{}, false
	}
	scope := scopes[len(scopes)-1]
	return scope, scope.depth == braceDepth
}

func fallbackOpenTypeBodyScopes(scopes []fallbackTypeBodyScope, braceDepth int) []fallbackTypeBodyScope {
	for len(scopes) > 0 && braceDepth < scopes[len(scopes)-1].depth {
		scopes = scopes[:len(scopes)-1]
	}
	return scopes
}
