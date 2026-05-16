package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type inTurnCompressionProvider struct {
	name          string
	responses     []string
	errByCall     map[int]error
	callCount     int
	histories     [][]api.Message
	responseID    string
	compactCalls  int
	cancelOnCall  map[int]context.CancelFunc
	waitForCancel map[int]bool
	canceledCalls []int
}

const inTurnCompressionCancelWaitTimeout = 2 * time.Second

func (p *inTurnCompressionProvider) Name() string {
	if p.name == "" {
		return "openai"
	}
	return p.name
}

func (p *inTurnCompressionProvider) SupportsImages() bool { return false }

func (p *inTurnCompressionProvider) IsFunctionCallingEnabled() bool { return true }

func (p *inTurnCompressionProvider) ChatWithTools(ctx context.Context, _ string, history []api.Message, _ string) (string, error) {
	call := p.recordChatCall(history)
	if err := p.handleCallContext(ctx, call); err != nil {
		return "", err
	}
	return p.responseForCall(call)
}

func (p *inTurnCompressionProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, _ string, _ *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

func (p *inTurnCompressionProvider) HasCachedResponseID() bool {
	return strings.TrimSpace(p.responseID) != ""
}

func (p *inTurnCompressionProvider) SetResponseID(id string) {
	p.responseID = id
}

func (p *inTurnCompressionProvider) GetResponseID() string {
	return p.responseID
}

func (p *inTurnCompressionProvider) SupportsCompact() bool { return true }

func (p *inTurnCompressionProvider) CompactHistory(context.Context, []api.InputItem, string, string) (*api.CompactResponse, error) {
	p.compactCalls++
	return &api.CompactResponse{Output: []api.InputItem{{Type: "compacted", Data: "should-not-be-used-in-turn"}}}, nil
}

func (p *inTurnCompressionProvider) recordChatCall(history []api.Message) int {
	call := p.callCount
	p.callCount++
	p.histories = append(p.histories, cloneMessagesForInTurnCompressionTest(history))
	return call
}

func (p *inTurnCompressionProvider) handleCallContext(ctx context.Context, call int) error {
	if cancel := p.cancelOnCall[call]; cancel != nil {
		cancel()
	}
	if p.waitForCancel[call] {
		select {
		case <-ctx.Done():
			p.canceledCalls = append(p.canceledCalls, call)
			return ctx.Err()
		case <-time.After(inTurnCompressionCancelWaitTimeout):
			return errors.New("timed out waiting for request context cancellation")
		}
	}
	select {
	case <-ctx.Done():
		p.canceledCalls = append(p.canceledCalls, call)
		return ctx.Err()
	default:
		return nil
	}
}

func (p *inTurnCompressionProvider) responseForCall(call int) (string, error) {
	if err := p.errByCall[call]; err != nil {
		return "", err
	}
	if call >= len(p.responses) {
		return p.responses[len(p.responses)-1], nil
	}
	return p.responses[call], nil
}

func (p *inTurnCompressionProvider) sawContextCancellationOnCall(call int) bool {
	for _, canceledCall := range p.canceledCalls {
		if canceledCall == call {
			return true
		}
	}
	return false
}

func (p *inTurnCompressionProvider) historyForCall(t *testing.T, call int) []api.Message {
	t.Helper()

	if call < 0 || call >= len(p.histories) {
		t.Fatalf("history for provider call %d missing; captured %d calls", call, len(p.histories))
	}
	return p.histories[call]
}

func cloneMessagesForInTurnCompressionTest(messages []api.Message) []api.Message {
	out := make([]api.Message, len(messages))
	copy(out, messages)
	for i := range out {
		out[i].ToolCalls = append([]api.OpenAIToolCall(nil), messages[i].ToolCalls...)
	}
	return out
}

func newInTurnCompressionConfig() *config.Config {
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesOff
	cfg.Compression.Enabled = true
	cfg.Compression.TokenThreshold = 1
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = true
	return cfg
}

func newInTurnCompressionTestAgent(t *testing.T, provider *inTurnCompressionProvider, out *bytes.Buffer) *Agent {
	t.Helper()

	return newInTurnCompressionTestAgentWithConfig(t, provider, newInTurnCompressionConfig(), out)
}

func newInTurnCompressionTestAgentWithConfig(t *testing.T, provider *inTurnCompressionProvider, cfg *config.Config, out *bytes.Buffer) *Agent {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	agent := newTurnRunnerTestAgent(provider, cfg, "", out, &loopTestTool{})
	t.Cleanup(agent.Cleanup)
	return agent
}

func seedInTurnCompressionOldHistory(agent *Agent) {
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("old context ", 100)},
		{Role: "assistant", Content: "old answer"},
	}
}

