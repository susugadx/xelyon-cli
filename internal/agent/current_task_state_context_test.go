package agent

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRequestContext_CurrentTaskStateContextDefaultOff(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateOpenAIResponses)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil when option is off", got)
	}
}

func TestRequestContext_CurrentTaskStateContextClearsInheritedBlocksWhenDefaultOff(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateOpenAIResponses)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want inherited blocks cleared when option is off", got)
	}
}

func TestRequestContext_CurrentTaskStateContextNilLedgerNoop(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options: RuntimeOptions{EnableCurrentTaskStateContext: true},
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateOpenAIResponses)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for nil TaskLedger", got)
	}
}

func TestRequestContext_CurrentTaskStateContextEmptyLedgerNoop(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: ledger.NewStoreWithRoot(t.TempDir()),
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateOpenAIResponses)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for empty TaskLedger", got)
	}
}

func TestRequestContext_CurrentTaskStateContextClearsInheritedBlocksWhenLedgerIsEmpty(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: ledger.NewStoreWithRoot(t.TempDir()),
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateOpenAIResponses)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want inherited blocks cleared when ledger is empty", got)
	}
}

func TestRequestContext_CurrentTaskStateContextIncludesSnapshotWithoutMutatingHistory(t *testing.T) {
	session := history.NewSession("gpt-5.4")
	session.AddMessage("user", "persisted", "gpt-5.4")
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "live"}},
		agentConversationState: agentConversationState{
			session: session,
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateOpenAIResponses)
	beforeHistory := append([]api.Message(nil), agent.History...)
	beforeSessionMessages := append([]history.MessageEntry(nil), session.Messages...)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if len(got) != 1 {
		t.Fatalf("len(ActiveContextBlocksFromContext()) = %d, want 1", len(got))
	}
	if got[0].Name != currentTaskStateActiveContextName {
		t.Fatalf("Name = %q, want %q", got[0].Name, currentTaskStateActiveContextName)
	}
	for _, want := range []string{
		ledger.CurrentTaskStateStartMarker,
		"Last passed tests:",
		"go test ./internal/ledger",
		ledger.CurrentTaskStateEndMarker,
	} {
		if !strings.Contains(got[0].Content, want) {
			t.Fatalf("active context content missing %q:\n%s", want, got[0].Content)
		}
	}

	if !reflect.DeepEqual(agent.History, beforeHistory) {
		t.Fatalf("History mutated:\ngot:  %#v\nwant: %#v", agent.History, beforeHistory)
	}
	if !reflect.DeepEqual(session.Messages, beforeSessionMessages) {
		t.Fatalf("session.Messages mutated:\ngot:  %#v\nwant: %#v", session.Messages, beforeSessionMessages)
	}
}

func TestRequestContext_CurrentTaskStateContextClearsBlocksWhenProviderDoesNotConsumeIt(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateDeepSeek)

	if len(agent.buildActiveContextBlocks()) == 0 {
		t.Fatal("test setup produced empty active context")
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for provider that does not consume active context", got)
	}
}

func TestRequestContextWithoutActiveContextClearsOwnedAndInheritedBlocks(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}
	applyCurrentTaskStateProviderFixture(agent, currentTaskStateOpenAIResponses)
	if len(agent.providerFacingActiveContextBlocks()) == 0 {
		t.Fatal("test setup produced empty provider-facing active context")
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContextWithoutActiveContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for internal model request context", got)
	}
}

func TestShouldSendActiveContextToProvider(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		fixture currentTaskStateProviderFixture
		want    bool
	}{
		{
			name:    "disabled",
			enabled: false,
			fixture: currentTaskStateOpenAIResponses,
			want:    false,
		},
		{
			name:    "openai responses",
			enabled: true,
			fixture: currentTaskStateOpenAIResponses,
			want:    true,
		},
		{
			name:    "azure responses",
			enabled: true,
			fixture: currentTaskStateAzureResponses,
			want:    true,
		},
		{
			name:    "openai chat completions",
			enabled: true,
			fixture: currentTaskStateOpenAIChatCompletions,
			want:    false,
		},
		{
			name:    "deepseek",
			enabled: true,
			fixture: currentTaskStateDeepSeek,
			want:    false,
		},
		{
			name:    "gemini",
			enabled: true,
			fixture: currentTaskStateGemini,
			want:    false,
		},
		{
			name:    "claude",
			enabled: true,
			fixture: currentTaskStateClaude,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				Runtime: &AgentRuntime{
					Options: RuntimeOptions{EnableCurrentTaskStateContext: tt.enabled},
				},
			}
			applyCurrentTaskStateProviderFixture(agent, tt.fixture)

			if got := agent.shouldSendActiveContextToProvider(); got != tt.want {
				t.Fatalf("shouldSendActiveContextToProvider() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEstimateTokens_IncludesModelFacingCurrentTaskStateContext(t *testing.T) {
	agent := &Agent{
		CurrentModel:      "gpt-5.4",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		SystemPrompt:      "system",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "hello"}},
	}

	blocks := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if len(blocks) != 1 {
		t.Fatalf("len(ActiveContextBlocksFromContext()) = %d, want 1", len(blocks))
	}

	wantActive := token.EstimateTokenCountForModel(agent.CurrentModel, blocks[0].Content)
	if got := agent.EstimateActiveContextTokens(); got != wantActive {
		t.Fatalf("EstimateActiveContextTokens() = %d, want %d", got, wantActive)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) +
		token.EstimateTokenCountForModel(agent.CurrentModel, "hello") +
		wantActive
	if got := agent.EstimateTokens(); got != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want system + history + active context = %d", got, wantTotal)
	}
}

