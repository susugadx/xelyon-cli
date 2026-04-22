package navigation

import "github.com/susugadx/xelyon-cli/internal/ast"

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