func seedInTurnCompressionHistory(agent *Agent, messages ...api.Message) {
	agent.History = append([]api.Message(nil), messages...)
}

func seedInTurnCompressionOldSession(agent *Agent) {
	seedInTurnCompressionOldHistory(agent)
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
}

func setInTurnCompressionResponseContext(agent *Agent, provider *inTurnCompressionProvider, responseID string) {
	provider.SetResponseID(responseID)
	agent.session.ResponseID = responseID
	agent.session.ResponseModel = agent.CurrentModel
	agent.session.ResponseProviderName = config.CanonicalProviderName(agent.ProviderName)
	agent.session.ResponseProviderConfigKey = agent.currentProviderConfigKey()
}

func TestNormalModeInTurnAutoCompressKeepsCurrentTurnTailBeforeNextModelCall(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			`{"tool":"loop_tool","args":{"iteration":"1"}}`,
			"in-turn summary",
			"final response",
		},
	}
	agent := newInTurnCompressionTestAgent(t, provider, &out)
	seedInTurnCompressionOldHistory(agent)
	setInTurnCompressionResponseContext(agent, provider, "resp_old")
	state := newAutoCompressionTurnState()

	if err := agent.runNormalModeWithAutoCompression(context.Background(), "current request", nil, state); err != nil {
		t.Fatalf("runNormalModeWithAutoCompression() error = %v", err)
	}

	if provider.callCount != 3 {
		t.Fatalf("ChatWithTools call count = %d, want 3", provider.callCount)
	}
	if provider.compactCalls != 0 {
		t.Fatalf("CompactHistory call count = %d, want 0 for in-turn compression", provider.compactCalls)
	}
	if !state.attemptedThisTurn() || !state.compressedThisTurn() {
		t.Fatalf("auto-compression state = attempted:%t compressed:%t, want both true", state.attemptedThisTurn(), state.compressedThisTurn())
	}
	if provider.GetResponseID() != "" || agent.session.ResponseID != "" {
		t.Fatalf("response context after compression = provider:%q session:%q, want both cleared", provider.GetResponseID(), agent.session.ResponseID)
	}

	nextHistory := provider.historyForCall(t, 2)
	if len(nextHistory) != 4 {
		t.Fatalf("next model call history len = %d, want summary + current user/assistant/tool", len(nextHistory))
	}
	if nextHistory[0].Role != "system" || !strings.Contains(nextHistory[0].Content, "in-turn summary") {
		t.Fatalf("next history first message = %#v, want summary system message", nextHistory[0])
	}
	if strings.Contains(nextHistory[0].Content, "old context") {
		t.Fatalf("summary message leaked old full context: %q", nextHistory[0].Content)
	}
	if nextHistory[1].Role != "user" || !strings.Contains(nextHistory[1].Content, "current request") {
		t.Fatalf("next history current user = %#v, want current request", nextHistory[1])
	}
	if nextHistory[2].Role != "assistant" || len(nextHistory[2].ToolCalls) != 1 {
		t.Fatalf("next history assistant tool call = %#v, want retained assistant ToolCalls", nextHistory[2])
	}
	if nextHistory[3].Role != "tool" || nextHistory[3].ToolCallID == "" || !strings.Contains(nextHistory[3].Content, "iteration=1") {
		t.Fatalf("next history tool result = %#v, want retained current tool result", nextHistory[3])
	}
}

