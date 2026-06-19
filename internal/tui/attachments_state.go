package tui

import (
	"os"
	"slices"
	"strings"

	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
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
	attachment tuiattachments.Attachment
	err        error
}

type attachmentAddDisplayContext int

const (
	attachmentAddDisplayCommand attachmentAddDisplayContext = iota
	attachmentAddDisplayClipboardImage
)

func (m *Model) clearAttachments() {
	for _, att := range m.attachments {
		cleanupTemporaryAttachment(att)
	}
	m.attachments = nil
}

func (m *Model) detachAttachmentsWithoutCleanup() []tuiattachments.Attachment {
	if len(m.attachments) == 0 {
		return nil
	}

	out := make([]tuiattachments.Attachment, len(m.attachments))
	copy(out, m.attachments)
	m.attachments = nil
	return out
}

func (m Model) hasAttachments() bool {
	return len(m.attachments) > 0
}

func (m Model) attachmentSnapshot() []tuiattachments.Attachment {
	if len(m.attachments) == 0 {
		return nil
	}
	out := make([]tuiattachments.Attachment, len(m.attachments))
	copy(out, m.attachments)
	return out
}

func (m Model) attachmentLimit() int {
	return tuiattachments.MaxComposerAttachments
}

func (m *Model) appendAttachment(att tuiattachments.Attachment) bool {
	return m.appendAttachmentWithResult(att) == tuiattachments.AppendAdded
}

func (m *Model) appendAttachmentWithResult(att tuiattachments.Attachment) tuiattachments.AppendResult {
	prepared, result := tuiattachments.PrepareAppend(m.attachments, att, m.attachmentLimit())
	if result != tuiattachments.AppendAdded {
		return result
	}
	m.attachments = append(m.attachments, prepared)
	return result
}

func (m *Model) addAttachmentFromPath(path string, source tuiattachments.Source) addAttachmentFromPathResult {
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

	kind := tuiattachments.KindFile
	if isImageAttachmentPath(path) {
		kind = tuiattachments.KindImage
	}
	att := tuiattachments.Attachment{
		Kind:   kind,
		Source: source,
		Path:   path,
		Size:   info.Size(),
	}
	switch m.appendAttachmentWithResult(att) {
	case tuiattachments.AppendAdded:
		return addAttachmentFromPathResult{status: addAttachmentFromPathAdded, attachment: att}
	case tuiattachments.AppendRejectedDuplicate:
		return addAttachmentFromPathResult{status: addAttachmentFromPathDuplicate, attachment: att}
	case tuiattachments.AppendRejectedLimit:
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
