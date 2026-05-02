package tui

import "os"

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
	parsed := parseDroppedPaths(content)
	return m.applyDroppedPathParseResult(parsed)
}

func (m *Model) applyDroppedPathParseResult(parsed droppedPathParseResult) droppedPathAttachResult {
	switch parsed.kind {
	case droppedPathParseNotPath:
		return droppedPathAttachResult{kind: droppedPathAttachNotPath}
	case droppedPathParseLimit:
		m.setTransientStatus(m.attachmentLimitReachedStatus())
		return droppedPathAttachResult{kind: droppedPathAttachLimit}
	case droppedPathParseInvalid:
		m.setTransientStatus(attachmentStatusInvalidDroppedPath)
		return droppedPathAttachResult{kind: droppedPathAttachInvalid}
	default:
		if !allDroppedPathsAttachable(parsed.paths) {
			return droppedPathAttachResult{kind: droppedPathAttachNotPath}
		}
		return m.applyDroppedPathCandidates(parsed.paths)
	}
}

func allDroppedPathsAttachable(paths []string) bool {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func (m *Model) applyDroppedPathCandidates(paths []string) droppedPathAttachResult {
	result := droppedPathAttachResult{kind: droppedPathAttachInvalid}
	invalid := 0
	for _, path := range paths {
		added := m.addAttachmentFromPath(path, composerAttachmentSourceDroppedPath)
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
