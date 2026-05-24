package agent

import (
	"bytes"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func newCurrentTaskStateAgent(t *testing.T, provider api.Provider, fixture activeContextProviderFixture, out *bytes.Buffer) *Agent {
	t.Helper()
	agent := newChatRequestTestAgent(t, provider, out)
	applyActiveContextProviderFixture(agent, fixture)
	enableCurrentTaskStateContext(t, agent)
	return agent
}

func newCurrentTaskStateResponseIDAgent(
	t *testing.T,
	fixture activeContextProviderFixture,
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

func enableCurrentTaskStateContext(t *testing.T, agent *Agent) {
	t.Helper()
	agent.Runtime.Options.EnableCurrentTaskStateContext = true
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)
}

func assertCurrentTaskLedgerReset(t *testing.T, agent *Agent, action string) {
	t.Helper()
	assertTaskLedgerReset(t, agent, action)
}

func assertCurrentTaskLedgerPreserved(t *testing.T, agent *Agent, action string) {
	t.Helper()
	assertTaskLedgerPreserved(t, agent, action)
}
