package ast

import (
	"fmt"
	"strings"

	"github.com/odvcencio/gotreesitter"
)

// ValidateSyntax はソースコードを AST パースし、構文エラー情報を返す。
// 構文エラーがない場合と対応外ファイルの場合は nil を返す。
func ValidateSyntax(path string, src []byte) []SyntaxError {
	if !IsSupportedFile(path) {
		return nil
	}

	tree, err := parseGoSource(src)
	if err != nil {
		return []SyntaxError{{Message: fmt.Sprintf("parse error: %v", err)}}
	}

	root := tree.RootNode()
	if root == nil || !root.HasError() {
		return nil
	}

	errors := collectSyntaxErrors(root, tree.Language(), src)
	if len(errors) > 0 {
		return errors
	}

	return []SyntaxError{{
		Line:    1,
		Column:  1,
		Message: "syntax error",
	}}
}

func collectSyntaxErrors(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte) []SyntaxError {
	if node == nil {
		return nil
	}

	errors := make([]SyntaxError, 0, 5)
	seen := make(map[string]struct{})
	stack := []*gotreesitter.Node{node}

	for len(stack) > 0 && len(errors) < 5 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}

		if current.IsError() || current.IsMissing() {
			syntaxError := buildSyntaxError(current, lang, src)
			key := fmt.Sprintf("%d:%d:%s", syntaxError.Line, syntaxError.Column, syntaxError.Message)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				errors = append(errors, syntaxError)
			}
		}

		for i := current.ChildCount() - 1; i >= 0; i-- {
			child := current.Child(i)
			if child != nil {
				stack = append(stack, child)
			}
		}
	}

	return errors
}

func buildSyntaxError(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte) SyntaxError {
	line := int(node.StartPoint().Row) + 1
	column := int(node.StartPoint().Column) + 1
	snippet := syntaxErrorSnippet(node, src)

	message := fmt.Sprintf("syntax error at L%d:%d", line, column)
	if node.IsMissing() {
		missingType := strings.TrimSpace(node.Type(lang))
		if missingType != "" {
			message = fmt.Sprintf("missing %s at L%d:%d", missingType, line, column)
		} else {
			message = fmt.Sprintf("missing syntax element at L%d:%d", line, column)
		}
	}
	if snippet != "" {
		message += fmt.Sprintf(" near %q", snippet)
	}

	return SyntaxError{
		Line:    line,
		Column:  column,
		Message: message,
	}
}

func syntaxErrorSnippet(node *gotreesitter.Node, src []byte) string {
	start := int(node.StartByte())
	end := int(node.EndByte())
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	if start >= end {
		return ""
	}

	snippet := string(src[start:end])
	snippet = strings.NewReplacer("\n", "\\n", "\r", "", "\t", "\\t").Replace(snippet)
	snippet = strings.TrimSpace(snippet)
	if snippet == "" {
		return ""
	}
	if len(snippet) > 30 {
		return snippet[:30] + "..."
	}
	return snippet
}
