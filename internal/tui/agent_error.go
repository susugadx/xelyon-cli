package tui

import "errors"

// AgentErrorKind は agent turn の失敗種別を TUI 表示へ渡す内部 contract。
type AgentErrorKind string

const (
	AgentErrorUnknown    AgentErrorKind = "error"
	AgentErrorProvider   AgentErrorKind = "provider"
	AgentErrorTool       AgentErrorKind = "tool"
	AgentErrorValidation AgentErrorKind = "validation"
	AgentErrorStartup    AgentErrorKind = "startup"
)

// AgentTurnError は error chain に agent turn の失敗種別を付与する。
type AgentTurnError struct {
	Kind AgentErrorKind
	Err  error
}

func (e *AgentTurnError) Error() string {
	if e == nil || e.Err == nil {
		return string(AgentErrorUnknown)
	}
	return e.Err.Error()
}

func (e *AgentTurnError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func WrapAgentTurnError(kind AgentErrorKind, err error) error {
	if err == nil {
		return nil
	}
	return &AgentTurnError{
		Kind: normalizeAgentErrorKind(kind),
		Err:  err,
	}
}

func AgentErrorKindFromError(err error, fallback AgentErrorKind) AgentErrorKind {
	if err == nil {
		return AgentErrorUnknown
	}
	var turnErr *AgentTurnError
	if errors.As(err, &turnErr) && turnErr != nil {
		return normalizeAgentErrorKind(turnErr.Kind)
	}
	return normalizeAgentErrorKind(fallback)
}

func normalizeAgentErrorKind(kind AgentErrorKind) AgentErrorKind {
	switch kind {
	case AgentErrorProvider, AgentErrorTool, AgentErrorValidation, AgentErrorStartup:
		return kind
	default:
		return AgentErrorUnknown
	}
}
