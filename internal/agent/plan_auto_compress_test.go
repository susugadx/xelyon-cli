package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	planAutoCompressionSummary                = "plan-mode pre-turn summary"
	planAutoCompressionImplementationResponse = "implementation done"
)

func planAutoCompressionToolResponse(path string) string {
	return fmt.Sprintf(`{"tool": "read_file", "args": {"paths": [%q]}}`, path)
}

func planAutoCompressionApprovedPlanResponse() string {
	return planHandoffTestApprovedPlanResponse(planHandoffTestApprovedStep)
}

func newPlanAutoCompressionTestAgent(t *testing.T, provider *inTurnCompressionProvider, input string, cfg *config.Config, out *bytes.Buffer) *Agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(input), out, out)
	runtime.Registry = tools.DefaultRegistry.Clone()

	agent := NewAgentWithRuntime("gpt-5.4", provider, false, runtime)
	agent.setAutoApprove(true)
	t.Cleanup(agent.Cleanup)
	return agent
}

func seedPlanAutoCompressionSession(agent *Agent, messages ...api.Message) {
	agent.History = append([]api.Message(nil), messages...)
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
}

func loadPlanAutoCompressionMessages(t *testing.T, agent *Agent) []api.Message {
	t.Helper()

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	return loaded.ToAPIMessages()
}

func assertPlanAutoCompressionSummary(t *testing.T, msg api.Message, context string) {
	t.Helper()

	if msg.Role != "assistant" || !strings.Contains(msg.Content, planAutoCompressionSummary) {
		t.Fatalf("%s first message = %#v, want compressed pre-plan summary", context, msg)
	}
	if strings.Contains(msg.Content, "old context") {
		t.Fatalf("%s summary leaked original old context: %q", context, msg.Content)
	}
}

func assertPlanAutoCompressionRetainedRecent(t *testing.T, msg api.Message, context string) {
	t.Helper()

	if msg.Content != "recent pre-plan context" {
		t.Fatalf("%s retained old tail = %#v, want keep_recent pre-plan message", context, msg)
	}
}

func TestPlanModeInTurnAutoCompressApprovalRestoresCompressedPrePlanHistory(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			planAutoCompressionToolResponse("missing.txt"),
			planAutoCompressionSummary,
			planAutoCompressionApprovedPlanResponse(),
			planAutoCompressionImplementationResponse,
		},
	}
	cfg := newInTurnCompressionConfig()
	cfg.Compression.KeepRecent = 4
	agent := newPlanAutoCompressionTestAgent(t, provider, "1\n", cfg, &out)
	seedPlanAutoCompressionSession(agent,
		api.Message{Role: "user", Content: strings.Repeat("old context 0 ", 20)},
		api.Message{Role: "assistant", Content: strings.Repeat("old context 1 ", 20)},
		api.Message{Role: "user", Content: strings.Repeat("old context 2 ", 20)},
		api.Message{Role: "assistant", Content: strings.Repeat("old context 3 ", 20)},
		api.Message{Role: "user", Content: "recent pre-plan context"},
	)
	setInTurnCompressionResponseContext(agent, provider, "resp_old")
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}

	if provider.callCount != 4 {
		t.Fatalf("ChatWithTools call count = %d, want planning tool + summary + plan + implementation", provider.callCount)
	}
	if provider.compactCalls != 0 {
		t.Fatalf("CompactHistory call count = %d, want 0 for plan in-turn compression", provider.compactCalls)
	}

	planFollowupHistory := provider.historyForCall(t, 2)
	if len(planFollowupHistory) != 5 {
		t.Fatalf("plan followup history len = %d, want summary + recent old + investigation user/assistant/tool", len(planFollowupHistory))
	}
	assertPlanAutoCompressionSummary(t, planFollowupHistory[0], "plan followup")
	assertPlanAutoCompressionRetainedRecent(t, planFollowupHistory[1], "plan followup")

	implementationHistory := provider.historyForCall(t, 3)
	if len(implementationHistory) != 3 {
		t.Fatalf("implementation history len = %d, want summary + recent old + approved plan handoff", len(implementationHistory))
	}
	assertPlanAutoCompressionSummary(t, implementationHistory[0], "implementation")
	assertPlanAutoCompressionRetainedRecent(t, implementationHistory[1], "implementation")
	assertImplementationHistoryContainsApprovedPlanHandoff(t, implementationHistory)
	assertImplementationHistoryOmitsInvestigationContext(t, implementationHistory)

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	if loaded.ResponseID != "" || provider.GetResponseID() != "" {
		t.Fatalf("response context after approved implementation = session:%q provider:%q, want empty", loaded.ResponseID, provider.GetResponseID())
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 4 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want summary + recent old + handoff + implementation", len(loadedMessages))
	}
	assertPlanAutoCompressionSummary(t, loadedMessages[0], "loaded")
	assertPlanAutoCompressionRetainedRecent(t, loadedMessages[1], "loaded")
	if strings.Contains(loadedMessages[2].Content, "Investigation Phase") {
		t.Fatalf("loaded handoff leaked investigation prompt: %q", loadedMessages[2].Content)
	}
}

