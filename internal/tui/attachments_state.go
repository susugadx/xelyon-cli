package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type composerAttachmentKind int

const (
	composerAttachmentFile composerAttachmentKind = iota
	composerAttachmentImage
)

type composerAttachmentSource int

const (
	composerAttachmentSourceUnknown composerAttachmentSource = iota
	composerAttachmentSourceDroppedPath
	composerAttachmentSourceClipboardImage
	composerAttachmentSourceCommand
)

type composerAttachment struct {
	Kind   composerAttachmentKind
	Source composerAttachmentSource
	Path   string
	Size   int64
}

const maxComposerAttachments = 12

type appendAttachmentResult int

const (
	appendAttachmentAdded appendAttachmentResult = iota
	appendAttachmentRejectedEmptyPath
	appendAttachmentRejectedDuplicate
	appendAttachmentRejectedLimit
)

type addAttachmentFromPathStatus int

const (
	addAttachmentFromPathAdded addAttachmentFromPathStatus = iota
	addAttachmentFromPathEmptyPath
	addAttachmentFromPathStatError
	addAttachmentFromPathDirectory
	addAttachmentFromPathDuplicate
	addAttachmentFromPathLimit
)

type addAttachmentFromPathResult struct {
	status     addAttachmentFromPathStatus
	attachment composerAttachment
	err        error
}

type attachmentAddDisplayContext int

const (
	attachmentAddDisplayCommand attachmentAddDisplayContext = iota
	attachmentAddDisplayClipboardImage
)

func (a composerAttachment) basename() string {
	base := filepath.Base(a.Path)
	if base == "." || base == string(filepath.Separator) {
		return a.Path
	}
	return base
}

func (a composerAttachment) kindLabel() string {
	if a.Kind == composerAttachmentImage {
		return "image"
	}
	return "file"
}

func (m *Model) clearAttachments() {
	for _, att := range m.attachments {
		cleanupTemporaryAttachment(att)
	}
	m.attachments = nil
}

func (m Model) hasAttachments() bool {
	return len(m.attachments) > 0
}

func (m Model) attachmentSnapshot() []composerAttachment {
	if len(m.attachments) == 0 {
		return nil
	}
	out := make([]composerAttachment, len(m.attachments))
	copy(out, m.attachments)
	return out
}

func (m Model) attachmentLimit() int {
	return maxComposerAttachments
}

func (m Model) hasAttachmentCapacity() bool {
	return len(m.attachments) < m.attachmentLimit()
}

func (m *Model) appendAttachment(att composerAttachment) bool {
	return m.appendAttachmentWithResult(att) == appendAttachmentAdded
}

func (m *Model) appendAttachmentWithResult(att composerAttachment) appendAttachmentResult {
	path := strings.TrimSpace(att.Path)
	if path == "" {
		return appendAttachmentRejectedEmptyPath
	}
	att.Path = path
	for _, existing := range m.attachments {
		if existing.Path == att.Path {
			return appendAttachmentRejectedDuplicate
		}
	}
	if !m.hasAttachmentCapacity() {
		return appendAttachmentRejectedLimit
	}
	m.attachments = append(m.attachments, att)
	return appendAttachmentAdded
}

func (m *Model) addAttachmentFromPath(path string, source composerAttachmentSource) addAttachmentFromPathResult {
	path = strings.TrimSpace(path)
	if path == "" {
		return addAttachmentFromPathResult{status: addAttachmentFromPathEmptyPath}
	}

	info, err := os.Stat(path)
	if err != nil {
		return addAttachmentFromPathResult{status: addAttachmentFromPathStatError, err: err}
	}
	if info.IsDir() {
		return addAttachmentFromPathResult{status: addAttachmentFromPathDirectory}
	}

	kind := composerAttachmentFile
	if isImageAttachmentPath(path) {
		kind = composerAttachmentImage
	}
	att := composerAttachment{
		Kind:   kind,
		Source: source,
		Path:   path,
		Size:   info.Size(),
	}
	switch m.appendAttachmentWithResult(att) {
	case appendAttachmentAdded:
		return addAttachmentFromPathResult{status: addAttachmentFromPathAdded, attachment: att}
	case appendAttachmentRejectedDuplicate:
		return addAttachmentFromPathResult{status: addAttachmentFromPathDuplicate, attachment: att}
	case appendAttachmentRejectedLimit:
		return addAttachmentFromPathResult{status: addAttachmentFromPathLimit, attachment: att}
	default:
		return addAttachmentFromPathResult{status: addAttachmentFromPathEmptyPath}
	}
}

func (m *Model) onAttachmentSetChanged() {
	m.syncComposerLayout()
	m.refreshSlashSuggestions()
	m.chromeDirty = true
}

func (m *Model) removeLastAttachment() bool {
	if len(m.attachments) == 0 {
		return false
	}
	return m.removeAttachmentAt(len(m.attachments) - 1)
}

func (m *Model) removeAttachmentAt(index int) bool {
	if index < 0 || index >= len(m.attachments) {
		return false
	}
	removed := m.attachments[index]
	m.attachments = slices.Delete(m.attachments, index, index+1)
	cleanupTemporaryAttachment(removed)
	m.onAttachmentSetChanged()
	return true
}

func (m *Model) clearAllAttachments() bool {
	if len(m.attachments) == 0 {
		return false
	}
	m.clearAttachments()
	m.onAttachmentSetChanged()
	return true
}
