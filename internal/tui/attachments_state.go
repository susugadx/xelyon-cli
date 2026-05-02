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
	removed := m.attachments[len(m.attachments)-1]
	m.attachments = slices.Delete(m.attachments, len(m.attachments)-1, len(m.attachments))
	cleanupTemporaryAttachment(removed)
	m.syncComposerLayout()
	m.refreshSlashSuggestions()
	m.chromeDirty = true
	return true
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