func TestEstimateTokens_DoesNotCountCurrentTaskStateForProvidersThatDoNotConsumeIt(t *testing.T) {
	agent := &Agent{
		CurrentModel:      "deepseek-chat",
		ProviderName:      "deepseek",
		ProviderConfigKey: "deepseek",
		SystemPrompt:      "system",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "hello"}},
	}

	if got := agent.EstimateActiveContextTokens(); got != 0 {
		t.Fatalf("EstimateActiveContextTokens() = %d, want 0 for non-Responses provider", got)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) +
		token.EstimateTokenCountForModel(agent.CurrentModel, "hello")
	if got := agent.EstimateTokens(); got != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want system + history = %d", got, wantTotal)
	}
}

func TestHandleTokensCommand_ShowsCurrentTaskStateContextTokens(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		CurrentModel:      "gpt-5.4",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
			UI:         ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "Active Context:") {
		t.Fatalf("tokens output missing active context row:\n%s", out.String())
	}
}

func TestPrepareChatRequest_CurrentTaskStateContextClearsResponseContextBeforeSessionSave(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent, provider := newCurrentTaskStateResponseIDAgent(t, currentTaskStateOpenAIResponses, "resp_old", &out)

	agent.prepareChatRequest(&chatRequest{input: "hello"})

	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want cleared before active-context request", provider.GetResponseID())
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared before session save", agent.session.ResponseID)
	}
	if len(agent.session.Messages) == 0 || agent.session.Messages[len(agent.session.Messages)-1].Content != "hello" {
		t.Fatalf("session messages = %#v, want appended user message", agent.session.Messages)
	}
}

func TestPrepareChatRequest_CurrentTaskStateContextKeepsResponseContextForProvidersThatDoNotConsumeIt(t *testing.T) {
	disableColors(t)

	tests := []struct {
		name    string
		fixture currentTaskStateProviderFixture
	}{
		{
			name:    "deepseek",
			fixture: currentTaskStateDeepSeek,
		},
		{
			name:    "gemini",
			fixture: currentTaskStateGemini,
		},
		{
			name:    "claude",
			fixture: currentTaskStateClaude,
		},
		{
			name:    "openai chat completions",
			fixture: currentTaskStateOpenAIChatCompletions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			agent, provider := newCurrentTaskStateResponseIDAgent(t, tt.fixture, "resp_old", &out)

			if len(agent.buildActiveContextBlocks()) == 0 {
				t.Fatal("test setup produced empty active context")
			}
			if agent.shouldSendActiveContextToProvider() {
				t.Fatalf("test setup provider/model %s/%s consumes active context", tt.fixture.providerName, tt.fixture.model)
			}

			agent.prepareChatRequest(&chatRequest{input: "hello"})

			if provider.GetResponseID() != "resp_old" {
				t.Fatalf("provider response ID = %q, want preserved for provider that does not consume active context", provider.GetResponseID())
			}
			if agent.session.ResponseID != "resp_old" {
				t.Fatalf("session.ResponseID = %q, want preserved for provider that does not consume active context", agent.session.ResponseID)
			}
			if len(agent.session.Messages) == 0 || agent.session.Messages[len(agent.session.Messages)-1].Content != "hello" {
				t.Fatalf("session messages = %#v, want appended user message", agent.session.Messages)
			}
		})
	}
}

func contextWithInheritedActiveContext() context.Context {
	return api.WithActiveContextBlocks(context.Background(), []api.ActiveContextBlock{{
		Name:    currentTaskStateActiveContextName,
		Content: "parent snapshot",
	}})
}
