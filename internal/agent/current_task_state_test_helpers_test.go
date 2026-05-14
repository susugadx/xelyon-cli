package agent

import (
	"bytes"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type currentTaskStateProviderFixture struct {
	providerName      string
	providerConfigKey string
	model             string
}

var (
	currentTaskStateOpenAIResponses = currentTaskStateProviderFixture{
		providerName:      "openai",
		providerConfigKey: "openai",
		model:             "gpt-5.4",
	}
	currentTaskStateAzureResponses = currentTaskStateProviderFixture{
		providerName:      "azure",
		providerConfigKey: "azure",
		model:             "corp-gpt55-deployment",
	}
	currentTaskStateOpenAIChatCompletions = currentTaskStateProviderFixture{
		providerName:      "openai",
		providerConfigKey: "openai",
		model:             "gpt-4-turbo",
	}
	currentTaskStateDeepSeek = currentTaskStateProviderFixture{
		providerName:      "deepseek",
		providerConfigKey: "deepseek",
		model:             "deepseek-chat",
	}
	currentTaskStateGemini = currentTaskStateProviderFixture{
		providerName:      "gemini",
		providerConfigKey: "gemini",
		model:             "gemini-2.5-pro",
	}
	currentTaskStateClaude = currentTaskStateProviderFixture{
		providerName:      "claude",
		providerConfigKey: "claude",
		model:             "claude-sonnet-4-6",
	}
)

func newCurrentTaskStateAgent(t *testing.T, provider api.Provider, fixture currentTaskStateProviderFixture, out *bytes.Buffer) *Agent {
	t.Helper()
	agent := newChatRequestTestAgent(t, provider, out)
	applyCurrentTaskStateProviderFixture(agent, fixture)
	enableCurrentTaskStateContext(t, agent)
	return agent
}

func newCurrentTaskStateResponseIDAgent(
	t *testing.T,
	fixture currentTaskStateProviderFixture,
	responseID string,
	out *bytes.Buffer,
) (*Agent, *mockResponseIDProvider) {
	t.Helper()
	provider := &mockResponseIDProvider{
		mockProvider: mockProvider{name: fixture.providerName},
		responseID:   responseID,
	}
	agent := newCurrentTaskStateAgent(t, provider, fixture, out)
	agent.session.ApplyResponseContext(responseID, agent.CurrentModel, agent.ProviderName, agent.ProviderConfigKey)
	return agent, provider
}

func applyCurrentTaskStateProviderFixture(agent *Agent, fixture currentTaskStateProviderFixture) {
	agent.CurrentModel = fixture.model
	agent.ProviderName = fixture.providerName
	agent.ProviderConfigKey = fixture.providerConfigKey
}

func enableCurrentTaskStateContext(t *testing.T, agent *Agent) {
	t.Helper()
	agent.Runtime.Options.EnableCurrentTaskStateContext = true
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)
}

func assertCurrentTaskLedgerReset(t *testing.T, agent *Agent, action string) {
	t.Helper()
	if !agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatalf("task ledger should be reset after %s: %#v", action, agent.Runtime.TaskLedger.Snapshot())
	}
}

func assertCurrentTaskLedgerPreserved(t *testing.T, agent *Agent, action string) {
	t.Helper()
	if agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatalf("task ledger was reset after %s", action)
	}
}
