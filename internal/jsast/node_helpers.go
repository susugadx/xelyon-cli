package jsast

import "github.com/odvcencio/gotreesitter"

func fieldContainsRange(parsed *ParsedFile, node *gotreesitter.Node, fieldName string, startByte uint32, endByte uint32) bool {
	field := childByField(parsed, node, fieldName)
	return nodeContainsRange(field, startByte, endByte)
}

func firstNamedChildContainsRange(node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	child := node.NamedChild(0)
	return nodeContainsRange(child, startByte, endByte)
}

func nodeContainsRange(node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	return node != nil && node.StartByte() <= startByte && endByte <= node.EndByte()
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
	case "member_expression", "optional_chain":
		switch field {
		case "object":
			return node.NamedChild(0)
		case "property":
			if count := node.NamedChildCount(); count > 1 {
				return node.NamedChild(count - 1)
			}
		}
	case "function_declaration", "generator_function_declaration", "function_signature",
		"class_declaration", "interface_declaration", "type_alias_declaration", "enum_declaration",
		"method_definition", "method_signature", "abstract_method_signature",
		"function", "function_expression", "generator_function",
		"class", "class_expression":
		if field == "name" {
			return firstNamedChildOfKind(parsed, node, "identifier", "type_identifier", "property_identifier", "private_property_identifier")
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
