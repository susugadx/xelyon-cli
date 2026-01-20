//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func extractRustFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャは関数定義行全体（最初の { まで）
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = "fn " + name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRustStruct(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "struct",
		Signature: "struct " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRustImpl(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// impl対象の型名を取得
	typeNode := node.ChildByFieldName("type")
	typeName := ""
	if typeNode != nil {
		typeName = typeNode.Content(content)
	}

	// impl ブロック内のメソッドを抽出
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "declaration_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				item := child.Child(j)
				if item.Type() == "function_item" {
					nameNode := item.ChildByFieldName("name")
					name := ""
					if nameNode != nil {
						name = nameNode.Content(content)
					}

					startByte := item.StartByte()
					sig := ""
					for k := startByte; k < item.EndByte(); k++ {
						if content[k] == '{' {
							sig = strings.TrimSpace(string(content[startByte:k]))
							break
						}
					}
					if sig == "" {
						sig = "fn " + name + "(...)"
					}

					symbols = append(symbols, Symbol{
						Name:      typeName + "::" + name,
						Kind:      "method",
						Signature: sig,
						FilePath:  filePath,
						Line:      int(item.StartPoint().Row) + 1,
					})
				}
			}
		}
	}

	return symbols
}

func extractRustTrait(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "trait",
		Signature: "trait " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRustEnum(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "enum",
		Signature: "enum " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}
