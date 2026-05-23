package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

// ExportBinding は export specifier が参照する local/exported name と re-export source を表す。
type ExportBinding struct {
	Local    string
	Exported string
	Source   string
	Line     int
}

// ExportBindingsWithParsed は export specifier の binding を抽出する。
func ExportBindingsWithParsed(parsed *ParsedFile) []ExportBinding {
	if parsed == nil || parsed.tree == nil {
		return nil
	}
	var bindings []ExportBinding
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		if nodeKind(parsed, node) != "export_statement" {
			return
		}
		source := exportStatementSource(parsed, node)
		walkNamed(node, func(current *gotreesitter.Node) {
			if nodeKind(parsed, current) != "export_specifier" {
				return
			}
			if binding, ok := exportBindingFromSpecifier(parsed, current, source); ok {
				bindings = append(bindings, binding)
			}
		})
	})
	return bindings
}

// SymbolExportedAsDefaultWithParsed は指定 symbol が default export されているかを返す。
func SymbolExportedAsDefaultWithParsed(parsed *ParsedFile, name string) bool {
	name = strings.TrimSpace(name)
	if parsed == nil || parsed.tree == nil || name == "" {
		return false
	}
	for _, binding := range ExportBindingsWithParsed(parsed) {
		if binding.Source == "" && binding.Local == name && binding.Exported == "default" {
			return true
		}
	}
	if symbolAssignedToCommonJSDefaultWithParsed(parsed, name) {
		return true
	}
	found := false
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		if found || nodeKind(parsed, node) != "export_statement" {
			return
		}
		exported, ok := defaultExportedExpressionName(parsed, node)
		found = ok && exported == name
	})
	return found
}

func symbolAssignedToCommonJSDefaultWithParsed(parsed *ParsedFile, name string) bool {
	found := false
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		if found || nodeKind(parsed, node) != "assignment_expression" {
			return
		}
		left := childByField(parsed, node, "left")
		if strings.TrimSpace(nodeText(parsed, left)) != "module.exports" {
			return
		}
		right := childByField(parsed, node, "right")
		found = declarationValueName(parsed, right) == name
	})
	return found
}

func defaultExportedExpressionName(parsed *ParsedFile, node *gotreesitter.Node) (string, bool) {
	if !strings.HasPrefix(strings.TrimSpace(nodeText(parsed, node)), "export default") {
		return "", false
	}
	if declaration := childByField(parsed, node, "declaration"); declaration != nil {
		nameNode := childByField(parsed, declaration, "name")
		name := strings.TrimSpace(nodeText(parsed, nameNode))
		return name, name != ""
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch nodeKind(parsed, child) {
		case "identifier", "type_identifier":
			name := strings.TrimSpace(nodeText(parsed, child))
			return name, name != ""
		}
	}
	return "", false
}

func exportBindingFromSpecifier(parsed *ParsedFile, node *gotreesitter.Node, source string) (ExportBinding, bool) {
	localNode := childByField(parsed, node, "name")
	if localNode == nil {
		return ExportBinding{}, false
	}
	exportedNode := childByField(parsed, node, "alias")
	if exportedNode == nil {
		exportedNode = localNode
	}
	local := strings.TrimSpace(nodeText(parsed, localNode))
	exported := strings.TrimSpace(nodeText(parsed, exportedNode))
	if local == "" || exported == "" {
		return ExportBinding{}, false
	}
	return ExportBinding{
		Local:    local,
		Exported: exported,
		Source:   source,
		Line:     int(node.StartPoint().Row) + 1,
	}, true
}

func exportStatementSource(parsed *ParsedFile, node *gotreesitter.Node) string {
	if parsed == nil || parsed.grammar == nil || node == nil {
		return ""
	}
	source := node.ChildByFieldName("source", parsed.grammar)
	if source != nil {
		return unquoteImportSource(nodeText(parsed, source))
	}
	if !strings.Contains(nodeText(parsed, node), " from ") {
		return ""
	}
	for i := node.NamedChildCount() - 1; i >= 0; i-- {
		child := node.NamedChild(i)
		if nodeKind(parsed, child) == "string" {
			return unquoteImportSource(nodeText(parsed, child))
		}
	}
	return ""
}
