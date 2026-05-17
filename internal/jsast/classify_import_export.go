package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

func classifyImportOrExportReference(parsed *ParsedFile, node *gotreesitter.Node, startByte uint32, endByte uint32) (codeast.MatchClass, bool) {
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "import_statement", "import_clause", "import_specifier", "namespace_import":
			return codeast.ClassImport, true
		case "export_specifier":
			if fieldContainsRange(parsed, current, "alias", startByte, endByte) && !fieldContainsRange(parsed, current, "name", startByte, endByte) {
				return ClassIgnored, true
			}
			return ClassExport, true
		case "export_clause":
			return ClassExport, true
		case "export_statement":
			if fieldContainsRange(parsed, current, "declaration", startByte, endByte) {
				return codeast.ClassUnknown, false
			}
			return ClassExport, true
		}
	}
	return codeast.ClassUnknown, false
}

func isRequireBindingReference(parsed *ParsedFile, node *gotreesitter.Node) bool {
	for current := node; current != nil; current = current.Parent() {
		if nodeKind(parsed, current) != "variable_declarator" {
			continue
		}
		value := childByField(parsed, current, "value")
		if !nodeContainsRequireCall(parsed, value) {
			continue
		}
		name := childByField(parsed, current, "name")
		return name != nil && nodeWithin(node, name)
	}
	return false
}

func nodeContainsRequireCall(parsed *ParsedFile, node *gotreesitter.Node) bool {
	if node == nil {
		return false
	}
	found := false
	walkNamed(node, func(current *gotreesitter.Node) {
		if found || nodeKind(parsed, current) != "call_expression" {
			return
		}
		function := childByField(parsed, current, "function")
		found = function != nil && strings.TrimSpace(nodeText(parsed, function)) == "require"
	})
	return found
}

func isCommonJSExportAssignment(parsed *ParsedFile, root *gotreesitter.Node, node *gotreesitter.Node, targetName string) bool {
	for current := node; current != nil; current = current.Parent() {
		kind := nodeKind(parsed, current)
		if kind != "assignment_expression" && kind != "augmented_assignment_expression" {
			continue
		}
		left := childByField(parsed, current, "left")
		if left == nil || !commonJSExportLeft(parsed, left) {
			continue
		}
		if commonJSExportLeftNamesSymbol(parsed, left, targetName) || !nodeWithin(node, left) || root == nil {
			return true
		}
	}
	return false
}

func commonJSExportLeft(parsed *ParsedFile, left *gotreesitter.Node) bool {
	text := strings.TrimSpace(nodeText(parsed, left))
	return text == "module.exports" ||
		strings.HasPrefix(text, "exports.") ||
		strings.HasPrefix(text, "module.exports.") ||
		strings.HasPrefix(text, "exports[") ||
		strings.HasPrefix(text, "module.exports[")
}

func commonJSExportLeftNamesSymbol(parsed *ParsedFile, left *gotreesitter.Node, targetName string) bool {
	text := strings.TrimSpace(nodeText(parsed, left))
	if text == "module.exports" {
		return true
	}
	for _, prefix := range []string{"exports.", "module.exports."} {
		if strings.HasPrefix(text, prefix) && strings.TrimPrefix(text, prefix) == targetName {
			return true
		}
	}
	for _, prefix := range []string{"exports[", "module.exports["} {
		if strings.HasPrefix(text, prefix) && quotedBracketKey(text) == targetName {
			return true
		}
	}
	return false
}
