package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

func findEnclosingScope(parsed *ParsedFile, node *gotreesitter.Node) string {
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "function_declaration", "generator_function_declaration":
			if name := nodeFieldText(parsed, current, "name"); name != "" {
				return "function " + name
			}
		case "method_definition":
			if name := nodeFieldText(parsed, current, "name"); name != "" {
				return "method " + name
			}
		case "class_declaration", "abstract_class_declaration":
			if name := nodeFieldText(parsed, current, "name"); name != "" {
				return "class " + name
			}
		case "arrow_function", "function", "function_expression":
			if name := variableDeclaratorName(parsed, current); name != "" {
				return "function " + name
			}
		}
	}
	return "package-level"
}

func variableDeclaratorName(parsed *ParsedFile, node *gotreesitter.Node) string {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if nodeKind(parsed, current) != "variable_declarator" {
			continue
		}
		return nodeFieldText(parsed, current, "name")
	}
	return ""
}

func nodeFieldText(parsed *ParsedFile, node *gotreesitter.Node, field string) string {
	fieldNode := childByField(parsed, node, field)
	if fieldNode == nil {
		return ""
	}
	return strings.TrimSpace(nodeText(parsed, fieldNode))
}
