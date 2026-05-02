package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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

const (
	attachmentStatusAlreadyAttached         = "Already attached"
	attachmentStatusInvalidDroppedPath      = "Attach failed: invalid dropped path"
	attachmentStatusAttachInvalidPath       = "Attach failed: invalid path"
	attachmentStatusAttachDirectoryNotValid = "Attach failed: directories are not supported"
	attachmentStatusPasteInvalidPath        = "Paste failed: invalid screenshot path"
	attachmentStatusPasteDirectoryNotValid  = "Paste failed: directories are not supported"
	attachmentStatusClipboardAttached       = "Attached screenshot from clipboard"
	attachmentStatusClipboardDuplicate      = "Screenshot already attached"
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

func (m Model) attachmentLimitReachedStatus() string {
	return "Attachment limit reached (" + strconv.Itoa(m.attachmentLimit()) + " max)"
}

func (m Model) attachedBatchStatus(added, limitRejected int) string {
	if limitRejected > 0 {
		return fmt.Sprintf("Attached %d item(s); attachment limit is %d", added, m.attachmentLimit())
	}
	return fmt.Sprintf("Attached %d item(s)", added)
}

func (m *Model) cleanupClipboardTempAttachmentPath(path string) {
	cleanupTemporaryAttachment(composerAttachment{
		Source: composerAttachmentSourceClipboardImage,
		Path:   path,
	})
}

func (m *Model) presentAttachmentAddResult(result addAttachmentFromPathResult, ctx attachmentAddDisplayContext, clipboardPath string) {
	switch result.status {
	case addAttachmentFromPathAdded:
		m.onAttachmentSetChanged()
		if ctx == attachmentAddDisplayClipboardImage {
			m.setTransientStatus(attachmentStatusClipboardAttached)
			return
		}
		m.setTransientStatus(fmt.Sprintf("Attached %s %s (#%d)", result.attachment.kindLabel(), result.attachment.basename(), len(m.attachments)))
	case addAttachmentFromPathDuplicate:
		if ctx == attachmentAddDisplayClipboardImage {
			m.cleanupClipboardTempAttachmentPath(clipboardPath)
			m.setTransientStatus(attachmentStatusClipboardDuplicate)
			return
		}
		m.setTransientStatus(attachmentStatusAlreadyAttached + ": " + result.attachment.basename())
	case addAttachmentFromPathLimit:
		if ctx == attachmentAddDisplayClipboardImage {
			m.cleanupClipboardTempAttachmentPath(clipboardPath)
		}
		m.setTransientStatus(m.attachmentLimitReachedStatus())
	case addAttachmentFromPathDirectory:
		if ctx == attachmentAddDisplayClipboardImage {
			m.cleanupClipboardTempAttachmentPath(clipboardPath)
			m.setTransientStatus(attachmentStatusPasteDirectoryNotValid)
			return
		}
		m.setTransientStatus(attachmentStatusAttachDirectoryNotValid)
	case addAttachmentFromPathStatError:
		if ctx == attachmentAddDisplayClipboardImage {
			m.cleanupClipboardTempAttachmentPath(clipboardPath)
			if result.err != nil {
				m.setTransientStatus("Paste failed: " + result.err.Error())
				return
			}
			m.setTransientStatus(attachmentStatusPasteInvalidPath)
			return
		}
		if result.err != nil {
			m.setTransientStatus("Attach failed: " + result.err.Error())
		} else {
			m.setTransientStatus(attachmentStatusAttachInvalidPath)
		}
	default:
		if ctx == attachmentAddDisplayClipboardImage {
			m.cleanupClipboardTempAttachmentPath(clipboardPath)
			m.setTransientStatus(attachmentStatusPasteInvalidPath)
			return
		}
		m.setTransientStatus(attachmentStatusAttachInvalidPath)
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

func (m Model) visibleAttachmentStartIndex() int {
	start, _ := m.visibleAttachmentRange()
	return start
}

func (m Model) visibleAttachmentRange() (start, end int) {
	if len(m.attachments) == 0 {
		return 0, 0
	}
	maxVisible := m.maxVisibleAttachmentRows()
	if maxVisible <= 0 {
		return len(m.attachments), len(m.attachments)
	}
	if len(m.attachments) <= maxVisible {
		return 0, len(m.attachments)
	}
	return len(m.attachments) - maxVisible, len(m.attachments)
}

func (m Model) visibleAttachmentCount() int {
	start, end := m.visibleAttachmentRange()
	return end - start
}

func (m Model) visibleAttachmentNumber(visibleIndex int) int {
	start, _ := m.visibleAttachmentRange()
	return start + visibleIndex + 1
}

func (m Model) maxVisibleAttachmentRows() int {
	if len(m.attachments) == 0 {
		return 0
	}
	if m.height <= 0 {
		return len(m.attachments)
	}
	remaining := m.maxFooterExpansionRows() - len(m.visibleComposerRows())
	if remaining <= 0 {
		return 0
	}
	return remaining
}

func (m Model) visibleAttachments() []composerAttachment {
	start, end := m.visibleAttachmentRange()
	if start >= end {
		return nil
	}
	out := make([]composerAttachment, end-start)
	copy(out, m.attachments[start:end])
	return out
}