func TestPlanModeInTurnAutoCompressInvestigationOnlyPersistsOriginalRequest(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			planAutoCompressionToolResponse("missing.txt"),
			planAutoCompressionSummary,
			"Investigation complete. No code changes needed.",
		},
	}
	cfg := newInTurnCompressionConfig()
	cfg.Compression.KeepRecent = 1
	agent := newPlanAutoCompressionTestAgent(t, provider, "", cfg, &out)
	seedPlanAutoCompressionSession(agent,
		api.Message{Role: "user", Content: strings.Repeat("old context user ", 20)},
		api.Message{Role: "assistant", Content: "old context assistant"},
	)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("investigate feature", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}

	if provider.callCount != 3 {
		t.Fatalf("ChatWithTools call count = %d, want investigation tool + summary + final investigation response", provider.callCount)
	}
	followupHistory := provider.historyForCall(t, 2)
	if len(followupHistory) != 4 {
		t.Fatalf("followup history len = %d, want summary + investigation user/assistant/tool", len(followupHistory))
	}
	if !strings.Contains(followupHistory[1].Content, "You are in PLAN MODE (Investigation Phase)") {
		t.Fatalf("followup runtime user = %q, want investigation prompt for model context", followupHistory[1].Content)
	}

	loadedMessages := loadPlanAutoCompressionMessages(t, agent)
	if len(loadedMessages) != 5 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want summary + original user + assistant/tool/final", len(loadedMessages))
	}
	assertPlanAutoCompressionSummary(t, loadedMessages[0], "loaded")
	if loadedMessages[1].Role != "user" || loadedMessages[1].Content != "investigate feature" {
		t.Fatalf("loaded current user = %#v, want original request instead of investigation prompt", loadedMessages[1])
	}
	if strings.Contains(loadedMessages[1].Content, "PLAN MODE") {
		t.Fatalf("loaded current user leaked investigation prompt: %q", loadedMessages[1].Content)
	}
	if loadedMessages[4].Role != "assistant" || loadedMessages[4].Content != "Investigation complete. No code changes needed." {
		t.Fatalf("loaded final response = %#v, want investigation-only assistant response", loadedMessages[4])
	}
}

func TestPlanModeInTurnAutoCompressSkipsWithoutPrePlanHistory(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			planAutoCompressionToolResponse("missing.txt"),
			"Investigation complete. No code changes needed.",
			"unexpected summary call",
		},
	}
	agent := newPlanAutoCompressionTestAgent(t, provider, "", newInTurnCompressionConfig(), &out)
	state := newAutoCompressionTurnState()

	handoff, err := agent.runPlanModeWithAutoCompression(context.Background(), "investigate feature", state)
	if err != nil {
		t.Fatalf("runPlanModeWithAutoCompression() error = %v", err)
	}
	if handoff != nil {
		t.Fatalf("runPlanModeWithAutoCompression() handoff = %#v, want nil for investigation-only response", handoff)
	}

	if provider.callCount != 2 {
		t.Fatalf("ChatWithTools call count = %d, want investigation tool + final response without pre-plan compression", provider.callCount)
	}
	if state.attemptedThisTurn() || state.compressedThisTurn() {
		t.Fatalf("auto-compression state = attempted:%t compressed:%t, want no attempt without pre-plan history", state.attemptedThisTurn(), state.compressedThisTurn())
	}
	followupHistory := provider.historyForCall(t, 1)
	if len(followupHistory) != 3 {
		t.Fatalf("followup history len = %d, want investigation user/assistant/tool", len(followupHistory))
	}
	if followupHistory[0].Role != "user" || !strings.Contains(followupHistory[0].Content, "You are in PLAN MODE (Investigation Phase)") {
		t.Fatalf("followup current user = %#v, want retained investigation prompt", followupHistory[0])
	}
	for _, msg := range followupHistory {
		if strings.Contains(msg.Content, planAutoCompressionSummary) {
			t.Fatalf("followup history was compressed without pre-plan history: %#v", followupHistory)
		}
	}
}

