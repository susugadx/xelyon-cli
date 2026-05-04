package tui

import (
	"fmt"
	"strconv"
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
