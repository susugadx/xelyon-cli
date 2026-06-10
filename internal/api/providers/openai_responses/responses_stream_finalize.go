package openairesponses

import "github.com/susugadx/xelyon-cli/internal/api"

type responsesStreamFinalizePolicy struct {
	state         *responsesStreamState
	usageCallback api.UsageCallback
}

func newResponsesStreamFinalizePolicy(state *responsesStreamState, usageCallback api.UsageCallback) responsesStreamFinalizePolicy {
	return responsesStreamFinalizePolicy{
		state:         state,
		usageCallback: usageCallback,
	}
}

func (p responsesStreamFinalizePolicy) finalize(content string, parseErr error) (string, string, error) {
	if parseErr != nil {
		return "", p.state.responseID, parseErr
	}

	p.emitUsage()
	return p.composeContent(content), p.state.responseID, nil
}

func (p responsesStreamFinalizePolicy) emitUsage() {
	if p.state.lastUsage == nil || p.usageCallback == nil {
		return
	}
	p.usageCallback(*p.state.lastUsage)
}

func (p responsesStreamFinalizePolicy) composeContent(content string) string {
	if p.state.toolCallsOut.Len() == 0 {
		return content
	}
	toolCalls := p.state.toolCallsOut.String()
	if content != "" {
		return content + toolCalls
	}
	return toolCalls
}
