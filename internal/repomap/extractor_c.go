//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// isCFamily は C系言語かどうかを判定
func isCFamily(ext string) bool {
	return ext == ".c" || ext == ".h" || ext == ".cpp" || ext == ".hpp" || ext == ".cc"
}

func extractCFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	// C/C++では declarator から関数名を取得
	declaratorNode := node.ChildByFieldName("declarator")
	name := ""
	if declaratorNode != nil {
		// function_declarator の場合
		if declaratorNode.Type() == "function_declarator" {
			nameNode := declaratorNode.ChildByFieldName("declarator")
			if nameNode != nil {
				name = nameNode.Content(content)
			}
		} else {
			name = declaratorNode.Content(content)
		}
	}

	// シグネチャ抽出（最初の { まで）
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}
