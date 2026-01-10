//go:build !norepomap
// +build !norepomap

package repomap

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ExtractSymbols はファイルからシンボルを抽出
func ExtractSymbols(filePath string) (*FileSymbols, error) {
	lang := GetLanguage(filePath)
	if lang == nil {
		return nil, nil // サポートされていない言語
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	root := tree.RootNode()
	symbols := extractFromNode(root, content, filePath)

	return &FileSymbols{
		Path:    filePath,
		Symbols: symbols,
	}, nil
}

// extractFromNode はノードからシンボルを再帰的に抽出
func extractFromNode(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// ノードタイプに応じてシンボルを抽出
	// 拡張子で言語を判定
	ext := filepath.Ext(filePath)
	switch node.Type() {
	// Go
	case "function_declaration":
		if ext == ".go" {
			symbols = append(symbols, extractGoFunction(node, content, filePath))
		} else {
			symbols = append(symbols, extractJSFunction(node, content, filePath))
		}
	case "method_declaration":
		symbols = append(symbols, extractGoMethod(node, content, filePath))
	case "type_declaration":
		symbols = append(symbols, extractGoType(node, content, filePath)...)

	// JavaScript/TypeScript
	case "function":
		symbols = append(symbols, extractJSFunction(node, content, filePath))
	case "class_declaration":
		symbols = append(symbols, extractJSClass(node, content, filePath))
	case "method_definition":
		symbols = append(symbols, extractJSMethod(node, content, filePath))

	// Python
	case "function_definition":
		symbols = append(symbols, extractPyFunction(node, content, filePath))
	case "class_definition":
		symbols = append(symbols, extractPyClass(node, content, filePath))
	}

	// 子ノードを再帰的に処理
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		symbols = append(symbols, extractFromNode(child, content, filePath)...)
	}

	return symbols
}

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

// JS/Python用の抽出関数も同様に実装
func extractJSFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: "function " + name + "(...)",
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractJSClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractJSMethod(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "method",
		Signature: name + "(...)",
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractPyFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: "def " + name + "(...)",
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractPyClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}
