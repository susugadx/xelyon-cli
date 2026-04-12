package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newResponseContextSession(model, providerName, providerConfigKey, responseID string) *history.Session {
	session := history.NewSession(model)
	session.ProviderName = providerName
	session.ProviderConfigKey = providerConfigKey
	session.ResponseID = responseID
	session.ResponseModel = model
	session.ResponseProviderName = providerName
	session.ResponseProviderConfigKey = providerConfigKey
	return session
}

func TestSyncSavedResponseContextFromProvider(t *testing.T) {
	t.Run("persists cached provider response context", func(t *testing.T) {
		provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "resp_live"}
		agent := &Agent{
			CurrentModel:      "gpt-5",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentProvider:   provider,
			agentConversationState: agentConversationState{
				session: history.NewSession("stale-model"),
			},
		}

		agent.syncSavedResponseContextFromProvider()

		if agent.session.ResponseID != "resp_live" {
			t.Fatalf("session.ResponseID = %q, want %q", agent.session.ResponseID, "resp_live")
		}
		if agent.session.ResponseModel != "gpt-5" {
			t.Fatalf("session.ResponseModel = %q, want %q", agent.session.ResponseModel, "gpt-5")
		}
		if agent.session.ResponseProviderName != "openai" || agent.session.ResponseProviderConfigKey != "openai" {
			t.Fatalf(
				"session response provider identity = (%q, %q), want (%q, %q)",
				agent.session.ResponseProviderName,
				agent.session.ResponseProviderConfigKey,
				"openai",
				"openai",
			)
		}
	})

	t.Run("missing provider cache preserves saved response context", func(t *testing.T) {
		agent := &Agent{
			CurrentModel:      "other-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentProvider:   &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}},
			agentConversationState: agentConversationState{
				session: newResponseContextSession("saved-model", "openai", "openai", "resp_saved"),
			},
		}

		agent.syncSavedResponseContextFromProvider()

		if agent.session.ResponseID != "resp_saved" {
			t.Fatalf("session.ResponseID = %q, want preserved saved value", agent.session.ResponseID)
		}
		if agent.session.ResponseModel != "saved-model" {
			t.Fatalf("session.ResponseModel = %q, want preserved saved model", agent.session.ResponseModel)
		}
	})
}

