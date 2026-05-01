package agent

import (
	"errors"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestPlanModeTokenLimitRetry_DoesNotPersistCompressedPlanningHistoryBeforeRestore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &compressionTestProvider{name: "openai", summary: "plan retry summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	preExisting := []api.Message{
		{Role: "user", Content: "pre-existing question"},
		{Role: "assistant", Content: "pre-existing answer"},
	}
	agent.History = append([]api.Message(nil), preExisting...)
	for _, msg := range preExisting {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	agent.session.AddMessage("user", "plan request", agent.CurrentModel)
	if err := agent.storage.Rewrite(agent.session); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	checkpoint := capturePlanModeCheckpoint(agent, "plan request")
	agent.History = append(agent.History,
		api.Message{Role: "user", Content: "planning prompt"},
		api.Message{Role: "assistant", Content: "planning response"},
	)

	handled := agent.handleTokenLimitErrorWithRetryOptions(
		errors.New("context length exceeded"),
		func() error {
			return checkpoint.restore(agent)
		},
		tokenLimitRetryOptions{skipCompressionPersistence: true},
	)
	if !handled {
		t.Fatal("handleTokenLimitErrorWithRetryOptions() = false, want true")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	messages := loaded.ToAPIMessages()
	if len(messages) != len(preExisting) {
		t.Fatalf("len(loaded messages) = %d, want %d: %#v", len(messages), len(preExisting), messages)
	}
	for i, want := range preExisting {
		if messages[i].Role != want.Role || messages[i].Content != want.Content {
			t.Fatalf("loaded message[%d] = %#v, want %#v", i, messages[i], want)
		}
	}
}
