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
	if parsed == nil || parsed.tree == nil {
		return nil
	}
	importedName = strings.TrimSpace(importedName)
	if importedName == "" {
		return nil
	}

	var aliases []NamedImportAlias
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		if nodeKind(parsed, node) != "import_specifier" || importSpecifierIsTypeOnly(parsed, node) {
			return
		}
		importedNode := childByField(parsed, node, "name")
		localNode := childByField(parsed, node, "alias")
		if importedNode == nil || localNode == nil {
			return
		}
		imported := strings.TrimSpace(nodeText(parsed, importedNode))
		local := strings.TrimSpace(nodeText(parsed, localNode))
		if imported != importedName || local == "" || local == imported {
			return
		}
		aliases = append(aliases, NamedImportAlias{
			Imported:       imported,
			Local:          local,
			Source:         importSpecifierSource(parsed, node),
			Line:           int(node.StartPoint().Row) + 1,
			localStartByte: localNode.StartByte(),
			localEndByte:   localNode.EndByte(),
		})
	})
	return aliases
}

func importSpecifierIsTypeOnly(parsed *ParsedFile, node *gotreesitter.Node) bool {
	if hasDirectAnonymousChildKind(parsed, node, "type", "typeof") {
		return true
	}
	for current := node; current != nil; current = current.Parent() {
		if nodeKind(parsed, current) != "import_statement" {
			continue
		}
		return hasDirectAnonymousChildKind(parsed, current, "type", "typeof")
	}
	return false
}

func importSpecifierSource(parsed *ParsedFile, node *gotreesitter.Node) string {
	for current := node; current != nil; current = current.Parent() {
		if nodeKind(parsed, current) != "import_statement" {
			continue
		}
		return unquoteImportSource(nodeText(parsed, childByField(parsed, current, "source")))
	}
	return ""
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
