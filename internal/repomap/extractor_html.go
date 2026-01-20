//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// semanticHTMLTags はセマンティックHTML要素のリスト
var semanticHTMLTags = map[string]bool{
	"header":  true,
	"footer":  true,
	"nav":     true,
	"main":    true,
	"aside":   true,
	"article": true,
	"section": true,
	"figure":  true,
	"form":    true,
}

// extractHTMLElement は HTML 要素から id/class 属性を抽出
func extractHTMLElement(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// start_tag を探す
	var startTag *sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "start_tag" {
			startTag = child
			break
		}
	}

	if startTag == nil {
		return symbols
	}

	// タグ名と属性を抽出
	var tagName string
	var idValue, classValue string

	for i := 0; i < int(startTag.ChildCount()); i++ {
		child := startTag.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "tag_name":
			tagName = child.Content(content)
		case "attribute":
			attrName, attrValue := extractHTMLAttribute(child, content)
			switch attrName {
			case "id":
				idValue = attrValue
			case "class":
				classValue = attrValue
			}
		}
	}

	line := int(node.StartPoint().Row) + 1

	// id属性があれば抽出
	if idValue != "" {
		symbols = append(symbols, Symbol{
			Name:      "#" + idValue,
			Kind:      "id",
			Signature: "<" + tagName + " id=\"" + idValue + "\">",
			FilePath:  filePath,
			Line:      line,
		})
	}

	// class属性があれば各クラスを個別に抽出
	if classValue != "" {
		classes := strings.Fields(classValue)
		for _, class := range classes {
			symbols = append(symbols, Symbol{
				Name:      "." + class,
				Kind:      "class",
				Signature: "<" + tagName + " class=\"" + class + "\">",
				FilePath:  filePath,
				Line:      line,
			})
		}
	}

	// セマンティックHTML要素（id/classがない場合でも抽出）
	if semanticHTMLTags[tagName] && idValue == "" && classValue == "" {
		symbols = append(symbols, Symbol{
			Name:      "<" + tagName + ">",
			Kind:      "semantic",
			Signature: "<" + tagName + ">",
			FilePath:  filePath,
			Line:      line,
		})
	}

	return symbols
}

// extractHTMLAttribute は属性ノードから名前と値を抽出
func extractHTMLAttribute(node *sitter.Node, content []byte) (name, value string) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "attribute_name":
			name = child.Content(content)
		case "quoted_attribute_value":
			// quoted_attribute_value の中から attribute_value を探す
			for j := 0; j < int(child.ChildCount()); j++ {
				grandchild := child.Child(j)
				if grandchild != nil && grandchild.Type() == "attribute_value" {
					value = grandchild.Content(content)
					break
				}
			}
		case "attribute_value":
			// クォートなしの属性値
			value = child.Content(content)
		}
	}
	return
}
