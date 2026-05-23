package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

// ImportBindingKind は import binding の種類を表す。
type ImportBindingKind string

const (
	// ImportBindingNamed は named import binding を表す。
	ImportBindingNamed ImportBindingKind = "named"
	// ImportBindingDefault は default import binding を表す。
	ImportBindingDefault ImportBindingKind = "default"
	// ImportBindingType は type-only import binding を表す。
	ImportBindingType ImportBindingKind = "type"
)

// ImportBinding は import 文が作る local binding を表す。
type ImportBinding struct {
	Kind     ImportBindingKind
	Imported string
	Local    string
	Source   string
	Line     int

	localStartByte     uint32
	localEndByte       uint32
	statementStartLine int
	statementEndLine   int
}

// ImportBindingCoversLine は binding を作った import/require 文が指定行を含むかを返す。
func ImportBindingCoversLine(binding ImportBinding, line int) bool {
	if line <= 0 {
		return false
	}
	if binding.statementStartLine <= 0 || binding.statementEndLine <= 0 {
		return binding.Line == line
	}
	return binding.statementStartLine <= line && line <= binding.statementEndLine
}

// ImportBindingsWithParsed は value import の local binding を抽出する。
func ImportBindingsWithParsed(parsed *ParsedFile) []ImportBinding {
	if parsed == nil || parsed.tree == nil {
		return nil
	}
	var bindings []ImportBinding
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		if nodeKind(parsed, node) != "import_statement" {
			return
		}
		source := importStatementSource(parsed, node)
		if source == "" {
			return
		}
		statementTypeOnly := importStatementIsTypeOnly(parsed, node)
		if statementTypeOnly {
			return
		}
		if binding, ok := defaultImportBinding(parsed, node, source, ImportBindingDefault); ok {
			bindings = append(bindings, binding)
		}
		bindings = append(bindings, namedImportBindings(parsed, node, source, ImportBindingNamed, valueImportSpecifier)...)
	})
	return bindings
}

func defaultImportBinding(parsed *ParsedFile, node *gotreesitter.Node, source string, kind ImportBindingKind) (ImportBinding, bool) {
	name := defaultImportNameNode(parsed, node)
	if name == nil {
		return ImportBinding{}, false
	}
	local := strings.TrimSpace(nodeText(parsed, name))
	if local == "" {
		return ImportBinding{}, false
	}
	return (ImportBinding{
		Kind:           kind,
		Imported:       "default",
		Local:          local,
		Source:         source,
		Line:           int(name.StartPoint().Row) + 1,
		localStartByte: name.StartByte(),
		localEndByte:   name.EndByte(),
	}).withStatementNode(node), true
}

func defaultImportNameNode(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	for i := 0; node != nil && i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch nodeKind(parsed, child) {
		case "import_clause":
			return defaultImportNameNode(parsed, child)
		case "identifier":
			return child
		case "named_imports", "namespace_import", "import_specifier", "string":
			return nil
		}
	}
	return nil
}

func namedImportBindings(parsed *ParsedFile, node *gotreesitter.Node, source string, kind ImportBindingKind, include func(*ParsedFile, *gotreesitter.Node) bool) []ImportBinding {
	var bindings []ImportBinding
	walkNamed(node, func(current *gotreesitter.Node) {
		if nodeKind(parsed, current) != "import_specifier" || !include(parsed, current) {
			return
		}
		importedNode := childByField(parsed, current, "name")
		if importedNode == nil {
			return
		}
		localNode := childByField(parsed, current, "alias")
		if localNode == nil {
			localNode = importedNode
		}
		imported := strings.TrimSpace(nodeText(parsed, importedNode))
		local := strings.TrimSpace(nodeText(parsed, localNode))
		if imported == "" || local == "" {
			return
		}
		bindings = append(bindings, (ImportBinding{
			Kind:           kind,
			Imported:       imported,
			Local:          local,
			Source:         source,
			Line:           int(current.StartPoint().Row) + 1,
			localStartByte: localNode.StartByte(),
			localEndByte:   localNode.EndByte(),
		}).withStatementNode(node))
	})
	return bindings
}

func valueImportSpecifier(parsed *ParsedFile, node *gotreesitter.Node) bool {
	return !importSpecifierIsTypeOnly(parsed, node)
}

func importStatementIsTypeOnly(parsed *ParsedFile, node *gotreesitter.Node) bool {
	return importStatementTextIsTypeOnly(nodeText(parsed, node))
}

func importStatementSource(parsed *ParsedFile, node *gotreesitter.Node) string {
	return unquoteImportSource(nodeText(parsed, childByField(parsed, node, "source")))
}

func (binding ImportBinding) withStatementNode(node *gotreesitter.Node) ImportBinding {
	if node == nil {
		return binding
	}
	binding.statementStartLine = int(node.StartPoint().Row) + 1
	binding.statementEndLine = int(node.EndPoint().Row) + 1
	if binding.statementEndLine < binding.statementStartLine {
		binding.statementEndLine = binding.statementStartLine
	}
	return binding
}

func (binding ImportBinding) withStatementLines(startLine int, endLine int) ImportBinding {
	if startLine <= 0 || endLine <= 0 {
		return binding
	}
	if endLine < startLine {
		endLine = startLine
	}
	binding.statementStartLine = startLine
	binding.statementEndLine = endLine
	return binding
}
