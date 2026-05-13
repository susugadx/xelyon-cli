package jsast

import (
	"regexp"
	"strings"
)

var (
	fallbackIdentifierPattern           = `[A-Za-z_$][A-Za-z0-9_$]*`
	fallbackFunctionDeclarationPattern  = regexp.MustCompile(`^(?:export\s+default\s+|export\s+)?(?:async\s+)?function\s+(` + fallbackIdentifierPattern + `)\b`)
	fallbackClassDeclarationPattern     = regexp.MustCompile(`^(?:export\s+default\s+|export\s+)?class\s+(` + fallbackIdentifierPattern + `)\b`)
	fallbackInterfaceDeclarationPattern = regexp.MustCompile(`^(?:export\s+)?interface\s+(` + fallbackIdentifierPattern + `)\b`)
	fallbackTypeDeclarationPattern      = regexp.MustCompile(`^(?:export\s+)?type\s+(` + fallbackIdentifierPattern + `)\b`)
	fallbackEnumDeclarationPattern      = regexp.MustCompile(`^(?:export\s+)?enum\s+(` + fallbackIdentifierPattern + `)\b`)
	fallbackVariableTypeAnnotation      = `(?:\s*:\s*[^=]+)?`
	fallbackFunctionVariablePattern     = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(` + fallbackIdentifierPattern + `)\b` + fallbackVariableTypeAnnotation + `\s*=\s*(?:async\s+)?function\b`)
	fallbackArrowVariablePattern        = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(` + fallbackIdentifierPattern + `)\b` + fallbackVariableTypeAnnotation + `\s*=\s*(?:async\s+)?(?:\([^)]*\)|` + fallbackIdentifierPattern + `)\s*=>`)
	fallbackClassVariablePattern        = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(` + fallbackIdentifierPattern + `)\b` + fallbackVariableTypeAnnotation + `\s*=\s*class\b`)
	fallbackModuleExportFunctionPattern = regexp.MustCompile(`^module\s*\.\s*exports\s*=\s*(?:async\s+)?function\s+(` + fallbackIdentifierPattern + `)\b`)
	fallbackModuleExportClassPattern    = regexp.MustCompile(`^module\s*\.\s*exports\s*=\s*class\s+(` + fallbackIdentifierPattern + `)\b`)
	fallbackNamedExportFunctionPattern  = regexp.MustCompile(`^(?:exports|module\s*\.\s*exports)\s*(?:\.\s*(` + fallbackIdentifierPattern + `)|\[\s*['"]([^'"]+)['"]\s*\])\s*=\s*(?:async\s+)?function(?:\s+(` + fallbackIdentifierPattern + `)\b|\s*\()`)
	fallbackNamedExportClassPattern     = regexp.MustCompile(`^(?:exports|module\s*\.\s*exports)\s*(?:\.\s*(` + fallbackIdentifierPattern + `)|\[\s*['"]([^'"]+)['"]\s*\])\s*=\s*class(?:\s+(` + fallbackIdentifierPattern + `)\b|\s*\{)`)
)

type fallbackSymbolPattern struct {
	pattern  *regexp.Regexp
	kind     string
	exported bool
}

var fallbackSymbolPatterns = []fallbackSymbolPattern{
	{fallbackFunctionDeclarationPattern, "function", false},
	{fallbackClassDeclarationPattern, "class", false},
	{fallbackInterfaceDeclarationPattern, "interface", false},
	{fallbackTypeDeclarationPattern, "type", false},
	{fallbackEnumDeclarationPattern, "enum", false},
	{fallbackFunctionVariablePattern, "function", false},
	{fallbackArrowVariablePattern, "function", false},
	{fallbackClassVariablePattern, "class", false},
	{fallbackModuleExportFunctionPattern, "function", true},
	{fallbackModuleExportClassPattern, "class", true},
	{fallbackNamedExportFunctionPattern, "function", true},
	{fallbackNamedExportClassPattern, "class", true},
}

func fallbackSymbolsFromSource(parsed *ParsedFile) []Symbol {
	if parsed == nil || len(parsed.src) == 0 {
		return nil
	}
	lines := strings.Split(string(parsed.src), "\n")
	symbols := make([]Symbol, 0)
	byteOffset := 0
	for idx, line := range lines {
		lineNo := idx + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			byteOffset += len(line) + 1
			continue
		}
		if symbol, ok := fallbackSymbolFromLine(parsed, line, trimmed, lineNo, byteOffset); ok {
			symbols = append(symbols, symbol)
		}
		byteOffset += len(line) + 1
	}
	return symbols
}

func fallbackSymbolFromLine(parsed *ParsedFile, line string, trimmed string, lineNo int, lineStart int) (Symbol, bool) {
	exportedDeclaration := strings.HasPrefix(trimmed, "export ")
	for _, candidate := range fallbackSymbolPatterns {
		if match := candidate.pattern.FindStringSubmatch(trimmed); len(match) > 0 {
			if name := firstNonEmptySubmatch(match[1:]); name != "" {
				return fallbackSymbol(parsed, line, trimmed, lineNo, lineStart, name, candidate.kind, candidate.exported || exportedDeclaration), true
			}
		}
	}
	return Symbol{}, false
}

func fallbackSymbol(parsed *ParsedFile, line string, trimmed string, lineNo int, lineStart int, name string, kind string, exported bool) Symbol {
	nameColumn := strings.Index(line, name)
	character := 1
	if nameColumn >= 0 {
		character = lspCharacterForByteOffset(parsed.src, uint32(lineStart+nameColumn))
	}
	return Symbol{
		Name:      name,
		Kind:      kind,
		Signature: trimmed,
		Line:      lineNo,
		EndLine:   lineNo,
		Character: character,
		Exported:  exported,
	}
}

func firstNonEmptySubmatch(matches []string) string {
	for _, match := range matches {
		if strings.TrimSpace(match) != "" {
			return strings.TrimSpace(match)
		}
	}
	return ""
}
