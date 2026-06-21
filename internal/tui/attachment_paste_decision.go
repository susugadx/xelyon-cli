package tui

import tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"

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
	if parsed.Kind == tuiattachments.DroppedPathParseLimit {
		return droppedPathDecision{kind: droppedPathDecisionLimit}
	}
	if parsed.Kind != tuiattachments.DroppedPathParseReady {
		return droppedPathDecision{kind: droppedPathDecisionFallbackText}
	}
	if droppedPathsAttachability(parsed.Paths) != droppedPathAttachabilityAttachable {
		return droppedPathDecision{kind: droppedPathDecisionFallbackText}
	}
	return droppedPathDecision{kind: droppedPathDecisionApplyCandidates, paths: parsed.Paths}
}
