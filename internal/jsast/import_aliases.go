package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

type NamedImportAlias struct {
	Imported string
	Local    string
	Source   string
	Line     int

	localStartByte uint32
	localEndByte   uint32
}

func NamedImportAliasesWithParsed(parsed *ParsedFile, importedName string) []NamedImportAlias {
	importedName = strings.TrimSpace(importedName)
	if importedName == "" {
		return nil
	}

	var aliases []NamedImportAlias
	for _, binding := range ImportBindingsWithParsed(parsed) {
		if binding.Kind != ImportBindingNamed || binding.Imported != importedName || binding.Local == binding.Imported {
			continue
		}
		aliases = append(aliases, NamedImportAlias{
			Imported:       binding.Imported,
			Local:          binding.Local,
			Source:         binding.Source,
			Line:           binding.Line,
			localStartByte: binding.localStartByte,
			localEndByte:   binding.localEndByte,
		})
	}
	return aliases
}

func importSpecifierIsTypeOnly(parsed *ParsedFile, node *gotreesitter.Node) bool {
	if importSpecifierTextIsTypeOnly(nodeText(parsed, node)) {
		return true
	}
	for current := node; current != nil; current = current.Parent() {
		if nodeKind(parsed, current) != "import_statement" {
			continue
		}
		return importStatementIsTypeOnly(parsed, current)
	}
	return false
}

func importStatementTextIsTypeOnly(text string) bool {
	i := skipImportBindingWhitespaceAndComments(text, 0)
	if !hasImportBindingKeywordAt(text, i, "import") {
		return false
	}
	i = skipImportBindingWhitespaceAndComments(text, i+len("import"))
	return hasImportBindingKeywordAt(text, i, "type") || hasImportBindingKeywordAt(text, i, "typeof")
}

func importSpecifierTextIsTypeOnly(text string) bool {
	i := skipImportBindingWhitespaceAndComments(text, 0)
	keyword := ""
	switch {
	case hasImportBindingKeywordAt(text, i, "type"):
		keyword = "type"
	case hasImportBindingKeywordAt(text, i, "typeof"):
		keyword = "typeof"
	default:
		return false
	}
	i = skipImportBindingWhitespaceAndComments(text, i+len(keyword))
	if hasImportBindingKeywordAt(text, i, "as") || i >= len(text) {
		return false
	}
	return isJSIdentifierStart(text[i])
}

func skipImportBindingWhitespaceAndComments(text string, i int) int {
	for i < len(text) {
		switch {
		case isJSWhitespace(text[i]):
			i++
		case i+1 < len(text) && text[i] == '/' && text[i+1] == '*':
			i += 2
			for i+1 < len(text) && (text[i] != '*' || text[i+1] != '/') {
				i++
			}
			if i+1 < len(text) {
				i += 2
			}
		case i+1 < len(text) && text[i] == '/' && text[i+1] == '/':
			i += 2
			for i < len(text) && text[i] != '\n' && text[i] != '\r' {
				i++
			}
		default:
			return i
		}
	}
	return i
}

func hasImportBindingKeywordAt(text string, i int, keyword string) bool {
	if i < 0 || i+len(keyword) > len(text) || text[i:i+len(keyword)] != keyword {
		return false
	}
	beforeOK := i == 0 || !isJSIdentifierPart(text[i-1])
	after := i + len(keyword)
	afterOK := after >= len(text) || !isJSIdentifierPart(text[after])
	return beforeOK && afterOK
}

func unquoteImportSource(text string) string {
	text = strings.TrimSpace(text)
	if len(text) < 2 {
		return text
	}
	first := text[0]
	last := text[len(text)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') || (first == '`' && last == '`') {
		return text[1 : len(text)-1]
	}
	return text
}
