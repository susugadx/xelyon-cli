package jsast

import "github.com/odvcencio/gotreesitter"

func isCallTarget(parsed *ParsedFile, node *gotreesitter.Node, startByte uint32, endByte uint32) bool {
	for current := node; current != nil; current = current.Parent() {
		if nodeKind(parsed, current) != "call_expression" {
			continue
		}
		function := childByField(parsed, current, "function")
		if function == nil || !nodeContainsRange(function, startByte, endByte) {
			continue
		}
		for _, candidate := range callTargetNameNodes(parsed, function) {
			if nodeContainsRange(candidate, startByte, endByte) {
				return true
			}
		}
	}
	return false
}

func callTargetNameNodes(parsed *ParsedFile, function *gotreesitter.Node) []*gotreesitter.Node {
	tail := calleeTailNameNode(parsed, function)
	if !calleeTailNameIs(parsed, tail, "call", "apply") {
		return []*gotreesitter.Node{tail}
	}
	receiver := callApplyReceiverNode(parsed, function)
	if receiver == nil {
		return []*gotreesitter.Node{tail}
	}
	if target := calleeTailNameNode(parsed, receiver); target != nil {
		return []*gotreesitter.Node{tail, target}
	}
	return []*gotreesitter.Node{tail}
}

func callApplyReceiverNode(parsed *ParsedFile, function *gotreesitter.Node) *gotreesitter.Node {
	if function == nil {
		return nil
	}
	switch nodeKind(parsed, function) {
	case "member_expression", "optional_chain":
		return childByField(parsed, function, "object")
	default:
		return nil
	}
}

func calleeTailNameIs(parsed *ParsedFile, node *gotreesitter.Node, names ...string) bool {
	if node == nil {
		return false
	}
	got := nodeText(parsed, node)
	for _, name := range names {
		if got == name {
			return true
		}
	}
	return false
}

func calleeTailNameNode(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	switch nodeKind(parsed, node) {
	case "identifier", "property_identifier", "private_property_identifier":
		return node
	}
	if property := childByField(parsed, node, "property"); property != nil {
		return calleeTailNameNode(parsed, property)
	}
	if name := childByField(parsed, node, "name"); name != nil {
		return calleeTailNameNode(parsed, name)
	}
	for idx := int(node.NamedChildCount()) - 1; idx >= 0; idx-- {
		if tail := calleeTailNameNode(parsed, node.NamedChild(idx)); tail != nil {
			return tail
		}
	}
	return nil
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
