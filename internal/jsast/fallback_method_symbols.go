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

func fallbackOpensTypeBody(trimmed string) bool {
	if !strings.Contains(trimmed, "{") {
		return false
	}
	return fallbackClassDeclarationPattern.MatchString(trimmed) ||
		fallbackInterfaceDeclarationPattern.MatchString(trimmed)
}

func fallbackInDirectTypeBody(braceDepth int, typeBodyDepths []int) bool {
	return len(typeBodyDepths) > 0 && typeBodyDepths[len(typeBodyDepths)-1] == braceDepth
}

func fallbackOpenTypeBodyDepths(depths []int, braceDepth int) []int {
	for len(depths) > 0 && braceDepth < depths[len(depths)-1] {
		depths = depths[:len(depths)-1]
	}
	return depths
}
