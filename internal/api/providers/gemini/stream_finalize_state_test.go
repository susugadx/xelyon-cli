package gemini

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestSSEFinalizeState_BuildOutput_NoContentReturnsError(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	interpret := newSSEInterpretState(ctx, nil, "", false)
	state := newSSEFinalizeState(interpret)

	_, err := state.buildOutput()
	if err == nil {
		t.Fatal("buildOutput() error = nil, want no content error")
	}
	if !strings.Contains(err.Error(), "no content in Gemini SSE response") {
		t.Fatalf("buildOutput() error = %q, want no content message", err.Error())
	}
}

func TestSSEFinalizeState_BuildOutput_RescueDoesNotEmitWarning(t *testing.T) {
	ctx, _, errOut := newGeminiResponseContext()
	interpret := newSSEInterpretState(ctx, nil, "", false)
	interpret.rescuedToolJSONs = []string{`{"tool":"read_file","args":{"path":"/tmp/demo.txt"}}`}
	state := newSSEFinalizeState(interpret)

	result, err := state.buildOutput()
	if err != nil {
		t.Fatalf("buildOutput() error = %v", err)
	}
	if result.rescuedToolCallCount != 1 {
		t.Fatalf("buildOutput() rescuedToolCallCount = %d, want 1", result.rescuedToolCallCount)
	}
	if !strings.Contains(result.response, `"tool":"read_file"`) {
		t.Fatalf("buildOutput() response = %q, want rescued tool JSON", result.response)
	}
	if strings.Contains(errOut.String(), "FC rescue") {
		t.Fatalf("buildOutput() should not emit warning, errOut = %q", errOut.String())
	}
}

func TestSSEFinalizeState_EmitFinalizeEffects_EmitsUsageAndWarning(t *testing.T) {
	ctx, _, errOut := newGeminiResponseContext()
	interpret := newSSEInterpretState(ctx, nil, "", false)
	interpret.usage = &GeminiUsageMetadata{
		PromptTokenCount:        11,
		CandidatesTokenCount:    5,
		ThoughtsTokenCount:      2,
		CachedContentTokenCount: 3,
	}
	state := newSSEFinalizeState(interpret)

	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		gotUsage = u
	})

	state.emitFinalizeEffects(p, 2)
	if !strings.Contains(errOut.String(), "FC rescue: 2 tool call(s)") {
		t.Fatalf("emitFinalizeEffects() errOut = %q, want rescue warning", errOut.String())
	}
	if gotUsage.InputTokens != 11 || gotUsage.OutputTokens != 5 || gotUsage.ThinkingTokens != 2 || gotUsage.CachedInputTokens != 3 {
		t.Fatalf("emitFinalizeEffects() usage = %+v, want prompt=11 output=5 thinking=2 cached=3", gotUsage)
	}
}
