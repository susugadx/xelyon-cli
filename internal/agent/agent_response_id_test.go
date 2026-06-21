package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
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

	t.Run("persist disabled clears saved response context", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Responses.PersistResponseID = false
		agent := &Agent{
			CurrentModel:      "gpt-5",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentProvider:   &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "resp_live"},
			Runtime:           NewAgentRuntimeWithConfig(cfg),
			agentConversationState: agentConversationState{
				session: newResponseContextSession("saved-model", "openai", "openai", "resp_saved"),
			},
		}

		agent.syncSavedResponseContextFromProvider()

		if agent.session.ResponseID != "" {
			t.Fatalf("session.ResponseID = %q, want cleared when persist_response_id=false", agent.session.ResponseID)
		}
		if agent.session.ResponseModel != "" || agent.session.ResponseProviderName != "" || agent.session.ResponseProviderConfigKey != "" {
			t.Fatalf(
				"saved response context = (%q, %q, %q), want cleared",
				agent.session.ResponseModel,
				agent.session.ResponseProviderName,
				agent.session.ResponseProviderConfigKey,
			)
		}
	})
}

func TestAzureProviderIdentityAndResponseIDContract(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	t.Setenv("XELYON_DISABLE_MCP", "1")

	cfg := newProjectMapDisabledConfig()
	cfg.MCP.Enabled = false
	runtime := NewAgentRuntimeWithConfig(cfg)
	provider := azureprovider.New("azure-key")

	agent := NewAgentWithRuntime("corp-gpt55-deployment", provider, false, runtime)
	t.Cleanup(agent.Cleanup)

	if agent.CurrentProvider.Name() != "Azure OpenAI" {
		t.Fatalf("CurrentProvider.Name() = %q, want Azure OpenAI display name", agent.CurrentProvider.Name())
	}
	if agent.ProviderName != "azure" {
		t.Fatalf("ProviderName = %q, want azure", agent.ProviderName)
	}
	if agent.ProviderConfigKey != "azure" {
		t.Fatalf("ProviderConfigKey = %q, want azure", agent.ProviderConfigKey)
	}
	if agent.session.ProviderName != "azure" || agent.session.ProviderConfigKey != "azure" {
		t.Fatalf(
			"session provider identity = (%q, %q), want (azure, azure)",
			agent.session.ProviderName,
			agent.session.ProviderConfigKey,
		)
	}
	if !strings.Contains(agent.SystemPrompt, "OpenAI-specific") {
		t.Fatalf("SystemPrompt does not include OpenAI provider notes for Azure")
	}

	ridProvider, ok := agent.CurrentProvider.(ResponseIDCapable)
	if !ok {
		t.Fatalf("Azure provider does not implement ResponseIDCapable")
	}
	ridProvider.SetResponseID("resp_azure")
	agent.syncSavedResponseContextFromProvider()
	if agent.session.ResponseID != "resp_azure" {
		t.Fatalf("session.ResponseID = %q, want resp_azure", agent.session.ResponseID)
	}
	if agent.session.ResponseProviderName != "azure" || agent.session.ResponseProviderConfigKey != "azure" {
		t.Fatalf(
			"session response provider identity = (%q, %q), want (azure, azure)",
			agent.session.ResponseProviderName,
			agent.session.ResponseProviderConfigKey,
		)
	}

	ridProvider.SetResponseID("")
	agent.restoreSessionResponseIDForCurrentContext()
	if got := ridProvider.GetResponseID(); got != "resp_azure" {
		t.Fatalf("restored response ID = %q, want resp_azure", got)
	}
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

	t.Run("persist disabled clears provider cache and saved context", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Responses.PersistResponseID = false
		provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "old"}
		agent := &Agent{
			CurrentModel:      "saved-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentProvider:   provider,
			Runtime:           NewAgentRuntimeWithConfig(cfg),
			agentConversationState: agentConversationState{
				session: newResponseContextSession("saved-model", "openai", "openai", "resp_123"),
			},
		}

		agent.restoreSessionResponseIDForCurrentContext()

		if provider.responseID != "" {
			t.Fatalf("provider.responseID = %q, want cleared when persist_response_id=false", provider.responseID)
		}
		if agent.session.ResponseID != "" {
			t.Fatalf("session.ResponseID = %q, want cleared when persist_response_id=false", agent.session.ResponseID)
		}
	})

	t.Run("persist disabled clears saved context without response capable provider", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Responses.PersistResponseID = false
		agent := &Agent{
			CurrentModel:      "saved-model",
			ProviderName:      "deepseek",
			ProviderConfigKey: "deepseek",
			CurrentProvider:   &mockProvider{name: "deepseek"},
			Runtime:           NewAgentRuntimeWithConfig(cfg),
			agentConversationState: agentConversationState{
				session: newResponseContextSession("saved-model", "openai", "openai", "resp_123"),
			},
		}

		agent.restoreSessionResponseIDForCurrentContext()

		if agent.session.ResponseID != "" {
			t.Fatalf("session.ResponseID = %q, want cleared even without ResponseIDCapable provider", agent.session.ResponseID)
		}
	})
}