func TestRestoreSessionResponseIDForCurrentContext(t *testing.T) {
	t.Run("matching saved response context restores provider cache", func(t *testing.T) {
		provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "old"}
		agent := &Agent{
			CurrentModel:      "saved-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentProvider:   provider,
			agentConversationState: agentConversationState{
				session: newResponseContextSession("saved-model", "openai", "openai", "resp_123"),
			},
		}
		agent.session.Model = "different-runtime-model"

		agent.restoreSessionResponseIDForCurrentContext()

		if provider.responseID != "resp_123" {
			t.Fatalf("provider.responseID = %q, want restored response id", provider.responseID)
		}
		if agent.session.ResponseID != "resp_123" {
			t.Fatalf("session.ResponseID = %q, want preserved saved response id", agent.session.ResponseID)
		}
	})

	t.Run("different model clears provider cache but preserves saved context", func(t *testing.T) {
		provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "old"}
		agent := &Agent{
			CurrentModel:      "different-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentProvider:   provider,
			agentConversationState: agentConversationState{
				session: newResponseContextSession("saved-model", "openai", "openai", "resp_123"),
			},
		}

		agent.restoreSessionResponseIDForCurrentContext()

		if provider.responseID != "" {
			t.Fatalf("provider.responseID = %q, want cleared", provider.responseID)
		}
		if agent.session.ResponseID != "resp_123" {
			t.Fatalf("session.ResponseID = %q, want saved response id to remain", agent.session.ResponseID)
		}
		if agent.session.ResponseModel != "saved-model" {
			t.Fatalf("session.ResponseModel = %q, want saved model to remain", agent.session.ResponseModel)
		}
	})

	t.Run("different provider config clears provider cache but preserves saved context", func(t *testing.T) {
		provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "old"}
		agent := &Agent{
			CurrentModel:      "saved-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai-alt",
			CurrentProvider:   provider,
			agentConversationState: agentConversationState{
				session: newResponseContextSession("saved-model", "openai", "openai", "resp_123"),
			},
		}

		agent.restoreSessionResponseIDForCurrentContext()

		if provider.responseID != "" {
			t.Fatalf("provider.responseID = %q, want cleared", provider.responseID)
		}
		if agent.session.ResponseID != "resp_123" {
			t.Fatalf("session.ResponseID = %q, want saved response id to remain", agent.session.ResponseID)
		}
	})

	t.Run("legacy openai response context without provider owner remains alias agnostic", func(t *testing.T) {
		provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "old"}
		session := history.NewSession("saved-model")
		session.ResponseID = "resp_legacy"
		session.ResponseModel = "saved-model"
		session.ResponseProviderName = "openai"
		agent := &Agent{
			CurrentModel:      "saved-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai-alt",
			CurrentProvider:   provider,
			agentConversationState: agentConversationState{
				session: session,
			},
		}

		agent.restoreSessionResponseIDForCurrentContext()

		if provider.responseID != "resp_legacy" {
			t.Fatalf("provider.responseID = %q, want restored legacy response id", provider.responseID)
		}
		if agent.session.ResponseID != "resp_legacy" {
			t.Fatalf("session.ResponseID = %q, want saved response id preserved", agent.session.ResponseID)
		}
	})

	t.Run("missing saved provider identity does not guess openai anymore", func(t *testing.T) {
		provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "old"}
		session := history.NewSession("saved-model")
		session.ResponseID = "resp_legacyish"
		session.ResponseModel = "saved-model"
		agent := &Agent{
			CurrentModel:      "saved-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentProvider:   provider,
			agentConversationState: agentConversationState{
				session: session,
			},
		}

		agent.restoreSessionResponseIDForCurrentContext()

		if provider.responseID != "" {
			t.Fatalf("provider.responseID = %q, want cleared when saved provider identity is missing", provider.responseID)
		}
		if agent.session.ResponseID != "resp_legacyish" {
			t.Fatalf("session.ResponseID = %q, want saved response id preserved", agent.session.ResponseID)
		}
	})
}

func TestHandleLoadCommand_PreservesSavedResponseContextAcrossMismatchedLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := newResponseContextSession("saved-model", "openai", "openai", "resp_saved")
	session.AddMessage("user", "previous question", "saved-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	agent := newSessionCommandTestAgent(&out)
	agent.storage = storage
	agent.CurrentModel = "different-model"
	agent.ProviderName = "openai"
	agent.ProviderConfigKey = "openai"
	agent.CurrentProvider = &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "old-provider-cache"}

	if !handleLoadCommand(agent, []string{session.ID}) {
		t.Fatal("handleLoadCommand() = false, want true")
	}

	ridProvider := agent.CurrentProvider.(*mockResponseIDProvider)
	if ridProvider.responseID != "" {
		t.Fatalf("provider.responseID = %q, want cleared for mismatched runtime", ridProvider.responseID)
	}
	if agent.session.ResponseID != "resp_saved" {
		t.Fatalf("session.ResponseID = %q, want preserved saved response id", agent.session.ResponseID)
	}
	if agent.session.ResponseModel != "saved-model" {
		t.Fatalf("session.ResponseModel = %q, want preserved saved model", agent.session.ResponseModel)
	}
	if agent.session.Model != "different-model" {
		t.Fatalf("session.Model = %q, want reconciled current runtime model", agent.session.Model)
	}

	agent.Cleanup()

	reloaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() after cleanup error = %v", err)
	}
	if reloaded.ResponseID != "resp_saved" {
		t.Fatalf("reloaded.ResponseID = %q, want preserved saved response id", reloaded.ResponseID)
	}
	if reloaded.ResponseModel != "saved-model" {
		t.Fatalf("reloaded.ResponseModel = %q, want preserved saved model", reloaded.ResponseModel)
	}
	if reloaded.Model != "different-model" {
		t.Fatalf("reloaded.Model = %q, want persisted current runtime model", reloaded.Model)
	}

	var out2 bytes.Buffer
	agent2 := newSessionCommandTestAgent(&out2)
	agent2.storage = storage
	agent2.CurrentModel = "saved-model"
	agent2.ProviderName = "openai"
	agent2.ProviderConfigKey = "openai"
	agent2.CurrentProvider = &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}}

	if !handleLoadCommand(agent2, []string{session.ID}) {
		t.Fatal("second handleLoadCommand() = false, want true")
	}

	if ridProvider2, ok := agent2.CurrentProvider.(*mockResponseIDProvider); !ok || ridProvider2.responseID != "resp_saved" {
		t.Fatalf("second load responseID = %#v, want restored saved response id", agent2.CurrentProvider)
	}
}

