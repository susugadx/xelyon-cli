package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

const (
	ClassTypeRef codeast.MatchClass = "type_ref"
	ClassExport  codeast.MatchClass = "export"
	ClassIgnored codeast.MatchClass = "ignored"
)

func classifyNode(parsed *ParsedFile, root *gotreesitter.Node, node *gotreesitter.Node, startByte uint32, endByte uint32, targetName string) codeast.MatchClass {
	if isCommonJSExportAssignment(parsed, root, node, targetName) {
		return ClassExport
	}
	if hasAncestorKind(parsed, node, "comment") || hasAncestorKind(parsed, node, "ERROR") {
		return codeast.ClassComment
	}
	if isStringLikeMatch(parsed, node) {
		return codeast.ClassString
	}
	if isRequireBindingReference(parsed, node) {
		return codeast.ClassImport
	}
	if isDefinitionName(parsed, node, startByte, endByte) {
		return codeast.ClassDef
	}
	if isCallTarget(parsed, node, startByte, endByte) || isNewTarget(parsed, node, startByte, endByte) || isJSXUsageTarget(parsed, node, targetName, startByte, endByte) {
		return codeast.ClassCall
	}
	if isTypeReference(parsed, node, startByte, endByte) {
		return ClassTypeRef
	}
	if class, ok := classifyImportOrExportReference(parsed, node, startByte, endByte); ok {
		return class
	}
	return codeast.ClassRef
}

func matchClassPriority(class codeast.MatchClass) int {
	switch class {
	case codeast.ClassDef:
		return 90
	case codeast.ClassImport, ClassExport:
		return 80
	case codeast.ClassCall:
		return 70
	case ClassTypeRef:
		return 60
	case codeast.ClassRef:
		return 50
	case codeast.ClassString, codeast.ClassComment:
		return 10
	case ClassIgnored:
		return 5
	default:
		return 0
	}
}

func isStringLikeMatch(parsed *ParsedFile, node *gotreesitter.Node) bool {
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "string", "string_fragment":
			return true
		case "template_substitution":
			return false
		case "template_string":
			return true
		}
	}
	return false
}

func isDefinitionName(parsed *ParsedFile, node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "function_declaration", "generator_function_declaration", "class_declaration",
			"interface_declaration", "type_alias_declaration", "enum_declaration",
			"variable_declarator", "method_definition":
			if fieldContainsRange(parsed, current, "name", startByte, endByte) {
				return true
			}
		}
	}
	return false
}

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

func isCallTarget(parsed *ParsedFile, node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	for current := node; current != nil; current = current.Parent() {
		if nodeKind(parsed, current) == "call_expression" && fieldContainsRange(parsed, current, "function", startByte, endByte) {
			return true
		}
	}
	return false
}

func isNewTarget(parsed *ParsedFile, node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	for current := node; current != nil; current = current.Parent() {
		if nodeKind(parsed, current) != "new_expression" {
			continue
		}
		if fieldContainsRange(parsed, current, "constructor", startByte, endByte) || firstNamedChildContainsRange(current, startByte, endByte) {
			return true
		}
	}
	return false
}

func isJSXUsageTarget(parsed *ParsedFile, node *gotreesitter.Node, targetName string, startByte uint32, endByte uint32) bool {
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return false
	}
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "jsx_opening_element", "jsx_self_closing_element":
			return jsxTargetIsBareLocalName(parsed, jsxElementName(parsed, current), targetName, startByte, endByte)
		case "jsx_element":
			if opening := jsxOpeningElement(parsed, current); opening != nil {
				return jsxTargetIsBareLocalName(parsed, jsxElementName(parsed, opening), targetName, startByte, endByte)
			}
			return false
		case "jsx_closing_element":
			return false
		}
	}
	return false
}

func isTypeReference(parsed *ParsedFile, node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	if isDefinitionName(parsed, node, startByte, endByte) {
		return false
	}
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "type_identifier":
			return true
		case "type_annotation", "type_arguments", "generic_type", "extends_type_clause", "implements_clause":
			return true
		}
	}
	return false
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

func fieldContainsRange(parsed *ParsedFile, node *gotreesitter.Node, fieldName string, startByte uint32, endByte uint32) bool {
	field := childByField(parsed, node, fieldName)
	if field == nil {
		return false
	}
	return field.StartByte() <= startByte && endByte <= field.EndByte()
}

