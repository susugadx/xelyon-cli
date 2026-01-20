//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func extractSwiftFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャ抽出
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = "func " + name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSwiftClass(node *sitter.Node, content []byte, filePath string) Symbol {
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

func extractSwiftStruct(node *sitter.Node, content []byte, filePath string) Symbol {
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

func extractSwiftProtocol(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "protocol",
		Signature: "protocol " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}
