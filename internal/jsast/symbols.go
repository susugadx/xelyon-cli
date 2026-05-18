package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

func symbolFromNode(parsed *ParsedFile, node *gotreesitter.Node) (Symbol, bool) {
	switch nodeKind(parsed, node) {
	case "function_declaration", "generator_function_declaration", "function_signature":
		return namedDeclarationSymbol(parsed, node, "function")
	case "class_declaration":
		return namedDeclarationSymbol(parsed, node, "class")
	case "interface_declaration":
		return namedDeclarationSymbol(parsed, node, "interface")
	case "type_alias_declaration":
		return namedDeclarationSymbol(parsed, node, "type")
	case "enum_declaration":
		return namedDeclarationSymbol(parsed, node, "enum")
	case "method_definition", "method_signature", "abstract_method_signature":
		return methodDeclarationSymbol(parsed, node)
	case "public_field_definition":
		return propertyDeclarationSymbol(parsed, node, "field")
	case "property_signature":
		return propertyDeclarationSymbol(parsed, node, "property")
	case "variable_declarator":
		return variableDeclaratorSymbol(parsed, node)
	case "assignment_expression":
		return commonJSAssignmentSymbol(parsed, node)
	default:
		return Symbol{}, false
	}
}

func methodDeclarationSymbol(parsed *ParsedFile, node *gotreesitter.Node) (Symbol, bool) {
	if !isTypeBodyMethodNode(parsed, node) {
		return Symbol{}, false
	}
	symbol, ok := namedDeclarationSymbol(parsed, node, "method")
	if !ok {
		return Symbol{}, false
	}
	if typeBodyMemberHasNonPublicAccess(parsed, node) {
		symbol.Exported = false
	}
	return symbol, true
}

func propertyDeclarationSymbol(parsed *ParsedFile, node *gotreesitter.Node, kind string) (Symbol, bool) {
	if !isTypeBodyPropertyNode(parsed, node) {
		return Symbol{}, false
	}
	symbol, ok := namedDeclarationSymbol(parsed, node, kind)
	if !ok {
		return Symbol{}, false
	}
	if typeBodyMemberHasNonPublicAccess(parsed, node) {
		symbol.Exported = false
	}
	return symbol, true
}

func namedDeclarationSymbol(parsed *ParsedFile, node *gotreesitter.Node, kind string) (Symbol, bool) {
	nameNode := childByField(parsed, node, "name")
	if nameNode == nil {
		return Symbol{}, false
	}
	name := strings.TrimSpace(nodeText(parsed, nameNode))
	if name == "" {
		return Symbol{}, false
	}
	start := node.StartPoint()
	end := node.EndPoint()
	return Symbol{
		Name:      name,
		Kind:      kind,
		Signature: signatureForNode(parsed, node),
		Line:      int(start.Row) + 1,
		EndLine:   int(end.Row) + 1,
		Character: lspCharacterForByteOffset(parsed.src, nameNode.StartByte()),
		Exported:  hasAncestorKind(parsed, node, "export_statement"),
	}, true
}

func typeBodyMemberHasNonPublicAccess(parsed *ParsedFile, node *gotreesitter.Node) bool {
	nameNode := childByField(parsed, node, "name")
	if nameNode == nil {
		return false
	}
	name := strings.TrimSpace(nodeText(parsed, nameNode))
	if strings.HasPrefix(name, "#") {
		return true
	}
	prefix := strings.TrimSpace(nodePrefixText(parsed, node, nameNode))
	for _, field := range strings.Fields(prefix) {
		switch field {
		case "private", "protected":
			return true
		}
	}
	return false
}

func nodePrefixText(parsed *ParsedFile, node *gotreesitter.Node, child *gotreesitter.Node) string {
	if parsed == nil || node == nil || child == nil || child.StartByte() < node.StartByte() {
		return ""
	}
	start, end := int(node.StartByte()), int(child.StartByte())
	if start < 0 || end > len(parsed.src) || start > end {
		return ""
	}
	return string(parsed.src[start:end])
}

func variableDeclaratorSymbol(parsed *ParsedFile, node *gotreesitter.Node) (Symbol, bool) {
	nameNode := childByField(parsed, node, "name")
	if nameNode == nil || nodeKind(parsed, nameNode) != "identifier" {
		return Symbol{}, false
	}
	name := strings.TrimSpace(nodeText(parsed, nameNode))
	if name == "" {
		return Symbol{}, false
	}
	kind := "const"
	if value := childByField(parsed, node, "value"); value != nil {
		kind = kindForVariableInitializer(parsed, value)
	}
	defNode := declarationStatementForNode(parsed, node)
	start := defNode.StartPoint()
	end := defNode.EndPoint()
	return Symbol{
		Name:      name,
		Kind:      kind,
		Signature: signatureForNode(parsed, defNode),
		Line:      int(start.Row) + 1,
		EndLine:   int(end.Row) + 1,
		Character: lspCharacterForByteOffset(parsed.src, nameNode.StartByte()),
		Exported:  hasAncestorKind(parsed, node, "export_statement"),
	}, true
}