func TestNormalModeInTurnAutoCompressPreservesKeepRecentBeforeCurrentTurn(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			`{"tool":"loop_tool","args":{"iteration":"1"}}`,
			"in-turn summary",
			"final response",
		},
	}
	cfg := newInTurnCompressionConfig()
	cfg.Compression.KeepRecent = 4
	agent := newInTurnCompressionTestAgentWithConfig(t, provider, cfg, &out)
	seedInTurnCompressionHistory(agent,
		api.Message{Role: "user", Content: strings.Repeat("old 0 ", 20)},
		api.Message{Role: "assistant", Content: strings.Repeat("old 1 ", 20)},
		api.Message{Role: "user", Content: strings.Repeat("old 2 ", 20)},
		api.Message{Role: "assistant", Content: strings.Repeat("old 3 ", 20)},
		api.Message{Role: "user", Content: "recent old 4"},
	)
	state := newAutoCompressionTurnState()

	if err := agent.runNormalModeWithAutoCompression(context.Background(), "current request", nil, state); err != nil {
		t.Fatalf("runNormalModeWithAutoCompression() error = %v", err)
	}

	nextHistory := provider.historyForCall(t, 2)
	if len(nextHistory) != 5 {
		t.Fatalf("next model call history len = %d, want summary + keep_recent old message + current user/assistant/tool", len(nextHistory))
	}
	if nextHistory[1].Role != "user" || nextHistory[1].Content != "recent old 4" {
		t.Fatalf("next history retained old tail = %#v, want keep_recent message before current turn", nextHistory[1])
	}
	if nextHistory[2].Role != "user" || !strings.Contains(nextHistory[2].Content, "current request") {
		t.Fatalf("next history current user = %#v, want current request after retained old tail", nextHistory[2])
	}
}

func TestNormalModeInTurnAutoCompressUsesRequestContextCancellation(t *testing.T) {
	disableColors(t)

	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			`{"tool":"loop_tool","args":{"iteration":"1"}}`,
			"",
			"final response",
		},
		cancelOnCall:  map[int]context.CancelFunc{1: cancel},
		waitForCancel: map[int]bool{1: true},
	}
	agent := newInTurnCompressionTestAgent(t, provider, &out)
	seedInTurnCompressionOldHistory(agent)
	state := newAutoCompressionTurnState()

	err := agent.runNormalModeWithAutoCompression(ctx, "current request", nil, state)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runNormalModeWithAutoCompression() error = %v, want context.Canceled", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("ChatWithTools call count = %d, want 2 without next provider request after canceled compression", provider.callCount)
	}
	if !provider.sawContextCancellationOnCall(1) {
		t.Fatalf("summary call did not observe request context cancellation; canceled calls = %#v", provider.canceledCalls)
	}
	if !state.attemptedThisTurn() || state.compressedThisTurn() {
		t.Fatalf("auto-compression state = attempted:%t compressed:%t, want canceled summary attempt without compression", state.attemptedThisTurn(), state.compressedThisTurn())
	}
}

func TestNormalModeInTurnAutoCompressFailureKeepsHistoryAndContinues(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			`{"tool":"loop_tool","args":{"iteration":"1"}}`,
			"",
			"final response",
		},
		errByCall: map[int]error{1: errors.New("summary failed")},
	}
	agent := newInTurnCompressionTestAgent(t, provider, &out)
	seedInTurnCompressionOldHistory(agent)
	setInTurnCompressionResponseContext(agent, provider, "resp_old")
	state := newAutoCompressionTurnState()

	if err := agent.runNormalModeWithAutoCompression(context.Background(), "current request", nil, state); err != nil {
		t.Fatalf("runNormalModeWithAutoCompression() error = %v", err)
	}

	if provider.callCount != 3 {
		t.Fatalf("ChatWithTools call count = %d, want 3", provider.callCount)
	}
	if !state.attemptedThisTurn() || state.compressedThisTurn() {
		t.Fatalf("auto-compression state = attempted:%t compressed:%t, want attempted failed", state.attemptedThisTurn(), state.compressedThisTurn())
	}
	if provider.GetResponseID() != "resp_old" || agent.session.ResponseID != "resp_old" {
		t.Fatalf("response context after failed compression = provider:%q session:%q, want restored resp_old", provider.GetResponseID(), agent.session.ResponseID)
	}

	nextHistory := provider.historyForCall(t, 2)
	if len(nextHistory) != 5 {
		t.Fatalf("next model call history len = %d, want original old history + current user/assistant/tool", len(nextHistory))
	}
	if nextHistory[0].Role != "user" || !strings.Contains(nextHistory[0].Content, "old context") {
		t.Fatalf("next history first message = %#v, want original old history retained", nextHistory[0])
	}
	if nextHistory[2].Role != "user" || !strings.Contains(nextHistory[2].Content, "current request") {
		t.Fatalf("next history current user = %#v, want current request retained", nextHistory[2])
	}
}

