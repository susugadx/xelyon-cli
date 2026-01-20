//go:build !norepomap
// +build !norepomap

package repomap

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// extractTailwindConfigSymbols は Tailwind CSS 設定ファイルからシンボルを抽出
func extractTailwindConfigSymbols(filePath string) (*FileSymbols, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// ファイル拡張子から言語を判定
	ext := strings.ToLower(filepath.Ext(filePath))
	var lang *sitter.Language
	switch ext {
	case ".ts", ".mts":
		lang = SupportedLanguages[".ts"]
	default:
		lang = SupportedLanguages[".js"]
	}

	if lang == nil {
		return nil, nil
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
	symbols := extractTailwindConfig(root, content, filePath)

	return &FileSymbols{
		Path:    filePath,
		Symbols: symbols,
	}, nil
}

// extractTailwindConfig は Tailwind config AST からシンボルを抽出
func extractTailwindConfig(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	nodeType := node.Type()

	// theme.extend のキーを探す
	if nodeType == "pair" {
		keyNode := findChildByType(node, "property_identifier")
		if keyNode == nil {
			keyNode = findChildByType(node, "string")
		}

		if keyNode != nil {
			key := strings.Trim(keyNode.Content(content), `"'`)

			// theme キーの場合
			if key == "theme" {
				themeSymbols := extractTailwindTheme(node, content, filePath, "theme")
				symbols = append(symbols, themeSymbols...)
			}

			// plugins キーの場合
			if key == "plugins" {
				pluginSymbols := extractTailwindPlugins(node, content, filePath)
				symbols = append(symbols, pluginSymbols...)
			}

			// content キーの場合（Tailwind v3+）
			if key == "content" {
				symbols = append(symbols, Symbol{
					Name:      "content",
					Kind:      "config",
					Signature: "content: [...]",
					FilePath:  filePath,
					Line:      int(node.StartPoint().Row) + 1,
				})
			}
		}
	}

	// 子ノードを再帰的に処理
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			symbols = append(symbols, extractTailwindConfig(child, content, filePath)...)
		}
	}

	return symbols
}

// extractTailwindTheme は theme/theme.extend からカスタマイズ項目を抽出
func extractTailwindTheme(node *sitter.Node, content []byte, filePath string, prefix string) []Symbol {
	var symbols []Symbol

	// value を取得（object）
	valueNode := findChildByType(node, "object")
	if valueNode == nil {
		return symbols
	}

	// object 内の各 pair を処理
	for i := 0; i < int(valueNode.ChildCount()); i++ {
		child := valueNode.Child(i)
		if child == nil || child.Type() != "pair" {
			continue
		}

		keyNode := findChildByType(child, "property_identifier")
		if keyNode == nil {
			keyNode = findChildByType(child, "string")
		}

		if keyNode == nil {
			continue
		}

		key := strings.Trim(keyNode.Content(content), `"'`)
		fullKey := prefix + "." + key

		// extend の場合は再帰
		if key == "extend" {
			extendSymbols := extractTailwindTheme(child, content, filePath, prefix+".extend")
			symbols = append(symbols, extendSymbols...)
			continue
		}

		// theme.extend.colors などの主要カテゴリを抽出
		themeCategories := map[string]bool{
			"colors": true, "spacing": true, "fontSize": true,
			"fontFamily": true, "screens": true, "borderRadius": true,
			"boxShadow": true, "animation": true, "keyframes": true,
			"container": true, "extend": true,
		}

		if themeCategories[key] || strings.HasPrefix(prefix, "theme.extend") {
			// ネストされた値の数をカウント
			childValues := countObjectKeys(child)
			sig := fullKey
			if childValues > 0 {
				sig = fullKey + " (" + strconv.Itoa(childValues) + " items)"
			}

			symbols = append(symbols, Symbol{
				Name:      fullKey,
				Kind:      "theme",
				Signature: sig,
				FilePath:  filePath,
				Line:      int(child.StartPoint().Row) + 1,
			})
		}
	}

	return symbols
}

// extractTailwindPlugins は plugins 配列からプラグイン名を抽出
func extractTailwindPlugins(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// value を取得（array）
	valueNode := findChildByType(node, "array")
	if valueNode == nil {
		return symbols
	}

	// array 内の各要素を処理
	for i := 0; i < int(valueNode.ChildCount()); i++ {
		child := valueNode.Child(i)
		if child == nil {
			continue
		}

		var pluginName string

		switch child.Type() {
		case "call_expression":
			// require('@tailwindcss/forms') のパターン
			funcNode := findChildByType(child, "identifier")
			if funcNode != nil && funcNode.Content(content) == "require" {
				argsNode := findChildByType(child, "arguments")
				if argsNode != nil {
					strNode := findChildByType(argsNode, "string")
					if strNode != nil {
						pluginName = strings.Trim(strNode.Content(content), `"'`)
					}
				}
			}
		case "identifier":
			// plugin変数参照
			pluginName = child.Content(content)
		case "member_expression":
			// plugin.xxx パターン
			pluginName = child.Content(content)
		}

		if pluginName != "" {
			symbols = append(symbols, Symbol{
				Name:      pluginName,
				Kind:      "plugin",
				Signature: "plugin: " + pluginName,
				FilePath:  filePath,
				Line:      int(child.StartPoint().Row) + 1,
			})
		}
	}

	return symbols
}

// findChildByType は指定タイプの子ノードを探す
func findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == nodeType {
			return child
		}
	}
	return nil
}

// countObjectKeys は object 内のキー数をカウント
func countObjectKeys(node *sitter.Node) int {
	valueNode := findChildByType(node, "object")
	if valueNode == nil {
		return 0
	}

	count := 0
	for i := 0; i < int(valueNode.ChildCount()); i++ {
		child := valueNode.Child(i)
		if child != nil && child.Type() == "pair" {
			count++
		}
	}
	return count
}
