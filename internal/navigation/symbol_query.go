package navigation

import (
	"fmt"
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

func extractMethodReceiver(signature string) string {
	signature = strings.TrimSpace(signature)
	if !strings.HasPrefix(signature, "func") {
		return ""
	}

	rest := strings.TrimSpace(strings.TrimPrefix(signature, "func"))
	if !strings.HasPrefix(rest, "(") {
		return ""
	}

	closeIdx := strings.Index(rest, ")")
	if closeIdx <= 1 {
		return ""
	}

	receiverSpec := strings.TrimSpace(rest[1:closeIdx])
	if receiverSpec == "" {
		return ""
	}

	fields := strings.Fields(receiverSpec)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func canonicalReceiver(receiver string) string {
	receiver = strings.TrimSpace(receiver)
	for strings.HasPrefix(receiver, "(") && strings.HasSuffix(receiver, ")") && len(receiver) > 1 {
		receiver = strings.TrimSpace(receiver[1 : len(receiver)-1])
	}
	if idx := strings.LastIndexAny(receiver, " \t"); idx >= 0 {
		receiver = strings.TrimSpace(receiver[idx+1:])
	}
	receiver = strings.TrimSpace(strings.TrimPrefix(receiver, "*"))
	return stripTypeArguments(receiver)
}

func stripTypeArguments(receiver string) string {
	if receiver == "" {
		return ""
	}

	var b strings.Builder
	depth := 0
	for _, r := range receiver {
		switch r {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func candidateDisplayName(c SymbolCandidate) string {
	if c.Kind != string(ast.SymbolMethod) || c.Receiver == "" {
		return c.Name
	}
	if strings.HasPrefix(c.Receiver, "*") {
		return fmt.Sprintf("(%s).%s", c.Receiver, c.Name)
	}
	return c.Receiver + "." + c.Name
}

type candidateShapeDecision struct {
	Matched bool
	Precise bool
}

func candidateShapeMatch(ref Reference, cand SymbolCandidate) candidateShapeDecision {
	switch cand.Kind {
	case string(ast.SymbolMethod):
		if ref.NodeType == "identifier" {
			return candidateShapeDecision{Matched: false, Precise: true}
		}
		if ref.NodeType != "field_identifier" {
			return candidateShapeDecision{Matched: true, Precise: false}
		}
		switch ref.SelectorKind {
		case "package":
			return candidateShapeDecision{Matched: false, Precise: true}
		case "method":
			if cand.Receiver == "" {
				return candidateShapeDecision{Matched: true, Precise: true}
			}
			if ref.ReceiverType == "" {
				return candidateShapeDecision{Matched: false, Precise: false}
			}
			return candidateShapeDecision{
				Matched: canonicalReceiver(ref.ReceiverType) == canonicalReceiver(cand.Receiver),
				Precise: true,
			}
		default:
			if cand.Receiver == "" {
				return candidateShapeDecision{Matched: true, Precise: false}
			}
			return candidateShapeDecision{Matched: false, Precise: false}
		}
	case string(ast.SymbolFunction):
		if ref.NodeType != "field_identifier" {
			return candidateShapeDecision{Matched: true, Precise: false}
		}
		switch ref.SelectorKind {
		case "package":
			return candidateShapeDecision{Matched: true, Precise: true}
		case "method":
			return candidateShapeDecision{Matched: false, Precise: true}
		default:
			return candidateShapeDecision{Matched: false, Precise: false}
		}
	default:
		return candidateShapeDecision{Matched: true, Precise: false}
	}
}
