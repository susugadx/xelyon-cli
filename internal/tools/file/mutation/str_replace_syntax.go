package mutation

import (
	"fmt"
	"strings"

	internalast "github.com/susugadx/xelyon-cli/internal/ast"
)

// validateGoSyntaxForReplace は置換結果の Go ファイルを AST 検証し、構文エラー警告を返す。
func validateGoSyntaxForReplace(path string, content []byte) string {
	errors := internalast.ValidateSyntax(path, content)
	if len(errors) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("⚠️  AST syntax check found issues after replacement:\n")
	for _, syntaxError := range errors {
		fmt.Fprintf(&b, "   • %s\n", syntaxError.Message)
	}
	b.WriteString("   The replacement was still applied. Consider fixing these issues.")
	return b.String()
}

func appendSyntaxWarning(result, warning string) string {
	if warning == "" {
		return result
	}
	return result + "\n\n" + warning
}
