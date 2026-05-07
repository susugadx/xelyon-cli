package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

type composerSubmissionKind int

const (
	composerSubmissionChat composerSubmissionKind = iota
	composerSubmissionCommand
)

type composerSubmission struct {
	kind         composerSubmissionKind
	commandInput string
	payload      string
	attachments  []composerAttachment
}

func (m Model) buildComposerSubmission() (composerSubmission, bool) {
	if !m.hasSubmittableComposerContent() {
		return composerSubmission{}, false
	}
	attachments := m.attachmentSnapshot()

	if input, isCommand := slash.TrimmedInput(m.textInput.Value()); isCommand {
		payload := input
		if !m.isPlainComposerInput() {
			payload = m.buildComposerPayload()
		}
		return composerSubmission{
			kind:         composerSubmissionCommand,
			commandInput: input,
			payload:      payload,
			attachments:  attachments,
		}, true
	}

	if m.isPlainComposerInput() {
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" && len(attachments) == 0 {
			return composerSubmission{}, false
		}
		return composerSubmission{
			kind:        composerSubmissionChat,
			payload:     input,
			attachments: attachments,
		}, true
	}

	return composerSubmission{
		kind:        composerSubmissionChat,
		payload:     m.buildComposerPayload(),
		attachments: attachments,
	}, true
}

func (m Model) resolveComposerCommand(sub composerSubmission) slash.Command {
	return slash.NewCommand(sub.commandInput, sub.payload)
}
