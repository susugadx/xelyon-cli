package tui

import tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"

type droppedPathAttachKind int

const (
	droppedPathAttachNotPath droppedPathAttachKind = iota
	droppedPathAttachAdded
	droppedPathAttachDuplicate
	droppedPathAttachLimit
	droppedPathAttachInvalid
)

type droppedPathAttachResult struct {
	kind          droppedPathAttachKind
	added         int
	duplicate     int
	limitRejected int
}

func (r droppedPathAttachResult) shouldFallbackToText() bool {
	return r.kind == droppedPathAttachNotPath
}

func (m *Model) tryAttachDroppedPaths(content string) droppedPathAttachResult {
	decision := decideDroppedPathHandling(content)
	switch decision.kind {
	case droppedPathDecisionFallbackText:
		return droppedPathAttachResult{kind: droppedPathAttachNotPath}
	case droppedPathDecisionLimit:
		m.setTransientStatus(m.attachmentLimitReachedStatus())
		return droppedPathAttachResult{kind: droppedPathAttachLimit}
	default:
		return m.applyDroppedPathCandidates(decision.paths)
	}
}

func (m *Model) applyDroppedPathCandidates(paths []string) droppedPathAttachResult {
	result := droppedPathAttachResult{kind: droppedPathAttachInvalid}
	invalid := 0
	for _, path := range paths {
		added := m.addAttachmentFromPath(path, tuiattachments.SourceDroppedPath)
		switch added.status {
		case addAttachmentFromPathAdded:
			result.added++
		case addAttachmentFromPathDuplicate:
			result.duplicate++
		case addAttachmentFromPathLimit:
			result.limitRejected++
		default:
			invalid++
		}
	}

	switch {
	case result.added > 0:
		m.onAttachmentSetChanged()
		result.kind = droppedPathAttachAdded
		m.setTransientStatus(m.attachedBatchStatus(result.added, result.limitRejected))
	case result.limitRejected > 0:
		result.kind = droppedPathAttachLimit
		m.setTransientStatus(m.attachmentLimitReachedStatus())
	case result.duplicate > 0 && invalid == 0:
		result.kind = droppedPathAttachDuplicate
		m.setTransientStatus(attachmentStatusAlreadyAttached)
	default:
		result.kind = droppedPathAttachInvalid
		m.setTransientStatus(attachmentStatusInvalidDroppedPath)
	}
	return result
}