func TestPlanModeChatOnceSkipsInTurnAutoCompression(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			planAutoCompressionToolResponse("missing.txt"),
			"Investigation complete. No code changes needed.",
			"unexpected extra model call",
		},
	}
	cfg := newInTurnCompressionConfig()
	cfg.Compression.KeepRecent = 1
	agent := newPlanAutoCompressionTestAgent(t, provider, "", cfg, &out)
	seedPlanAutoCompressionSession(agent,
		api.Message{Role: "user", Content: strings.Repeat("old context user ", 20)},
		api.Message{Role: "assistant", Content: "old context assistant"},
	)
	agent.PlanModeEnabled = true

	if err := agent.ChatOnce("investigate feature"); err != nil {
		t.Fatalf("ChatOnce() error = %v", err)
	}

	if provider.callCount != 2 {
		t.Fatalf("ChatWithTools call count = %d, want investigation tool + final response without one-shot compression", provider.callCount)
	}
	if provider.compactCalls != 0 {
		t.Fatalf("CompactHistory call count = %d, want 0 for one-shot plan mode", provider.compactCalls)
	}
	followupHistory := provider.historyForCall(t, 1)
	if len(followupHistory) != 5 {
		t.Fatalf("followup history len = %d, want original old history + investigation user/assistant/tool", len(followupHistory))
	}
	if !strings.Contains(followupHistory[0].Content, "old context user") {
		t.Fatalf("followup first message = %#v, want original old history without summary", followupHistory[0])
	}

	loadedMessages := loadPlanAutoCompressionMessages(t, agent)
	if len(loadedMessages) != 6 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want old history + original user + assistant/tool/final", len(loadedMessages))
	}
	if loadedMessages[0].Role == "assistant" && strings.Contains(loadedMessages[0].Content, planAutoCompressionSummary) {
		t.Fatalf("loaded history was compressed during one-shot plan mode: %#v", loadedMessages[0])
	}
	if !strings.Contains(loadedMessages[0].Content, "old context user") {
		t.Fatalf("loaded first message = %#v, want original old history", loadedMessages[0])
	}
}

func TestPlanModeInTurnAutoCompressFailureKeepsOldHistoryAndSkipsPostTurnRetry(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			planAutoCompressionToolResponse("missing.txt"),
			"",
			planAutoCompressionApprovedPlanResponse(),
			planAutoCompressionImplementationResponse,
			"unexpected post-turn summary",
		},
		errByCall: map[int]error{1: errors.New("summary failed")},
	}
	cfg := newInTurnCompressionConfig()
	cfg.Compression.KeepRecent = 1
	agent := newPlanAutoCompressionTestAgent(t, provider, "1\n", cfg, &out)
	seedPlanAutoCompressionSession(agent,
		api.Message{Role: "user", Content: strings.Repeat("old context user ", 20)},
		api.Message{Role: "assistant", Content: "old context assistant"},
	)
	setInTurnCompressionResponseContext(agent, provider, "resp_old")
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}

	if provider.callCount != 4 {
		t.Fatalf("ChatWithTools call count = %d, want no post-turn compression retry after failed plan in-turn compression", provider.callCount)
	}
	planFollowupHistory := provider.historyForCall(t, 2)
	if len(planFollowupHistory) != 5 {
		t.Fatalf("plan followup history len = %d, want original old history + investigation user/assistant/tool", len(planFollowupHistory))
	}
	if !strings.Contains(planFollowupHistory[0].Content, "old context user") {
		t.Fatalf("plan followup first message = %#v, want original old history retained after failed compression", planFollowupHistory[0])
	}

	implementationHistory := provider.historyForCall(t, 3)
	if len(implementationHistory) != 3 {
		t.Fatalf("implementation history len = %d, want original old history + approved plan handoff", len(implementationHistory))
	}
	if !strings.Contains(implementationHistory[0].Content, "old context user") || implementationHistory[1].Content != "old context assistant" {
		t.Fatalf("implementation history old prefix = %#v, want original old history", implementationHistory[:2])
	}
	assertImplementationHistoryContainsApprovedPlanHandoff(t, implementationHistory)

	loadedMessages := loadPlanAutoCompressionMessages(t, agent)
	if len(loadedMessages) != 4 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want original old history + handoff + implementation", len(loadedMessages))
	}
	if !strings.Contains(loadedMessages[0].Content, "old context user") || loadedMessages[1].Content != "old context assistant" {
		t.Fatalf("loaded old prefix = %#v, want original old history after failed compression", loadedMessages[:2])
	}
}

func TestPlanModeInTurnAutoCompressPropagatesCancellation(t *testing.T) {
	disableColors(t)

	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			planAutoCompressionToolResponse("missing.txt"),
			"",
			planAutoCompressionApprovedPlanResponse(),
		},
		cancelOnCall:  map[int]context.CancelFunc{1: cancel},
		waitForCancel: map[int]bool{1: true},
	}
	agent := newPlanAutoCompressionTestAgent(t, provider, "", newInTurnCompressionConfig(), &out)
	seedPlanAutoCompressionSession(agent,
		api.Message{Role: "user", Content: strings.Repeat("old context ", 20)},
		api.Message{Role: "assistant", Content: "old answer"},
	)
	state := newAutoCompressionTurnState()

	_, err := agent.runPlanModeWithAutoCompression(ctx, "implement feature", state)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runPlanModeWithAutoCompression() error = %v, want context.Canceled", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("ChatWithTools call count = %d, want investigation request + canceled summary only", provider.callCount)
	}
	if !provider.sawContextCancellationOnCall(1) {
		t.Fatalf("summary call did not observe request context cancellation; canceled calls = %#v", provider.canceledCalls)
	}
	if !state.attemptedThisTurn() || state.compressedThisTurn() {
		t.Fatalf("auto-compression state = attempted:%t compressed:%t, want canceled summary attempt without compression", state.attemptedThisTurn(), state.compressedThisTurn())
	}
}
