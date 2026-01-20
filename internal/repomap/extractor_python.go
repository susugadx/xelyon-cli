//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func extractPyFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	paramsNode := node.ChildByFieldName("parameters")
	returnTypeNode := node.ChildByFieldName("return_type")

	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// パラメータから型注釈を含むシグネチャを抽出
	params := "()"
	if paramsNode != nil {
		params = extractPyParams(paramsNode, content)
	}

	// async def 検出: 親ノードまたは前の兄弟ノードをチェック
	isAsync := isPythonAsyncFunction(node)

	prefix := "def"
	if isAsync {
		prefix = "async def"
	}

	sig := prefix + " " + name + params
	if returnTypeNode != nil {
		sig += " -> " + returnTypeNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// isPythonAsyncFunction はPython関数がasyncかどうかを判定
func isPythonAsyncFunction(node *sitter.Node) bool {
	// function_definition の前の兄弟ノードをチェック
	parent := node.Parent()
	if parent != nil {
		for i := 0; i < int(parent.ChildCount()); i++ {
			child := parent.Child(i)
			if child == node {
				// 自分自身の前をチェック
				if i > 0 {
					prev := parent.Child(i - 1)
					if prev != nil && prev.Type() == "async" {
						return true
					}
				}
				break
			}
		}
	}

	// ノード自身の子ノードをチェック（一部のパーサーバージョン用）
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "async" {
			return true
		}
	}

	return false
}

// extractPyParams はPythonパラメータから型注釈を含むシグネチャを抽出
func extractPyParams(paramsNode *sitter.Node, content []byte) string {
	params := paramsNode.Content(content)
	// 改行を除去してコンパクトに
	params = strings.ReplaceAll(params, "\n", " ")
	params = strings.ReplaceAll(params, "\t", "")
	// 連続スペースを単一に
	for strings.Contains(params, "  ") {
		params = strings.ReplaceAll(params, "  ", " ")
	}
	return params
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
