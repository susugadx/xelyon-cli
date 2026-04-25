package tui

func (m Model) hasComposerDraft() bool {
	return m.composer.HasDraft(m.textInput.Value())
}

func (m Model) hasSubmittableComposerContent() bool {
	return m.composer.HasSubmittableContent(m.textInput.Value())
}

func (m Model) hasFoldedPasteBlocks() bool {
	return m.composer.HasFoldedPasteBlocks()
}

func (m Model) isPlainComposerInput() bool {
	return m.composer.IsPlainInput()
}

func (m Model) buildComposerPayload() string {
	return m.composer.BuildPayload(m.textInput.Value())
}
