package agent

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type scriptedChatProvider struct {
	name            string
	functionCalling bool
	callCount       int
	usageCallback   api.UsageCallback
	chatWithToolsFn func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error)
}

func (p *scriptedChatProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "openai"
}

func (p *scriptedChatProvider) SupportsImages() bool { return false }

func (p *scriptedChatProvider) IsFunctionCallingEnabled() bool { return p.functionCalling }

func (p *scriptedChatProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

func (p *scriptedChatProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	call := p.callCount
	p.callCount++
	if p.chatWithToolsFn != nil {
		response, err := p.chatWithToolsFn(call, ctx, systemPrompt, history, model)
		if err != nil {
			return "", err
		}
		return compressionSummaryResponseForHistory(history, response), nil
	}
	return compressionSummaryResponseForHistory(history, "done"), nil
}

func (p *scriptedChatProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

func newChatRequestTestConfig() *config.Config {
	cfg := newProjectMapDisabledConfig()
	cfg.MCP.Enabled = false
	cfg.LSP.Enabled = false
	cfg.Compression.Enabled = false
	cfg.Compression.KeepRecent = 10
	return cfg
}

func newChatRequestTestAgent(t *testing.T, provider api.Provider, out *bytes.Buffer) *Agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	runtime := NewAgentRuntimeWithConfig(newChatRequestTestConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), out, out)
	runtime.Registry = tools.DefaultRegistry.Clone()

	agent := NewAgentWithRuntime("gpt-5.4", provider, false, runtime)
	agent.setAutoApprove(true)
	return agent
}

func seedHistoryForTokenRetry(agent *Agent, pairs int) {
	for i := 0; i < pairs; i++ {
		agent.History = append(agent.History,
			api.Message{Role: "user", Content: "previous user"},
			api.Message{Role: "assistant", Content: "previous assistant"},
		)
	}
}

func TestChatCore_OneShotReturnsProviderErrorWithoutInteractiveHandling(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockErrorProvider{}, &out)

	err := agent.chatCore("please fail", nil, true)
	if err == nil || !strings.Contains(err.Error(), "mock error") {
		t.Fatalf("chatCore() error = %v, want mock error", err)
	}
	if strings.Contains(out.String(), "Error:") {
		t.Fatalf("oneShot should not print interactive error output, got %q", out.String())
	}
	if strings.Contains(out.String(), "Context ") {
		t.Fatalf("oneShot should skip post-turn context output, got %q", out.String())
	}
}

func TestChatCore_CanceledRequestPrintsInterruptedMessage(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &blockingCancelProvider{started: make(chan struct{})}
	agent := newChatRequestTestAgent(t, provider, &out)

	done := make(chan struct{})
	go func() {
		_ = agent.chatCore("please block", nil, false)
		close(done)
	}()

	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("ChatWithTools was not called")
	}

	agent.cancelActiveRequest("signal: interrupt")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("chatCore did not finish after cancellation")
	}

	if !strings.Contains(out.String(), "Response interrupted") {
		t.Fatalf("expected interruption output, got %q", out.String())
	}
	status := agent.statusRef().getStatus()
	if status.State != StateAborted {
		t.Fatalf("status.State = %q, want %q", status.State, StateAborted)
	}
}

