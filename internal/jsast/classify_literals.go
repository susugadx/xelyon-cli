package jsast

import "github.com/odvcencio/gotreesitter"

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
