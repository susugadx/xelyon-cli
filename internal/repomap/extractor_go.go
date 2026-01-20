//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// extractGoFunction はGo関数を抽出
func extractGoFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャは関数定義行全体
	startByte := node.StartByte()
	// 最初の { まで取得
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = strings.TrimSpace(string(content[startByte:node.EndByte()]))
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// extractGoMethod はGoメソッドを抽出
func extractGoMethod(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "method",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// extractGoType はGo型定義を抽出
func extractGoType(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// type_spec を探す
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_spec" {
			nameNode := child.ChildByFieldName("name")
			typeNode := child.ChildByFieldName("type")

			name := ""
			if nameNode != nil {
				name = nameNode.Content(content)
			}

			kind := "type"
			if typeNode != nil {
				switch typeNode.Type() {
				case "struct_type":
					kind = "struct"
				case "interface_type":
					kind = "interface"
				}
			}

			sig := "type " + name
			if typeNode != nil {
				sig = strings.TrimSpace(string(content[child.StartByte():child.EndByte()]))
				// 長すぎる場合は省略
				if len(sig) > 100 {
					sig = sig[:100] + "..."
				}
			}

			symbols = append(symbols, Symbol{
				Name:      name,
				Kind:      kind,
				Signature: sig,
				FilePath:  filePath,
				Line:      int(child.StartPoint().Row) + 1,
			})
		}
	}

	return symbols
}
