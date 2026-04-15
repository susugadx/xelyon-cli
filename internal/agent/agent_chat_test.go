package agent

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestAgent_Cleanup_NoStorage(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)

	agent.storage = nil
	agent.session = nil
	agent.Cleanup()
}

func TestAgent_SwitchProvider_Success(t *testing.T) {
	os.Setenv("DEEPSEEK_API_KEY", "test-key")
	defer os.Unsetenv("DEEPSEEK_API_KEY")

	provider := &mockProvider{name: "test"}
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	agent.ProviderName = "test"
	agent.Stats = NewSessionStats("test")

	err := agent.SwitchProvider("deepseek")
	if err != nil {
		t.Fatalf("SwitchProvider() error = %v", err)
	}
	if agent.ProviderName != "deepseek" {
		t.Errorf("SwitchProvider() ProviderName = %v, want 'deepseek'", agent.ProviderName)
	}
	if agent.Stats.Provider != "deepseek" {
		t.Errorf("SwitchProvider() Stats.Provider = %v, want 'deepseek'", agent.Stats.Provider)
	}
	if agent.CurrentProvider == nil {
		t.Error("SwitchProvider() CurrentProvider should not be nil")
	}
}

func TestAgent_SwitchProvider_NoAPIKey_ChatTest(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")

	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)

	err := agent.SwitchProvider("deepseek")
	if err == nil {
		t.Error("SwitchProvider() should return error when API key is not set")
	}
}

func TestPrintHeader_ChatTest(t *testing.T) {
	provider := &mockProvider{name: "Test Provider"}
	printHeaderToWriter(io.Discard, "test-model", provider)
}

func TestChatCore_SetsAbortedStatusWithActualError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)
	agent := NewAgentWithRuntime("test-model", &mockErrorProvider{}, false, runtime)

	if err := agent.chatCore("please fail", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v, want nil", err)
	}

	status := agent.statusRef().getStatus()
	if status.State != StateAborted {
		t.Fatalf("status.State = %q, want %q", status.State, StateAborted)
	}
	if !strings.Contains(status.ReasonEN, "mock error") {
		t.Fatalf("status.ReasonEN = %q, want to contain %q", status.ReasonEN, "mock error")
	}
	if status.ReasonEN == "Request failed" {
		t.Fatalf("status.ReasonEN = %q, want concrete error summary", status.ReasonEN)
	}
}

func TestChatCore_SetsAbortedStatusWithCancelReason(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	provider := &blockingCancelProvider{started: make(chan struct{})}
	agent := newAgentChatTestAgent(t, provider)
	cfg := config.DefaultConfig()
	cfg.ProjectMap.Enabled = false
	agent.Runtime = &AgentRuntime{
		Config:   cfg,
		Registry: tools.DefaultRegistry.Clone(),
		UI:       ui.NewRuntime(strings.NewReader(""), &out, &out),
	}

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

	status := agent.statusRef().getStatus()
	if status.State != StateAborted {
		t.Fatalf("status.State = %q, want %q", status.State, StateAborted)
	}
	if !strings.Contains(status.ReasonEN, "signal: interrupt") {
		t.Fatalf("status.ReasonEN = %q, want to contain %q", status.ReasonEN, "signal: interrupt")
	}
}

func TestPrintTaskUsage(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)
	agent.Stats = NewSessionStats("openai", "gpt-5.4")
	agent.ProviderName = "openai"

	startStats := SessionStats{
		InputTokens:       100,
		OutputTokens:      50,
		ThinkingTokens:    20,
		CachedInputTokens: 40,
		Provider:          "openai",
		Model:             "gpt-5.4",
		AccumulatedCost:   0.0100,
	}

	agent.Stats.InputTokens = 200
	agent.Stats.OutputTokens = 100
	agent.Stats.ThinkingTokens = 35
	agent.Stats.CachedInputTokens = 90
	agent.Stats.AccumulatedCost = 0.0325

	var out bytes.Buffer
	agent.Runtime = &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	}

	agent.printTaskUsage(startStats)
	output := out.String()

	if !strings.Contains(output, "100") || !strings.Contains(output, "50") || !strings.Contains(output, "150") {
		t.Errorf("printTaskUsage output incorrect, got: %s", output)
	}
	if agent.Stats.LastTurnUsage == nil {
		t.Fatal("LastTurnUsage should be recorded")
	}
	if agent.Stats.LastTurnUsage.InputTokens != 100 {
		t.Fatalf("LastTurnUsage.InputTokens = %d, want 100", agent.Stats.LastTurnUsage.InputTokens)
	}
	if agent.Stats.LastTurnUsage.CachedInputTokens != 50 {
		t.Fatalf("LastTurnUsage.CachedInputTokens = %d, want 50", agent.Stats.LastTurnUsage.CachedInputTokens)
	}
	if agent.Stats.LastTurnUsage.ThinkingTokens != 15 {
		t.Fatalf("LastTurnUsage.ThinkingTokens = %d, want 15", agent.Stats.LastTurnUsage.ThinkingTokens)
	}
	if agent.Stats.LastTurnCost < 0.0224 || agent.Stats.LastTurnCost > 0.0226 {
		t.Fatalf("LastTurnCost = %.4f, want about 0.0225", agent.Stats.LastTurnCost)
	}
}
