package tui

type droppedPathDecisionKind int

const (
	droppedPathDecisionFallbackText droppedPathDecisionKind = iota
	droppedPathDecisionLimit
	droppedPathDecisionApplyCandidates
)

type droppedPathDecision struct {
	kind  droppedPathDecisionKind
	paths []string
}

// decideDroppedPathHandling は pasted content を text として扱うか、添付処理に進めるかを判定する owner。
func decideDroppedPathHandling(content string) droppedPathDecision {
	parsed := parseDroppedPaths(content)
	if parsed.kind == droppedPathParseLimit {
		return droppedPathDecision{kind: droppedPathDecisionLimit}
	}
	if parsed.kind != droppedPathParseReady {
		return droppedPathDecision{kind: droppedPathDecisionFallbackText}
	}
	if droppedPathsAttachability(parsed.paths) != droppedPathAttachabilityAttachable {
		return droppedPathDecision{kind: droppedPathDecisionFallbackText}
	}
	return droppedPathDecision{kind: droppedPathDecisionApplyCandidates, paths: parsed.paths}
}
