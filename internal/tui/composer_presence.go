package tui

import "strings"

func (m Model) hasComposerDraft() bool {
	return strings.TrimSpace(m.textInput.Value()) != "" || len(m.composerParts) > 0 || len(m.pasteBlocks) > 0
}

func (m Model) hasSubmittableComposerContent() bool {
	if strings.TrimSpace(m.textInput.Value()) != "" {
		return true
	}
	for _, part := range m.composerParts {
		switch part.kind {
		case composerPartText:
			if strings.TrimSpace(part.text) != "" {
				return true
			}
		case composerPartPaste:
			block, ok := m.findPasteBlock(part.pasteUID)
			if ok && block.content != "" {
				return true
			}
		}
	}
	return false
}

func (m Model) hasFoldedPasteBlocks() bool {
	return len(m.pasteBlocks) > 0
}

func (m Model) isPlainComposerInput() bool {
	return len(m.composerParts) == 0 && len(m.pasteBlocks) == 0
}

func (m Model) findPasteBlock(uid int) (pasteBlock, bool) {
	for _, block := range m.pasteBlocks {
		if block.uid == uid {
			return block, true
		}
	}
	return pasteBlock{}, false
}

func (m Model) buildComposerPayload() string {
	var builder strings.Builder
	for _, part := range m.composerParts {
		switch part.kind {
		case composerPartText:
			builder.WriteString(part.text)
		case composerPartPaste:
			block, ok := m.findPasteBlock(part.pasteUID)
			if ok {
				builder.WriteString(block.content)
			}
		}
	}
	builder.WriteString(m.textInput.Value())
	return builder.String()
}
