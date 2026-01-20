//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func extractCSSRuleSet(node *sitter.Node, content []byte, filePath string) Symbol {
	// セレクタを取得
	var selector string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "selectors" {
			selector = child.Content(content)
			break
		}
	}

	// セレクタが取れなかった場合は最初の部分
	if selector == "" {
		selector = strings.TrimSpace(strings.Split(node.Content(content), "{")[0])
	}

	// 長すぎるセレクタは省略
	if len(selector) > 80 {
		selector = selector[:80] + "..."
	}

	kind := "selector"
	// セレクタタイプの判定
	if strings.HasPrefix(selector, "#") {
		kind = "id"
	} else if strings.HasPrefix(selector, ".") {
		kind = "class"
	} else if strings.HasPrefix(selector, "@") {
		kind = "at-rule"
	}

	return Symbol{
		Name:      selector,
		Kind:      kind,
		Signature: selector,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractCSSKeyframes(node *sitter.Node, content []byte, filePath string) Symbol {
	// @keyframes name を取得
	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "keyframes_name" {
			name = child.Content(content)
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "keyframes",
		Signature: "@keyframes " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractCSSMediaQuery(node *sitter.Node, content []byte, filePath string) Symbol {
	// @media クエリを取得
	nodeContent := node.Content(content)
	// 最初の { まで
	sig := strings.TrimSpace(strings.Split(nodeContent, "{")[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	return Symbol{
		Name:      sig,
		Kind:      "media",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func isCSSCustomProperty(node *sitter.Node, content []byte) bool {
	// CSS カスタムプロパティ（--xxx）かどうか
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "property_name" {
			name := child.Content(content)
			return strings.HasPrefix(name, "--")
		}
	}
	return false
}

func extractCSSCustomProperty(node *sitter.Node, content []byte, filePath string) Symbol {
	var name, value string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "property_name":
			name = child.Content(content)
		case "plain_value", "integer_value", "color_value":
			value = child.Content(content)
		}
	}

	sig := name
	if value != "" {
		sig = name + ": " + value
	}

	return Symbol{
		Name:      name,
		Kind:      "custom-property",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}
