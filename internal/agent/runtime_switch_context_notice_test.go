package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestRuntimeSwitchContextNotice_WarnsForLargeKeptContext(t *testing.T) {
	agent := &Agent{
		CurrentModel: "gpt-5.4",
		SystemPrompt: "system",
		History: []api.Message{
			{Role: "user", Content: strings.Repeat("large context token ", 80_000)},
		},
		Runtime: NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
	}

	notice := agent.runtimeSwitchContextNotice(true)

	if !notice.LocalContextKept || !notice.ResponseContinuationReset {
		t.Fatalf("notice = %#v, want kept context with response continuation reset", notice)
	}
	if !notice.ContextResendWarning {
		t.Fatalf("ContextResendWarning = false, want true for %d estimated tokens", notice.EstimatedNextInputTokens)
	}
}

func TestRuntimeSwitchContextNotice_DoesNotWarnForSmallKeptContext(t *testing.T) {
	agent := &Agent{
		CurrentModel: "gpt-5.4",
		SystemPrompt: "system",
		History: []api.Message{
			{Role: "user", Content: "small context"},
		},
		Runtime: NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
	}

	notice := agent.runtimeSwitchContextNotice(true)

	if !notice.LocalContextKept || !notice.ResponseContinuationReset {
		t.Fatalf("notice = %#v, want kept context with response continuation reset", notice)
	}
	if notice.ContextResendWarning {
		t.Fatalf("ContextResendWarning = true, want false for %d estimated tokens", notice.EstimatedNextInputTokens)
	}
}
