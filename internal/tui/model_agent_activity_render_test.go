package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

func TestModel_AgentActivitySpinnerTickReplacesTrackedBlock(t *testing.T) {
	agent := &stubAgent{
		statusLine: "processing",
		processing: true,
		statusSnapshot: StatusSnapshot{
			Provider:   "openai",
			Model:      "gpt-5.4",
			Tokens:     "12.3k",
			Cost:       "~$0.123",
			LegacyLine: "processing",
		},
	}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()
	m.agentActivity.startedAt = time.Now().Add(-65 * time.Second)
	initialLen := len(m.rawLines)
	blockStart := m.agentActivity.block.lineStart

	updated, _ := m.Update(spinner.TickMsg{})
	m = updated.(Model)

	if len(m.rawLines) != initialLen {
		t.Fatalf("spinner tick should replace agent block, rawLines len = %d, want %d", len(m.rawLines), initialLen)
	}
	if m.agentActivity.block.lineStart != blockStart {
		t.Fatalf("agent block start = %d, want %d", m.agentActivity.block.lineStart, blockStart)
	}
	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · working · 01:05 ──", "openai/gpt-5.4", "12.3k tok", "~$0.123"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("agent activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
}

func TestModel_AgentDoneWithErrorBlocksActivity(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()

	updated, _, handled := m.handleStreamMessage(AgentDoneMsg{
		Error:     errors.New("network down"),
		ErrorKind: AgentErrorProvider,
	})
	if !handled {
		t.Fatal("AgentDoneMsg should be handled")
	}
	m = updated

	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · blocked ·", "✕ [provider error] network down", "! check provider/network and retry"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("blocked activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
}

func TestModel_AgentDoneWithValidationErrorUsesValidationLabel(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.beginAgentActivity()

	updated, _, handled := m.handleStreamMessage(AgentDoneMsg{
		Error: WrapAgentTurnError(AgentErrorValidation, errors.New("missing image file")),
	})
	if !handled {
		t.Fatal("AgentDoneMsg should be handled")
	}
	m = updated

	plain := plainRawTranscript(m)
	for _, fragment := range []string{"── agent · blocked ·", "✕ [validation error] missing image file", "! fix the input and retry"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("validation activity missing %q, transcript:\n%s", fragment, plain)
		}
	}
}
