package tui

import (
	"github.com/susugadx/xelyon-cli/internal/tui/commandrouter"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

type commandSubmissionDecisionKind int

const (
	commandSubmissionDecisionLocalSyntaxError commandSubmissionDecisionKind = iota
	commandSubmissionDecisionLocalAction
	commandSubmissionDecisionDispatchAgent
	commandSubmissionDecisionFallbackChat
)

type commandSubmissionDecision struct {
	kind        commandSubmissionDecisionKind
	action      commandrouter.Action
	errorDetail string
}

func decideCommandSubmission(command slash.Command, hasMouseSelection bool) commandSubmissionDecision {
	ctx := commandrouter.Context{HasMouseSelection: hasMouseSelection}
	action := commandrouter.Route(command, ctx)
	if !command.ParseOK() {
		if commandrouter.UsesRawTUILocalArgs(command, ctx) && isLocalCommandAction(action) {
			return commandSubmissionDecision{kind: commandSubmissionDecisionLocalAction, action: action}
		}
		if isLocalCommandAction(action) {
			return commandSubmissionDecision{
				kind:        commandSubmissionDecisionLocalSyntaxError,
				action:      action,
				errorDetail: command.ParseStatus.ErrorSummary(),
			}
		}
		return commandSubmissionDecision{kind: commandSubmissionDecisionFallbackChat, action: action}
	}

	if isLocalCommandAction(action) {
		return commandSubmissionDecision{kind: commandSubmissionDecisionLocalAction, action: action}
	}
	if action == commandrouter.ActionDispatchAgent {
		return commandSubmissionDecision{kind: commandSubmissionDecisionDispatchAgent, action: action}
	}
	return commandSubmissionDecision{kind: commandSubmissionDecisionFallbackChat, action: action}
}

func isLocalCommandAction(action commandrouter.Action) bool {
	_, ok := localCommandActionHandlers[action]
	return ok
}
