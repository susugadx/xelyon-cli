package tui

import "strings"

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

type composerCommand struct {
	input        string
	payload      string
	resolvedName string
	argCount     int
}

func trimmedCommandInput(value string) (string, bool) {
	input := strings.TrimSpace(value)
	return input, strings.HasPrefix(input, "/")
}

func (m Model) buildComposerSubmission() (composerSubmission, bool) {
	if !m.hasSubmittableComposerContent() {
		return composerSubmission{}, false
	}

	if input, isCommand := trimmedCommandInput(m.textInput.Value()); isCommand {
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

func (m Model) resolveComposerCommand(sub composerSubmission) composerCommand {
	cmdParts := strings.Fields(sub.commandInput)
	resolvedCommand := sub.commandInput
	if len(cmdParts) > 0 {
		resolvedCommand = m.agent.ResolveAlias(cmdParts[0])
	}
	return composerCommand{
		input:        sub.commandInput,
		payload:      sub.payload,
		resolvedName: resolvedCommand,
		argCount:     len(cmdParts),
	}
}

func (c composerCommand) isBare(name string) bool {
	return c.resolvedName == name && c.argCount == 1
}

func (c composerCommand) matches(name string) bool {
	return c.resolvedName == name
}