func firstNamedChildContainsRange(node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	child := node.NamedChild(0)
	return child != nil && child.StartByte() <= startByte && endByte <= child.EndByte()
}

func nodeWithin(node *gotreesitter.Node, parent *gotreesitter.Node) bool {
	if node == nil || parent == nil {
		return false
	}
	return parent.StartByte() <= node.StartByte() && node.EndByte() <= parent.EndByte()
}

func hasAncestorKind(parsed *ParsedFile, node *gotreesitter.Node, kinds ...string) bool {
	for current := node; current != nil; current = current.Parent() {
		for _, kind := range kinds {
			if nodeKind(parsed, current) == kind {
				return true
			}
		}
	}
	return false
}

func jsxOpeningElement(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	if opening := childByField(parsed, node, "open_tag"); opening != nil {
		return opening
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		kind := nodeKind(parsed, child)
		if kind == "jsx_opening_element" || kind == "jsx_self_closing_element" {
			return child
		}
	}
	return nil
}

func jsxElementName(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	if name := childByField(parsed, node, "name"); name != nil {
		return name
	}
	return node.NamedChild(0)
}

func jsxTargetIsBareLocalName(parsed *ParsedFile, node *gotreesitter.Node, targetName string, startByte uint32, endByte uint32) bool {
	if node == nil {
		return false
	}
	if node.StartByte() > startByte || endByte > node.EndByte() {
		return false
	}
	if strings.TrimSpace(nodeText(parsed, node)) != targetName {
		return false
	}
	start, end := node.StartByte(), node.EndByte()
	if start > 0 && parsed.src[int(start)-1] == '.' {
		return false
	}
	if int(end) < len(parsed.src) && parsed.src[int(end)] == '.' {
		return false
	}
	return true
}

func nodeKind(parsed *ParsedFile, node *gotreesitter.Node) string {
	if parsed == nil || parsed.grammar == nil || node == nil {
		return ""
	}
	return node.Type(parsed.grammar)
}

func nodeText(parsed *ParsedFile, node *gotreesitter.Node) string {
	if parsed == nil || node == nil {
		return ""
	}
	return node.Text(parsed.src)
}

func childByField(parsed *ParsedFile, node *gotreesitter.Node, field string) *gotreesitter.Node {
	if parsed == nil || parsed.grammar == nil || node == nil {
		return nil
	}
	if child := node.ChildByFieldName(field, parsed.grammar); child != nil {
		return child
	}
	switch nodeKind(parsed, node) {
	case "variable_declarator":
		switch field {
		case "name":
			return node.NamedChild(0)
		case "value":
			if count := node.NamedChildCount(); count > 1 {
				return node.NamedChild(count - 1)
			}
		}
	case "assignment_expression", "augmented_assignment_expression":
		switch field {
		case "left":
			return node.NamedChild(0)
		case "right":
			return node.NamedChild(1)
		}
	case "call_expression":
		switch field {
		case "function":
			return node.NamedChild(0)
		case "arguments":
			return node.NamedChild(1)
		}
	case "function_declaration", "generator_function_declaration", "function_signature",
		"class_declaration", "interface_declaration", "type_alias_declaration", "enum_declaration",
		"method_definition", "function", "function_expression", "generator_function",
		"class", "class_expression":
		if field == "name" {
			return firstNamedChildOfKind(parsed, node, "identifier", "type_identifier", "property_identifier")
		}
	case "export_statement":
		if field == "declaration" {
			child := node.NamedChild(0)
			switch nodeKind(parsed, child) {
			case "function_declaration", "generator_function_declaration", "class_declaration",
				"interface_declaration", "type_alias_declaration", "enum_declaration",
				"lexical_declaration", "variable_declaration":
				return child
			}
		}
	case "jsx_opening_element", "jsx_self_closing_element":
		if field == "name" {
			return node.NamedChild(0)
		}
	case "jsx_element":
		if field == "open_tag" {
			return node.NamedChild(0)
		}
	}
	return nil
}

func firstNamedChildOfKind(parsed *ParsedFile, node *gotreesitter.Node, kinds ...string) *gotreesitter.Node {
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		for _, kind := range kinds {
			if nodeKind(parsed, child) == kind {
				return child
			}
		}
	}
	return nil
}