func commonJSAssignmentSymbol(parsed *ParsedFile, node *gotreesitter.Node) (Symbol, bool) {
	left := childByField(parsed, node, "left")
	right := childByField(parsed, node, "right")
	if left == nil || right == nil {
		return Symbol{}, false
	}
	name, ok := commonJSExportedName(parsed, left)
	if !ok {
		return Symbol{}, false
	}
	kind := kindForInitializer(parsed, right)
	if kind != "function" && kind != "class" {
		return Symbol{}, false
	}
	if name == "" {
		name = declarationValueName(parsed, right)
	}
	if name == "" {
		return Symbol{}, false
	}
	start := node.StartPoint()
	end := node.EndPoint()
	return Symbol{
		Name:      name,
		Kind:      kind,
		Signature: signatureForNode(parsed, node),
		Line:      int(start.Row) + 1,
		EndLine:   int(end.Row) + 1,
		Character: commonJSAssignmentSymbolCharacter(parsed, left, right, name),
		Exported:  true,
	}, true
}

func commonJSExportedName(parsed *ParsedFile, left *gotreesitter.Node) (string, bool) {
	text := strings.TrimSpace(nodeText(parsed, left))
	switch {
	case text == "module.exports":
		return "", true
	case strings.HasPrefix(text, "exports."):
		return strings.TrimPrefix(text, "exports."), true
	case strings.HasPrefix(text, "module.exports."):
		return strings.TrimPrefix(text, "module.exports."), true
	case strings.HasPrefix(text, "exports[") || strings.HasPrefix(text, "module.exports["):
		if key := quotedBracketKey(text); key != "" {
			return key, true
		}
	}
	return "", false
}

func quotedBracketKey(text string) string {
	start := strings.IndexAny(text, `'"`)
	if start < 0 {
		return ""
	}
	quote := text[start]
	rest := text[start+1:]
	end := strings.IndexByte(rest, quote)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func commonJSAssignmentSymbolCharacter(parsed *ParsedFile, left *gotreesitter.Node, right *gotreesitter.Node, name string) int {
	if node := commonJSExportedNameNode(parsed, left, name); node != nil {
		return lspCharacterForByteOffset(parsed.src, node.StartByte())
	}
	if node := declarationValueNameNode(parsed, right, name); node != nil {
		return lspCharacterForByteOffset(parsed.src, node.StartByte())
	}
	return lspCharacterForByteOffset(parsed.src, left.StartByte())
}

func commonJSExportedNameNode(parsed *ParsedFile, left *gotreesitter.Node, name string) *gotreesitter.Node {
	if left == nil || name == "" {
		return nil
	}
	text := strings.TrimSpace(nodeText(parsed, left))
	if text == "module.exports" {
		return nil
	}
	for i := 0; i < left.NamedChildCount(); i++ {
		child := left.NamedChild(i)
		if child != nil && strings.TrimSpace(nodeText(parsed, child)) == name {
			return child
		}
	}
	return nil
}

func declarationValueNameNode(parsed *ParsedFile, value *gotreesitter.Node, name string) *gotreesitter.Node {
	if value == nil || name == "" {
		return nil
	}
	switch nodeKind(parsed, value) {
	case "function", "function_expression", "generator_function", "class", "class_expression":
		nameNode := childByField(parsed, value, "name")
		if nameNode != nil && strings.TrimSpace(nodeText(parsed, nameNode)) == name {
			return nameNode
		}
	case "identifier":
		if strings.TrimSpace(nodeText(parsed, value)) == name {
			return value
		}
	}
	return nil
}

func declarationValueName(parsed *ParsedFile, value *gotreesitter.Node) string {
	switch nodeKind(parsed, value) {
	case "function", "function_expression", "generator_function", "class", "class_expression":
		if nameNode := childByField(parsed, value, "name"); nameNode != nil {
			return strings.TrimSpace(nodeText(parsed, nameNode))
		}
	case "identifier":
		return strings.TrimSpace(nodeText(parsed, value))
	}
	return ""
}

func kindForInitializer(parsed *ParsedFile, value *gotreesitter.Node) string {
	switch nodeKind(parsed, value) {
	case "arrow_function", "function", "function_expression", "generator_function", "generator_function_declaration":
		return "function"
	case "class", "class_expression", "class_declaration":
		return "class"
	default:
		return "const"
	}
}

func kindForVariableInitializer(parsed *ParsedFile, value *gotreesitter.Node) string {
	if parsed.lang == LanguageTypeScript && (nodeKind(parsed, value) == "function" || nodeKind(parsed, value) == "function_expression") {
		return "const"
	}
	return kindForInitializer(parsed, value)
}

func declarationStatementForNode(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "lexical_declaration", "variable_declaration", "export_statement":
			return current
		}
	}
	return node
}

func signatureForNode(parsed *ParsedFile, node *gotreesitter.Node) string {
	if node == nil {
		return ""
	}
	text := strings.TrimSpace(nodeText(parsed, node))
	if text == "" {
		return ""
	}
	if idx := strings.Index(text, "\n"); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
	}
	return text
}
