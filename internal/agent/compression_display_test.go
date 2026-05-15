package agent

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	displayui "github.com/susugadx/xelyon-cli/internal/ui"
)

func TestCompressHistory_TUIUsesStructuredCompressionDisplay(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "secret summary content"}
	agent, out := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.tuiToolResultCh = make(chan tools.ToolResultInfo, 4)
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "older"},
		{Role: "user", Content: "latest"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	events := readCompressionEvents(t, agent.tuiToolResultCh, 2)
	running, done := events[0], events[1]
	if running.ToolName != compressionDisplayToolName || running.Status != tools.ToolStatusRunning {
		t.Fatalf("running event = %#v, want compress running", running)
	}
	if running.Args[displayui.ToolArgCompressionMode] != compressionDisplayModeHistory ||
		running.Args[displayui.ToolArgCompressionReason] != compressionDisplayReasonManual {
		t.Fatalf("running args = %#v, want history/manual", running.Args)
	}
	if done.ToolName != compressionDisplayToolName || done.Status != tools.ToolStatusOK || done.Error {
		t.Fatalf("done event = %#v, want compress ok", done)
	}
	if done.ID == "" || done.ID != running.ID {
		t.Fatalf("event IDs = running:%q done:%q, want same non-empty ID", running.ID, done.ID)
	}
	if strings.Contains(done.Result, "secret summary content") {
		t.Fatalf("TUI compression detail leaked summary: %q", done.Result)
	}
	if output := out.String(); strings.Contains(output, "Compressing history") || strings.Contains(output, "Before:") {
		t.Fatalf("TUI compression should not write legacy progress output, got:\n%s", output)
	}
}

func TestCompressHistory_TUISplitErrorDoesNotLeaveRunningCompressionDisplay(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "unused summary"}
	agent, out := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.tuiToolResultCh = make(chan tools.ToolResultInfo, 4)
	agent.History = []api.Message{
		{
			Role: "assistant",
			ToolCalls: []api.OpenAIToolCall{{
				ID: "call_1",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"a.go"}`,
				},
			}},
		},
		{Role: "tool", Content: "file content", ToolCallID: "call_1"},
	}

	err := agent.CompressHistory(1)
	if err == nil || !strings.Contains(err.Error(), "FC ターン保護") {
		t.Fatalf("CompressHistory() error = %v, want FC turn protection split error", err)
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools calls = %d, want 0 before compression can start", provider.chatCalls)
	}
	assertNoCompressionEvents(t, agent.tuiToolResultCh)
	if output := out.String(); strings.Contains(output, "Compressing history") {
		t.Fatalf("split error should not print compression progress, got:\n%s", output)
	}
}

func TestRunAutoCompression_TUICompactFallbackKeepsCompressionRowRunning(t *testing.T) {
	provider := &compressionTestProvider{
		name:            "openai",
		supportsCompact: true,
		compactErr:      errors.New("compact unavailable"),
		summary:         "fallback summary",
	}
	cfg := config.DefaultConfig()
	cfg.Compression.PreferCompactAPI = true
	cfg.Compression.KeepRecent = 1
	agent, out := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.tuiToolResultCh = make(chan tools.ToolResultInfo, 8)
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "older"},
		{Role: "user", Content: "latest"},
	}

	if !agent.runAutoCompression(autoCompressionDecision{
		action:        autoCompressionActionRun,
		reason:        autoCompressionReasonTokenThreshold,
		currentTokens: agent.EstimateTokens(),
	}) {
		t.Fatal("runAutoCompression() = false, want true after history fallback")
	}

	events := readCompressionEvents(t, agent.tuiToolResultCh, 3)
	if events[0].Status != tools.ToolStatusRunning ||
		events[0].Args[displayui.ToolArgCompressionMode] != compressionDisplayModeCompactAPI {
		t.Fatalf("first event = %#v, want compact running", events[0])
	}
	if events[1].Status != tools.ToolStatusRunning ||
		events[1].Args[displayui.ToolArgCompressionMode] != compressionDisplayModeHistory {
		t.Fatalf("fallback event = %#v, want history running", events[1])
	}
	if events[2].Status != tools.ToolStatusOK || events[2].Error {
		t.Fatalf("final event = %#v, want ok", events[2])
	}
	for _, event := range events {
		if event.Status == tools.ToolStatusError || event.Error {
			t.Fatalf("fallback path should not emit an intermediate error event: %#v", event)
		}
		if event.ID != events[0].ID {
			t.Fatalf("event ID = %q, want %q", event.ID, events[0].ID)
		}
	}
	if output := out.String(); strings.Contains(output, "Compact API failed") || strings.Contains(output, "fallback summary") {
		t.Fatalf("TUI auto-compression should not write legacy fallback output, got:\n%s", output)
	}
}

func TestRunAutoCompression_TUISkipDoesNotReportAfterTokens(t *testing.T) {
	provider := &compressionTestProvider{name: "openai"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 5
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.tuiToolResultCh = make(chan tools.ToolResultInfo, 4)
	agent.History = []api.Message{
		{Role: "user", Content: "short"},
		{Role: "assistant", Content: "history"},
	}

	if agent.runAutoCompression(autoCompressionDecision{
		action:        autoCompressionActionRun,
		reason:        autoCompressionReasonTokenThreshold,
		currentTokens: agent.EstimateTokens(),
	}) {
		t.Fatal("runAutoCompression() = true, want false when history is too short")
	}

	events := readCompressionEvents(t, agent.tuiToolResultCh, 2)
	skipped := events[1]
	if skipped.Status != tools.ToolStatusOK ||
		skipped.Args[displayui.ToolArgCompressionOutcome] != "history too short" {
		t.Fatalf("skipped event = %#v, want ok history too short", skipped)
	}
	if _, ok := skipped.Args[displayui.ToolArgCompressionAfterTokens]; ok {
		t.Fatalf("skipped event after_tokens = %q, want absent", skipped.Args[displayui.ToolArgCompressionAfterTokens])
	}
}

func TestHandleCompressCommand_TUISuppressesLegacyCommandOutput(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "secret summary content"}
	agent, out := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.tuiToolResultCh = make(chan tools.ToolResultInfo, 4)
	agent.ui().SetPrompter(&promptConfirmTestTUIPrompter{})
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "older"},
		{Role: "user", Content: "latest"},
	}

	if !handleCompressCommand(agent, []string{"1"}) {
		t.Fatal("handleCompressCommand() = false, want true")
	}

	events := readCompressionEvents(t, agent.tuiToolResultCh, 2)
	if events[0].Status != tools.ToolStatusRunning || events[1].Status != tools.ToolStatusOK {
		t.Fatalf("compression events = %#v", events)
	}
	output := out.String()
	for _, legacy := range []string{"Compress History", "現在の履歴", "Warning:", "Compressing history", "secret summary content"} {
		if strings.Contains(output, legacy) {
			t.Fatalf("TUI /compress should not write legacy output %q, got:\n%s", legacy, output)
		}
	}
}

func readCompressionEvents(t *testing.T, ch <-chan tools.ToolResultInfo, count int) []tools.ToolResultInfo {
	t.Helper()

	events := make([]tools.ToolResultInfo, 0, count)
	for len(events) < count {
		select {
		case event := <-ch:
			events = append(events, event)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for compression event %d/%d", len(events)+1, count)
		}
	}
	return events
}

func assertNoCompressionEvents(t *testing.T, ch <-chan tools.ToolResultInfo) {
	t.Helper()

	select {
	case event := <-ch:
		t.Fatalf("unexpected compression event = %#v", event)
	default:
	}
}
