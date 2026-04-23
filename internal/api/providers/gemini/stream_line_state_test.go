package gemini

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSSEInterpretState_ProcessLine_IgnoresNonDataLine(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", false)
	thinkingTimer := time.NewTimer(time.Second)
	defer thinkingTimer.Stop()

	processed := state.processLine(ctx, "event: message", thinkingTimer, time.Second)
	if processed {
		t.Fatal("processLine() = true, want false for non-data line")
	}
	if got := state.response(); got != "" {
		t.Fatalf("response = %q, want empty", got)
	}
}

func TestSSEInterpretState_ProcessLine_DecodeErrorIsSkipped(t *testing.T) {
	ctx, _, errOut := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", true)
	thinkingTimer := time.NewTimer(time.Second)
	defer thinkingTimer.Stop()

	processed := state.processLine(ctx, "data: {broken json", thinkingTimer, time.Second)
	if processed {
		t.Fatal("processLine() = true, want false for decode error")
	}
	if !strings.Contains(errOut.String(), "Failed to unmarshal chunk") {
		t.Fatalf("errOut = %q, want decode debug log", errOut.String())
	}
}

func TestSSEInterpretState_ProcessLine_AppliesValidChunk(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", false)
	thinkingTimer := time.NewTimer(time.Second)
	defer thinkingTimer.Stop()

	chunk := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{{
			Content: GeminiFunctionContent{
				Parts: []GeminiFunctionPart{{Text: "hello"}},
			},
		}},
	}
	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	processed := state.processLine(ctx, "data: "+string(data), thinkingTimer, time.Second)
	if !processed {
		t.Fatal("processLine() = false, want true for valid chunk")
	}
	if got := state.response(); got != "hello" {
		t.Fatalf("response = %q, want %q", got, "hello")
	}
}