func TestChatCore_TokenLimitRetrySuccess(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{
		name:            "openai",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "", errors.New("input tokens exceed model limit")
			case 1:
				return "compressed summary", nil
			default:
				return "final response", nil
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	seedHistoryForTokenRetry(agent, 6)

	if err := agent.chatCore("please retry", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}
	if provider.callCount != 3 {
		t.Fatalf("provider.callCount = %d, want 3", provider.callCount)
	}
	if !strings.Contains(out.String(), "✅ 自動圧縮＆リトライ成功") {
		t.Fatalf("expected retry success output, got %q", out.String())
	}
	if agent.tokenLimitRetryCount != 0 {
		t.Fatalf("tokenLimitRetryCount = %d, want 0", agent.tokenLimitRetryCount)
	}
	status := agent.statusRef().getStatus()
	if status.State != StateWaitingInput {
		t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
	}
}

func TestChatCore_TokenLimitRetryFailureSetsAbortedStatus(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{
		name:            "openai",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "", errors.New("input tokens exceed model limit")
			case 1:
				return "compressed summary", nil
			default:
				return "", errors.New("input tokens exceed model limit")
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	seedHistoryForTokenRetry(agent, 6)

	if err := agent.chatCore("please fail again", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v, want nil", err)
	}
	if !strings.Contains(out.String(), "圧縮後もトークン上限を超えています") {
		t.Fatalf("expected retry failure output, got %q", out.String())
	}
	status := agent.statusRef().getStatus()
	if status.State != StateAborted {
		t.Fatalf("status.State = %q, want %q", status.State, StateAborted)
	}
}

func TestChatCore_PostTurnUsageDisplay(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	provider.chatWithToolsFn = func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
		if provider.usageCallback != nil {
			provider.usageCallback(api.Usage{
				InputTokens:       100,
				OutputTokens:      40,
				ThinkingTokens:    15,
				CachedInputTokens: 20,
			})
		}
		return "done", nil
	}

	agent := newChatRequestTestAgent(t, provider, &out)
	if err := agent.chatCore("hello", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}
	if !strings.Contains(out.String(), "In: 100 + Out: 40 = 140 tok") {
		t.Fatalf("expected usage output, got %q", out.String())
	}
	if agent.Stats.LastTurnUsage == nil {
		t.Fatal("LastTurnUsage should be recorded")
	}
	if agent.Stats.LastTurnUsage.InputTokens != 100 {
		t.Fatalf("LastTurnUsage.InputTokens = %d, want 100", agent.Stats.LastTurnUsage.InputTokens)
	}
	if agent.Stats.LastTurnUsage.OutputTokens != 40 {
		t.Fatalf("LastTurnUsage.OutputTokens = %d, want 40", agent.Stats.LastTurnUsage.OutputTokens)
	}
	if agent.Stats.LastTurnUsage.ThinkingTokens != 15 {
		t.Fatalf("LastTurnUsage.ThinkingTokens = %d, want 15", agent.Stats.LastTurnUsage.ThinkingTokens)
	}
	if agent.Stats.LastTurnUsage.CachedInputTokens != 20 {
		t.Fatalf("LastTurnUsage.CachedInputTokens = %d, want 20", agent.Stats.LastTurnUsage.CachedInputTokens)
	}
}

func TestExecuteChatRequest_OneShotNormalModeRestoresExcludedTools(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &headlessToolSetProbeProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.registry().SetExcludedTools([]string{"read_file"})

	req := &chatRequest{
		input:   "probe",
		oneShot: true,
	}

	if err := agent.executeChatRequest(context.Background(), req); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}

	if toolNameInList(provider.toolNames, "ask_user_question") {
		t.Fatal("normal mode should exclude ask_user_question")
	}
	if toolNameInList(provider.toolNames, "list_dir") {
		t.Fatal("normal mode should exclude list_dir")
	}
	if !toolNameInList(provider.toolNames, "apply_patch") {
		t.Fatal("normal mode should expose apply_patch")
	}

	gotExcluded := agent.registry().GetExcludedTools()
	sort.Strings(gotExcluded)
	wantExcluded := []string{"read_file"}
	if strings.Join(gotExcluded, ",") != strings.Join(wantExcluded, ",") {
		t.Fatalf("excluded tools after oneShot = %v, want %v", gotExcluded, wantExcluded)
	}
}

func TestExecuteChatRequest_PlanModeUsesPlanModeExcludedTools(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &headlessToolSetProbeProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.PlanModeEnabled = true

	req := &chatRequest{input: "investigate only"}
	if err := agent.executeChatRequest(context.Background(), req); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}

	if !toolNameInList(provider.toolNames, "ask_user_question") {
		t.Fatal("plan mode should expose ask_user_question")
	}
	if toolNameInList(provider.toolNames, "list_dir") {
		t.Fatal("plan mode should exclude list_dir")
	}
}

func TestPrintContextSuggestion_CleanState(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockErrorProvider{}, &out)
	agent.printContextSuggestion()

	if !strings.Contains(out.String(), "clean state") {
		t.Fatalf("expected clean state message, got %q", out.String())
	}
}

func TestPrintContextSuggestion_UsesSavingsMessageWhenPricingIsKnown(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockErrorProvider{}, &out)
	agent.ProviderName = "openai"
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("large context block ", 6000)},
	}

	agent.printContextSuggestion()

	if !strings.Contains(out.String(), "/clear saves") {
		t.Fatalf("expected pricing-aware suggestion, got %q", out.String())
	}
}

func TestPrintContextSuggestion_FallsBackWithoutPricing(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockErrorProvider{}, &out)
	agent.ProviderName = "ollama"
	agent.History = []api.Message{
		{Role: "user", Content: strings.Repeat("large context block ", 6000)},
	}

	agent.printContextSuggestion()

	if !strings.Contains(out.String(), "/clear or /compress") {
		t.Fatalf("expected generic suggestion, got %q", out.String())
	}
}
