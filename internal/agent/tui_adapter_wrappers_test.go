package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestTUIAdapter_WrapperStateAccessors(t *testing.T) {
	disableColors(t)

	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "claude-sonnet-4-6",
		Stats:             &SessionStats{},
		Runtime:           NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
	}

	adapter := NewTUIAdapter(agent, nil)

	if got, want := adapter.GetStatusLine(), agent.FormatStatusLine(); got != want {
		t.Fatalf("GetStatusLine() = %q, want %q", got, want)
	}
	if adapter.IsProcessing() {
		t.Fatal("IsProcessing() = true, want false")
	}
	adapter.processing.Store(true)
	if !adapter.IsProcessing() {
		t.Fatal("IsProcessing() = false, want true")
	}
	if got := adapter.GetProviderName(); got != "claude" {
		t.Fatalf("GetProviderName() = %q, want %q", got, "claude")
	}
	if got := adapter.GetProviderConfigKey(); got != "anthropic" {
		t.Fatalf("GetProviderConfigKey() = %q, want %q", got, "anthropic")
	}
}

func TestTUIAdapter_StatusSnapshotLabelsPlanModeState(t *testing.T) {
	disableColors(t)

	agent := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		Stats:           &SessionStats{},
		Runtime:         NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
		PlanModeEnabled: false,
	}
	adapter := NewTUIAdapter(agent, nil)

	if got := adapter.StatusSnapshot().Mode; got != "Plan: OFF" {
		t.Fatalf("StatusSnapshot().Mode = %q, want Plan: OFF", got)
	}

	agent.PlanModeEnabled = true
	if got := adapter.StatusSnapshot().Mode; got != "Plan: ON" {
		t.Fatalf("StatusSnapshot().Mode = %q, want Plan: ON", got)
	}
}

func TestTUIAdapter_CancelAndCleanup(t *testing.T) {
	agent := &Agent{}
	adapter := NewTUIAdapter(agent, nil)

	cancelled := false
	agent.cancelFunc = func() { cancelled = true }
	adapter.Cancel()
	if !cancelled {
		t.Fatal("Cancel() should invoke agent cancelFunc")
	}
	if agent.lastCancelReason != "user cancelled" {
		t.Fatalf("lastCancelReason = %q, want %q", agent.lastCancelReason, "user cancelled")
	}

	cleanupCalled := false
	cleanupHook = func() { cleanupCalled = true }
	t.Cleanup(func() { cleanupHook = nil })

	adapter.Cleanup()
	if !cleanupCalled {
		t.Fatal("Cleanup() should delegate to agent cleanup")
	}
}

func TestTUIAdapter_CopyTextAndCopyLastOutput(t *testing.T) {
	oldClipboardWriteAll := clipboardWriteAll
	t.Cleanup(func() { clipboardWriteAll = oldClipboardWriteAll })

	var copied []string
	clipboardWriteAll = func(text string) error {
		copied = append(copied, text)
		return nil
	}

	agent := &Agent{agentConversationState: agentConversationState{lastOutputs: []string{"line1\nline2"}}}
	adapter := NewTUIAdapter(agent, nil)

	if err := adapter.CopyText("hello"); err != nil {
		t.Fatalf("CopyText() error = %v", err)
	}
	msg, err := adapter.CopyLastOutput()
	if err != nil {
		t.Fatalf("CopyLastOutput() error = %v", err)
	}
	if msg != "Copied 2 lines" {
		t.Fatalf("CopyLastOutput() message = %q, want %q", msg, "Copied 2 lines")
	}
	if len(copied) != 2 || copied[0] != "hello" || copied[1] != "line1\nline2" {
		t.Fatalf("clipboard writes = %v, want [hello line1\\nline2]", copied)
	}
}

func TestTUIAdapter_CopyLastOutput_ErrorPaths(t *testing.T) {
	adapter := NewTUIAdapter(&Agent{}, nil)
	if _, err := adapter.CopyLastOutput(); err == nil || !strings.Contains(err.Error(), "no AI output") {
		t.Fatalf("CopyLastOutput() error = %v, want no output error", err)
	}

	oldClipboardWriteAll := clipboardWriteAll
	t.Cleanup(func() { clipboardWriteAll = oldClipboardWriteAll })
	clipboardWriteAll = func(text string) error {
		return errors.New("clipboard unavailable")
	}

	adapter = NewTUIAdapter(&Agent{agentConversationState: agentConversationState{lastOutputs: []string{"result"}}}, nil)
	if _, err := adapter.CopyLastOutput(); err == nil || !strings.Contains(err.Error(), "clipboard unavailable") {
		t.Fatalf("CopyLastOutput() error = %v, want clipboard error", err)
	}
}

func TestTUIAdapter_LoadConfigForEditReturnsClone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultModel = "gpt-before"
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	adapter := NewTUIAdapter(&Agent{}, nil)
	loaded, err := adapter.LoadConfigForEdit()
	if err != nil {
		t.Fatalf("LoadConfigForEdit() error = %v", err)
	}

	loaded.DefaultModel = "gpt-edited"
	reloaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if reloaded.DefaultModel != "gpt-before" {
		t.Fatalf("LoadConfigForEdit() should return clone, file model = %q", reloaded.DefaultModel)
	}
}

func TestTUIAdapter_SaveAndSyncConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-old"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-old"})

	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := NewAgentWithRuntime("gpt-old", &MockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	adapter := NewTUIAdapter(agent, nil)
	next := config.CloneConfig(cfg)
	next.DefaultModel = "gpt-new"
	next.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-new"})

	if err := adapter.SaveAndSyncConfig(next); err != nil {
		t.Fatalf("SaveAndSyncConfig() error = %v", err)
	}

	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.DefaultModel != "gpt-new" {
		t.Fatalf("saved DefaultModel = %q, want %q", loaded.DefaultModel, "gpt-new")
	}
	if agent.cfg().DefaultModel != "gpt-new" {
		t.Fatalf("runtime DefaultModel = %q, want %q", agent.cfg().DefaultModel, "gpt-new")
	}
	if agent.CurrentModel != "gpt-new" {
		t.Fatalf("agent.CurrentModel = %q, want %q", agent.CurrentModel, "gpt-new")
	}
}