func TestShouldRestoreSessionResponseID_CanonicalizesAzureDisplayNameProviderConfigKey(t *testing.T) {
	if !shouldRestoreSessionResponseID(
		"corp-gpt55-deployment",
		"corp-gpt55-deployment",
		"Azure OpenAI",
		"azure",
		"Azure OpenAI",
		"azure",
		"resp_azure",
	) {
		t.Fatal("shouldRestoreSessionResponseID() = false, want true for Azure display-name provider_config_key compatibility")
	}
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
	if ridProvider.responseID != "resp_saved" {
		t.Fatalf("provider.responseID = %q, want restored saved response id", ridProvider.responseID)
	}
	if agent.session.ResponseID != "resp_saved" {
		t.Fatalf("session.ResponseID = %q, want preserved saved response id", agent.session.ResponseID)
	}
	if agent.session.ResponseModel != "saved-model" {
		t.Fatalf("session.ResponseModel = %q, want preserved saved model", agent.session.ResponseModel)
	}
	if agent.session.Model != "saved-model" {
		t.Fatalf("session.Model = %q, want restored saved runtime model", agent.session.Model)
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
	if reloaded.Model != "saved-model" {
		t.Fatalf("reloaded.Model = %q, want persisted saved runtime model", reloaded.Model)
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

func TestHandleModelCommand_ClearsSavedResponseIDWhenSwitchingModel(t *testing.T) {
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
		History: []api.Message{
			{Role: "user", Content: "old task"},
		},
		agentConversationState: agentConversationState{
			session: newResponseContextSession("saved-model", "openai", "openai", "resp_saved"),
		},
	}
	agent.session.AddMessage("user", "old task", "different-model")
	agent.Runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, &out)

	if !handleModelCommand(agent, []string{"saved-model"}) {
		t.Fatal("handleModelCommand() = false, want true")
	}

	if ridProvider, ok := agent.CurrentProvider.(*mockResponseIDProvider); !ok || ridProvider.responseID != "" {
		t.Fatalf("provider responseID = %#v, want cleared saved response id", agent.CurrentProvider)
	}
	if agent.session.ResponseID != "" || agent.session.ResponseModel != "" || agent.session.ResponseProviderName != "" || agent.session.ResponseProviderConfigKey != "" {
		t.Fatalf("session response context = (%q, %q, %q, %q), want cleared",
			agent.session.ResponseID,
			agent.session.ResponseModel,
			agent.session.ResponseProviderName,
			agent.session.ResponseProviderConfigKey,
		)
	}
	if len(agent.History) != 1 || len(agent.session.Messages) != 1 {
		t.Fatalf("conversation should be kept, got history=%d session=%d", len(agent.History), len(agent.session.Messages))
	}
	if !strings.Contains(out.String(), "Context kept locally; provider remote continuation reset") {
		t.Fatalf("output = %q, want context kept notice", out.String())
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
