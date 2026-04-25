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
}

func (m Model) buildComposerSubmission() (composerSubmission, bool) {
	if !m.hasSubmittableComposerContent() {
		return composerSubmission{}, false
	}

	if input, isCommand := slash.TrimmedInput(m.textInput.Value()); isCommand {
		payload := input
		if !m.isPlainComposerInput() {
			payload = m.buildComposerPayload()
		}
		return composerSubmission{
			kind:         composerSubmissionCommand,
			commandInput: input,
			payload:      payload,
		}, true
	}

	if m.isPlainComposerInput() {
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return composerSubmission{}, false
		}
		return composerSubmission{
			kind:    composerSubmissionChat,
			payload: input,
		}, true
	}

	return composerSubmission{
		kind:    composerSubmissionChat,
		payload: m.buildComposerPayload(),
	}, true
}

func (m Model) resolveComposerCommand(sub composerSubmission) slash.Command {
	return slash.NewCommand(sub.commandInput, sub.payload, m.commands.ResolveAlias)
}
