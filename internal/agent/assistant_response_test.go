package agent

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrepareAssistantResponse_StripsCompactionForDisplay(t *testing.T) {
	prepared := prepareAssistantResponse("before [COMPACTION]hidden[/COMPACTION] after")

	if prepared.raw != "before [COMPACTION]hidden[/COMPACTION] after" {
		t.Fatalf("raw = %q", prepared.raw)
	}
	if prepared.display != "before  after" {
		t.Fatalf("display = %q, want %q", prepared.display, "before  after")
	}
	if !prepared.hasCompactionNotice {
		t.Fatal("hasCompactionNotice = false, want true")
	}
}

func TestAppendAssistantResponseHistory_SeparatesRawAndDisplayState(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockProvider{name: "test"}, &out)
	agent.Stats = NewSessionStats("test")

	prepared := prepareAssistantResponse("before [COMPACTION]hidden[/COMPACTION] after")
	agent.emitAssistantResponseNotices(prepared)
	agent.appendAssistantResponseHistory(prepared)

	if got := agent.History[len(agent.History)-1].Content; got != prepared.raw {
		t.Fatalf("history content = %q, want %q", got, prepared.raw)
	}
	if got := agent.lastOutputs[len(agent.lastOutputs)-1]; got != prepared.display {
		t.Fatalf("last output = %q, want %q", got, prepared.display)
	}
	if agent.Stats.AssistantMessages != 1 {
		t.Fatalf("AssistantMessages = %d, want 1", agent.Stats.AssistantMessages)
	}
	if !strings.Contains(out.String(), "Context compacted by Claude") {
		t.Fatalf("expected compaction notice, got %q", out.String())
	}
}

func TestShowAssistantResponse_DisplaysWithoutRecordingHistory(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockProvider{name: "test"}, &out)

	agent.showAssistantResponse("before [COMPACTION]hidden[/COMPACTION] after")

	if len(agent.History) != 0 {
		t.Fatalf("History length = %d, want 0", len(agent.History))
	}
	if len(agent.lastOutputs) != 0 {
		t.Fatalf("lastOutputs length = %d, want 0", len(agent.lastOutputs))
	}
	if !strings.Contains(out.String(), "Context compacted by Claude") {
		t.Fatalf("expected compaction notice, got %q", out.String())
	}
	if !strings.Contains(out.String(), "💬 before  after") {
		t.Fatalf("expected final response output, got %q", out.String())
	}
}
