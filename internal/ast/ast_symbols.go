package ast

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/odvcencio/gotreesitter"
)

// ExtractSymbols はファイルから定義シンボルを抽出する。
func ExtractSymbols(path string) ([]Symbol, error) {
	tree, src, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	return extractSymbolsFromTree(tree, src)
}

// ExtractSymbolsFromBytes はソースコードバイト列から定義シンボルを抽出する。
func ExtractSymbolsFromBytes(path string, src []byte) ([]Symbol, error) {
	tree, src, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	return extractSymbolsFromTree(tree, src)
}

func extractSymbolsFromTree(tree *gotreesitter.Tree, src []byte) ([]Symbol, error) {
	query, err := getGoSymbolQuery()
	if err != nil {
		return nil, err
	}

	lang := tree.Language()
	cursor := query.Exec(tree.RootNode(), lang, src)
	var symbols []Symbol
	seen := make(map[string]int)

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		var defNode *gotreesitter.Node
		var typeBody *gotreesitter.Node
		nameNodes := make([]*gotreesitter.Node, 0, len(match.Captures))
		for _, capture := range match.Captures {
			switch capture.Name {
			case "def":
				if defNode == nil {
					defNode = capture.Node
				}
			case "type_body":
				if typeBody == nil {
					typeBody = capture.Node
				}
			case "name":
				if capture.Node != nil {
					nameNodes = append(nameNodes, capture.Node)
				}
			}
		}
		if defNode == nil || len(nameNodes) == 0 {
			continue
		}

		kind := symbolKindForNode(defNode, typeBody, lang)
		signature := extractSignature(defNode, src, lang)
		line := int(defNode.StartPoint().Row) + 1
		endLine := int(defNode.EndPoint().Row) + 1
		for _, nameNode := range nameNodes {
			name := strings.TrimSpace(nameNode.Text(src))
			if name == "" {
				continue
			}
			symbol := Symbol{
				Name:      name,
				Kind:      kind,
				Signature: signature,
				Line:      line,
				EndLine:   endLine,
				Exported:  isExportedIdentifier(name),
			}
			appendSymbol(&symbols, seen, symbol)
		}
	}

	collectTypeAliasSymbols(tree.RootNode(), src, lang, &symbols, seen)

	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].Line != symbols[j].Line {
			return symbols[i].Line < symbols[j].Line
		}
		if symbols[i].EndLine != symbols[j].EndLine {
			return symbols[i].EndLine < symbols[j].EndLine
		}
		return symbols[i].Name < symbols[j].Name
	})
	return symbols, nil
}

func symbolKindForNode(defNode, typeBody *gotreesitter.Node, lang *gotreesitter.Language) SymbolKind {
	switch defNode.Type(lang) {
	case "function_declaration":
		return SymbolFunction
	case "method_declaration":
		return SymbolMethod
	case "type_alias":
		return SymbolType
	case "const_spec":
		return SymbolConst
	case "var_spec":
		return SymbolVar
	case "type_spec":
		if typeBody == nil {
			return SymbolType
		}
		switch typeBody.Type(lang) {
		case "struct_type":
			return SymbolStruct
		case "interface_type":
			return SymbolInterface
		default:
			return SymbolType
		}
	default:
		return SymbolUnknown
	}
}

func extractSignature(defNode *gotreesitter.Node, src []byte, lang *gotreesitter.Language) string {
	if defNode == nil {
		return ""
	}

	switch defNode.Type(lang) {
	case "function_declaration", "method_declaration":
		if body := defNode.ChildByFieldName("body", lang); body != nil {
			return strings.TrimSpace(string(src[defNode.StartByte():body.StartByte()]))
		}
	case "type_alias":
		return "type " + strings.TrimSpace(defNode.Text(src))
	case "type_spec":
		return "type " + strings.TrimSpace(defNode.Text(src))
	case "const_spec":
		return "const " + strings.TrimSpace(defNode.Text(src))
	case "var_spec":
		return "var " + strings.TrimSpace(defNode.Text(src))
	}

	return strings.TrimSpace(defNode.Text(src))
}

func isExportedIdentifier(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	return unicode.IsUpper(r)
}

func collectTypeAliasSymbols(node *gotreesitter.Node, src []byte, lang *gotreesitter.Language, symbols *[]Symbol, seen map[string]int) {
	if node == nil {
		return
	}

	stack := []*gotreesitter.Node{node}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}

		if current.Type(lang) == "type_alias" {
			nameNode := current.NamedChild(0)
			if nameNode != nil {
				name := strings.TrimSpace(nameNode.Text(src))
				if name != "" {
					symbol := Symbol{
						Name:      name,
						Kind:      SymbolType,
						Signature: extractSignature(current, src, lang),
						Line:      int(current.StartPoint().Row) + 1,
						EndLine:   int(current.EndPoint().Row) + 1,
						Exported:  isExportedIdentifier(name),
					}
					appendSymbol(symbols, seen, symbol)
				}
			}
		}

		for i := int(current.NamedChildCount()) - 1; i >= 0; i-- {
			child := current.NamedChild(i)
			if child != nil {
				stack = append(stack, child)
			}
		}
	}
}

func appendSymbol(symbols *[]Symbol, seen map[string]int, symbol Symbol) {
	key := fmt.Sprintf("%d:%d:%s", symbol.Line, symbol.EndLine, symbol.Name)
	if idx, ok := seen[key]; ok {
		if (*symbols)[idx].Kind == SymbolType && symbol.Kind != SymbolType {
			(*symbols)[idx] = symbol
		}
		return
	}
	seen[key] = len(*symbols)
	*symbols = append(*symbols, symbol)
}
