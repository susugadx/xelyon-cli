package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestPlanModeCheckpointRestoreSessionResponseContext_CanonicalizesAzureDisplayNameProviderConfigKey(t *testing.T) {
	checkpoint := &planModeCheckpoint{
		responseID:          "resp_azure",
		responseModel:       "corp-gpt55-deployment",
		responseProvider:    "Azure OpenAI",
		responseProviderKey: "Azure OpenAI",
	}
	session := history.NewSession("corp-gpt55-deployment")
	session.ProviderName = "azure"
	session.ProviderConfigKey = "azure"

	checkpoint.restoreSessionResponseContext(session)

	if session.ResponseProviderName != "azure" {
		t.Fatalf("session.ResponseProviderName = %q, want azure", session.ResponseProviderName)
	}
	if session.ResponseProviderConfigKey != "azure" {
		t.Fatalf("session.ResponseProviderConfigKey = %q, want azure", session.ResponseProviderConfigKey)
	}
}

func TestPlanModeCheckpoint_RespectsResponsesPersistencePolicy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Responses.PersistResponseID = false
	provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "resp_live"}
	agent := &Agent{
		CurrentModel:      "gpt-5",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentProvider:   provider,
		Runtime:           NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: newResponseContextSession("saved-model", "openai", "openai", "resp_saved"),
		},
	}

	checkpoint := capturePlanModeCheckpoint(agent, "")
	provider.SetResponseID("")

	if err := checkpoint.restore(agent); err != nil {
		t.Fatalf("checkpoint.restore() error = %v", err)
	}
	if provider.responseID != "resp_live" {
		t.Fatalf("provider.responseID = %q, want in-memory checkpoint response id restored", provider.responseID)
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared when persist_response_id=false", agent.session.ResponseID)
	}
}

func TestPlanModeCheckpoint_StoreFalseClearsProviderResponseID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Responses.Store = false
	provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "resp_live"}
	agent := &Agent{
		CurrentModel:      "gpt-5",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentProvider:   provider,
		Runtime:           NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: newResponseContextSession("saved-model", "openai", "openai", "resp_saved"),
		},
	}

	checkpoint := capturePlanModeCheckpoint(agent, "")

	if err := checkpoint.restore(agent); err != nil {
		t.Fatalf("checkpoint.restore() error = %v", err)
	}
	if provider.responseID != "" {
		t.Fatalf("provider.responseID = %q, want cleared when responses.store=false", provider.responseID)
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared when responses.store=false", agent.session.ResponseID)
	}
}
