package jsast

import (
	"regexp"
	"strings"
)

var fallbackTypeBodyPropertyPattern = regexp.MustCompile(`^(?:(?:public|private|protected|static|abstract|override|readonly|accessor|declare)\s+)*(` + fallbackIdentifierPattern + `)\s*[?!]?\s*(?::|=)`)

func fallbackPropertySymbolFromLine(parsed *ParsedFile, line string, trimmed string, lineNo int, lineStart int, scopeKind string) (Symbol, bool) {
	if strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "interface ") {
		return Symbol{}, false
	}
	match := fallbackTypeBodyPropertyPattern.FindStringSubmatch(trimmed)
	if len(match) == 0 {
		return Symbol{}, false
	}
	name := strings.TrimSpace(match[1])
	if name == "" || fallbackMethodNameIsKeyword(name) {
		return Symbol{}, false
	}
	kind := "property"
	if scopeKind == "class" {
		kind = "field"
	}
	return fallbackSymbol(parsed, line, trimmed, lineNo, lineStart, name, kind, false), true
}