func TestAppendSessionMessage_InvalidatesSavedResponseContextOnlyAfterDivergingTurn(t *testing.T) {
	agent := &Agent{
		CurrentModel:      "different-model",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentProvider:   &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}},
		agentConversationState: agentConversationState{
			session: newResponseContextSession("saved-model", "openai", "openai", "resp_saved"),
		},
	}

	agent.appendSessionMessage("user", "new question", "different-model")

	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared after diverging turn", agent.session.ResponseID)
	}
	if agent.session.ResponseModel != "" || agent.session.ResponseProviderName != "" || agent.session.ResponseProviderConfigKey != "" {
		t.Fatalf(
			"saved response context = (%q, %q, %q), want cleared",
			agent.session.ResponseModel,
			agent.session.ResponseProviderName,
			agent.session.ResponseProviderConfigKey,
		)
	}
	if len(agent.session.Messages) != 1 || agent.session.Messages[0].Content != "new question" {
		t.Fatalf("session.Messages = %#v, want appended new message", agent.session.Messages)
	}
}

func TestHandleModelCommand_RestoresSavedResponseIDWhenReturningToMatchingModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	withConfigCommandHooks(t)
	cfg := newProjectMapDisabledConfig()
	loadConfigForCommand = func() (*config.Config, error) {
		return cfg, nil
	}
	saveConfigForCommand = func(*config.Config) error {
		return nil
	}

	var out bytes.Buffer
	agent := &Agent{
		CurrentModel:      "different-model",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentProvider:   &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}},
		Runtime:           NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: newResponseContextSession("saved-model", "openai", "openai", "resp_saved"),
		},
	}
	agent.Runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	if !handleModelCommand(agent, []string{"saved-model"}) {
		t.Fatal("handleModelCommand() = false, want true")
	}

	if ridProvider, ok := agent.CurrentProvider.(*mockResponseIDProvider); !ok || ridProvider.responseID != "resp_saved" {
		t.Fatalf("provider responseID = %#v, want restored saved response id", agent.CurrentProvider)
	}
}

func TestApplyLoadedSession_ReconcilesRuntimeIdentityAndRestoresHistory(t *testing.T) {
	provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}}
	session := newResponseContextSession("saved-model", "openai", "openai", "resp_123")
	session.AddMessage("user", "hello", "saved-model")

	agent := &Agent{
		CurrentModel:      "saved-model",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentProvider:   provider,
	}

	agent.applyLoadedSession(session)

	if agent.session != session {
		t.Fatal("agent.session was not replaced with loaded session")
	}
	if len(agent.History) != 1 || agent.History[0].Content != "hello" {
		t.Fatalf("agent.History = %#v, want restored API history", agent.History)
	}
	if agent.session.Model != "saved-model" {
		t.Fatalf("session.Model = %q, want reconciled current runtime model", agent.session.Model)
	}
	if provider.responseID != "resp_123" {
		t.Fatalf("provider.responseID = %q, want restored response id", provider.responseID)
	}
}