func TestChatCoreInTurnAutoCompressPersistsSessionTailAndSkipsPostTurn(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			`{"tool":"loop_tool","args":{"iteration":"1"}}`,
			"in-turn summary",
			"final response",
		},
	}
	agent := newInTurnCompressionTestAgent(t, provider, &out)
	seedInTurnCompressionOldSession(agent)

	if err := agent.chatCore("current request", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}

	if provider.callCount != 3 {
		t.Fatalf("ChatWithTools call count = %d, want 3 without post-turn retry", provider.callCount)
	}
	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	if loaded.ResponseID != "" {
		t.Fatalf("loaded.ResponseID = %q, want empty after local compression", loaded.ResponseID)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 5 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want summary + current user/assistant/tool/final", len(loadedMessages))
	}
	if loadedMessages[0].Role != "system" || !strings.Contains(loadedMessages[0].Content, "in-turn summary") {
		t.Fatalf("loaded first message = %#v, want compressed summary", loadedMessages[0])
	}
	if strings.Contains(loadedMessages[0].Content, "old context") {
		t.Fatalf("loaded summary leaked original full context: %q", loadedMessages[0].Content)
	}
	if loadedMessages[1].Role != "user" || loadedMessages[1].Content != "current request" {
		t.Fatalf("loaded current user = %#v, want raw current request without normal mode prompt", loadedMessages[1])
	}
	if loadedMessages[2].Role != "assistant" || len(loadedMessages[2].ToolCalls) != 1 {
		t.Fatalf("loaded assistant tool call = %#v, want retained assistant ToolCalls", loadedMessages[2])
	}
	if loadedMessages[3].Role != "tool" || loadedMessages[3].ToolCallID == "" || !strings.Contains(loadedMessages[3].Content, "iteration=1") {
		t.Fatalf("loaded tool result = %#v, want retained current tool result", loadedMessages[3])
	}
	if loadedMessages[4].Role != "assistant" || loadedMessages[4].Content != "final response" {
		t.Fatalf("loaded final response = %#v, want final assistant response", loadedMessages[4])
	}
}

func TestChatCoreInTurnAutoCompressFailureSkipsPostTurnRetry(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &inTurnCompressionProvider{
		name: "openai",
		responses: []string{
			`{"tool":"loop_tool","args":{"iteration":"1"}}`,
			"",
			"final response",
			"unexpected post-turn summary",
		},
		errByCall: map[int]error{1: errors.New("summary failed")},
	}
	agent := newInTurnCompressionTestAgent(t, provider, &out)
	seedInTurnCompressionOldHistory(agent)
	setInTurnCompressionResponseContext(agent, provider, "resp_old")

	if err := agent.chatCore("current request", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}

	if provider.callCount != 3 {
		t.Fatalf("ChatWithTools call count = %d, want 3 with no post-turn compression retry", provider.callCount)
	}
	if provider.GetResponseID() != "resp_old" || agent.session.ResponseID != "resp_old" {
		t.Fatalf("response context after failed compression = provider:%q session:%q, want restored resp_old", provider.GetResponseID(), agent.session.ResponseID)
	}
}
