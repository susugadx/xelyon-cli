package jsast

import "github.com/odvcencio/gotreesitter"

func isTypeBodyMethodNode(parsed *ParsedFile, node *gotreesitter.Node) bool {
	switch nodeKind(parsed, node) {
	case "method_definition", "method_signature", "abstract_method_signature":
	default:
		return false
	}

	for current := node.Parent(); current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "class_body", "interface_body":
			return true
		case "object_type":
			return objectTypeIsNamedTypeAliasBody(parsed, current)
		case "object", "object_pattern", "statement_block", "program":
			return false
		}
	}
	return false
}

func objectTypeIsNamedTypeAliasBody(parsed *ParsedFile, node *gotreesitter.Node) bool {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "type_alias_declaration":
			return true
		case "intersection_type", "union_type", "parenthesized_type":
			continue
		default:
			return false
		}
	}
	return false
}
