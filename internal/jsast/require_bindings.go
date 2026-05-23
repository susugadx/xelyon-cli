package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

// RequireBindingsWithParsed は CommonJS require が作る local binding を抽出する。
func RequireBindingsWithParsed(parsed *ParsedFile) []ImportBinding {
	if parsed == nil || parsed.tree == nil {
		return nil
	}
	var bindings []ImportBinding
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		if nodeKind(parsed, node) != "variable_declarator" {
			return
		}
		source, imported, ok := requireBindingSourceAndImported(parsed, childByField(parsed, node, "value"))
		if !ok {
			return
		}
		statement := declarationStatementForNode(parsed, node)
		bindings = append(bindings, requireBindingsFromPattern(parsed, childByField(parsed, node, "name"), source, imported, statement)...)
	})
	return bindings
}

func requireCallSource(parsed *ParsedFile, node *gotreesitter.Node) (string, bool) {
	if nodeKind(parsed, node) != "call_expression" {
		return "", false
	}
	function := strings.TrimSpace(nodeText(parsed, childByField(parsed, node, "function")))
	if function != "require" {
		return "", false
	}
	args := childByField(parsed, node, "arguments")
	source := unquoteImportSource(nodeText(parsed, firstNamedChildOfKind(parsed, args, "string")))
	return source, source != ""
}

func requireBindingSourceAndImported(parsed *ParsedFile, node *gotreesitter.Node) (string, string, bool) {
	if source, ok := requireCallSource(parsed, node); ok {
		return source, "default", true
	}
	if nodeKind(parsed, node) != "member_expression" {
		return "", "", false
	}
	source, ok := requireCallSource(parsed, childByField(parsed, node, "object"))
	if !ok {
		return "", "", false
	}
	imported := requireBindingImportedName(parsed, childByField(parsed, node, "property"))
	return source, imported, imported != ""
}

func requireBindingsFromPattern(parsed *ParsedFile, pattern *gotreesitter.Node, source string, imported string, statement *gotreesitter.Node) []ImportBinding {
	switch nodeKind(parsed, pattern) {
	case "identifier":
		if imported == "default" {
			return []ImportBinding{requireDefaultBinding(parsed, pattern, source, statement)}
		}
		return []ImportBinding{requireNamedBinding(parsed, pattern, source, imported, statement)}
	case "object_pattern":
		if imported != "default" {
			return nil
		}
		return requireBindingsFromObjectPattern(parsed, pattern, source, statement)
	default:
		return nil
	}
}

func requireDefaultBinding(parsed *ParsedFile, localNode *gotreesitter.Node, source string, statement *gotreesitter.Node) ImportBinding {
	local := strings.TrimSpace(nodeText(parsed, localNode))
	return (ImportBinding{
		Kind:           ImportBindingDefault,
		Imported:       "default",
		Local:          local,
		Source:         source,
		Line:           int(localNode.StartPoint().Row) + 1,
		localStartByte: localNode.StartByte(),
		localEndByte:   localNode.EndByte(),
	}).withStatementNode(statement)
}

func requireNamedBinding(parsed *ParsedFile, localNode *gotreesitter.Node, source string, imported string, statement *gotreesitter.Node) ImportBinding {
	local := strings.TrimSpace(nodeText(parsed, localNode))
	return (ImportBinding{
		Kind:           ImportBindingNamed,
		Imported:       imported,
		Local:          local,
		Source:         source,
		Line:           int(localNode.StartPoint().Row) + 1,
		localStartByte: localNode.StartByte(),
		localEndByte:   localNode.EndByte(),
	}).withStatementNode(statement)
}

func requireBindingsFromObjectPattern(parsed *ParsedFile, pattern *gotreesitter.Node, source string, statement *gotreesitter.Node) []ImportBinding {
	var bindings []ImportBinding
	for i := 0; i < int(pattern.NamedChildCount()); i++ {
		node := pattern.NamedChild(i)
		switch nodeKind(parsed, node) {
		case "pair_pattern", "pair":
			if binding, ok := requireBindingFromPairPattern(parsed, node, source, statement); ok {
				bindings = append(bindings, binding)
			}
		case "shorthand_property_identifier_pattern", "shorthand_property_identifier":
			bindings = append(bindings, requireBindingFromShorthandPattern(parsed, node, source, statement))
		}
	}
	return bindings
}

func requireBindingFromPairPattern(parsed *ParsedFile, node *gotreesitter.Node, source string, statement *gotreesitter.Node) (ImportBinding, bool) {
	imported := requireBindingImportedName(parsed, childByField(parsed, node, "key"))
	localNode := requireBindingLocalNameNode(parsed, childByField(parsed, node, "value"))
	if imported == "" || localNode == nil {
		return ImportBinding{}, false
	}
	local := strings.TrimSpace(nodeText(parsed, localNode))
	if local == "" {
		return ImportBinding{}, false
	}
	return (ImportBinding{
		Kind:           ImportBindingNamed,
		Imported:       imported,
		Local:          local,
		Source:         source,
		Line:           int(localNode.StartPoint().Row) + 1,
		localStartByte: localNode.StartByte(),
		localEndByte:   localNode.EndByte(),
	}).withStatementNode(statement), true
}

func requireBindingFromShorthandPattern(parsed *ParsedFile, node *gotreesitter.Node, source string, statement *gotreesitter.Node) ImportBinding {
	local := strings.TrimSpace(nodeText(parsed, node))
	return (ImportBinding{
		Kind:           ImportBindingNamed,
		Imported:       local,
		Local:          local,
		Source:         source,
		Line:           int(node.StartPoint().Row) + 1,
		localStartByte: node.StartByte(),
		localEndByte:   node.EndByte(),
	}).withStatementNode(statement)
}

func requireBindingImportedName(parsed *ParsedFile, node *gotreesitter.Node) string {
	text := strings.TrimSpace(nodeText(parsed, node))
	return strings.Trim(text, `"'`)
}

func requireBindingLocalNameNode(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	switch nodeKind(parsed, node) {
	case "identifier", "shorthand_property_identifier", "shorthand_property_identifier_pattern":
		return node
	case "assignment_pattern":
		return node.NamedChild(0)
	default:
		return nil
	}
}
