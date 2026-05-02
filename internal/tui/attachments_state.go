package tui

import (
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

func (m *Model) appendAttachment(att composerAttachment) bool {
	path := strings.TrimSpace(att.Path)
	if path == "" {
		return false
	}
	att.Path = path
	for _, existing := range m.attachments {
		if existing.Path == att.Path {
			return false
		}
	}
	m.attachments = append(m.attachments, att)
	return true
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
	m.syncComposerLayout()
	m.refreshSlashSuggestions()
	m.chromeDirty = true
	return true
}

func (m *Model) clearAllAttachments() bool {
	if len(m.attachments) == 0 {
		return false
	}
	m.clearAttachments()
	m.syncComposerLayout()
	m.refreshSlashSuggestions()
	m.chromeDirty = true
	return true
}

func (m Model) visibleAttachmentStartIndex() int {
	if len(m.attachments) == 0 {
		return 0
	}
	maxVisible := m.maxVisibleAttachmentRows()
	if maxVisible <= 0 || len(m.attachments) <= maxVisible {
		return 0
	}
	return len(m.attachments) - maxVisible
}

func (m Model) visibleAttachmentNumber(visibleIndex int) int {
	return m.visibleAttachmentStartIndex() + visibleIndex + 1
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
	maxVisible := m.maxVisibleAttachmentRows()
	if maxVisible <= 0 || len(m.attachments) == 0 {
		return nil
	}
	if len(m.attachments) <= maxVisible {
		out := make([]composerAttachment, len(m.attachments))
		copy(out, m.attachments)
		return out
	}
	out := make([]composerAttachment, maxVisible)
	copy(out, m.attachments[len(m.attachments)-maxVisible:])
	return out
}
