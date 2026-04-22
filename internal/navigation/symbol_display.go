package navigation

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func candidateDisplayName(c SymbolCandidate) string {
	if c.Kind != string(ast.SymbolMethod) || c.Receiver == "" {
		return c.Name
	}
	if strings.HasPrefix(c.Receiver, "*") {
		return fmt.Sprintf("(%s).%s", c.Receiver, c.Name)
	}
	return c.Receiver + "." + c.Name
}
