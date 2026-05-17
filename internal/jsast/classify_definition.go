package jsast

import "github.com/odvcencio/gotreesitter"

func isDefinitionName(parsed *ParsedFile, node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "function_declaration", "generator_function_declaration", "class_declaration",
			"interface_declaration", "type_alias_declaration", "enum_declaration",
			"variable_declarator":
			if fieldContainsRange(parsed, current, "name", startByte, endByte) {
				return true
			}
		case "method_definition", "method_signature", "abstract_method_signature":
			if isTypeBodyMethodNode(parsed, current) && fieldContainsRange(parsed, current, "name", startByte, endByte) {
				return true
			}
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
