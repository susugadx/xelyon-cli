package navigation

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

type symbolQuery struct {
	Raw      string
	BaseName string
	Receiver string
}

func parseSymbolQuery(raw string) symbolQuery {
	raw = strings.TrimSpace(raw)
	query := symbolQuery{Raw: raw, BaseName: raw}
	if raw == "" {
		return query
	}

	if idx := strings.LastIndex(raw, "."); idx > 0 && idx < len(raw)-1 {
		receiver := strings.TrimSpace(raw[:idx])
		name := strings.TrimSpace(raw[idx+1:])
		if receiver != "" && name != "" {
			query.BaseName = name
			query.Receiver = canonicalReceiver(receiver)
		}
	}

	return query
}

func symbolQueryMatches(query symbolQuery, symbol ast.Symbol) bool {
	if symbol.Name != query.BaseName {
		return false
	}
	if query.Receiver == "" {
		return true
	}
	if symbol.Kind != ast.SymbolMethod {
		return false
	}
	return canonicalReceiver(extractMethodReceiver(symbol.Signature)) == query.Receiver
}
