package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newTUIAdapterTestAgent(t *testing.T) (*Agent, *bytes.Buffer) {
	t.Helper()

	cfg := newProjectMapDisabledConfig()
	out := &bytes.Buffer{}
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), out, out)

	agent := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	return agent, out
}

func TestTUIConfig_UnknownSubcommand_DoesNotRunInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/config foo") {
		t.Fatal("HandleCommand(/config foo) = false, want true")
	}

	got := out.String()
	if !strings.Contains(got, "/config foo is not available in TUI mode") {
		t.Fatalf("output = %q, want unsupported TUI /config message", got)
	}
	if strings.Contains(got, "Configuration Menu") {
		t.Fatalf("output = %q, should not contain interactive config menu", got)
	}
}

func TestTUIConfig_Show_StillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/config show") {
		t.Fatal("HandleCommand(/config show) = false, want true")
	}

	got := out.String()
	if !strings.Contains(got, "default_provider                    = openai") {
		t.Fatalf("output = %q, want config show output", got)
	}
}

func TestTUIConfig_Model_StillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-old"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-old"})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/config model gpt-new") {
		t.Fatal("HandleCommand(/config model gpt-new) = false, want true")
	}

	if agent.CurrentModel != "gpt-new" {
		t.Fatalf("agent.CurrentModel = %q, want %q", agent.CurrentModel, "gpt-new")
	}
	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.DefaultModel != "gpt-new" {
		t.Fatalf("loaded.DefaultModel = %q, want %q", loaded.DefaultModel, "gpt-new")
	}
	if loaded.ProviderModels["openai"].DefaultModel != "gpt-new" {
		t.Fatalf("loaded.ProviderModels[openai].DefaultModel = %q, want %q", loaded.ProviderModels["openai"].DefaultModel, "gpt-new")
	}
	if !strings.Contains(out.String(), "Default model updated to: gpt-new") {
		t.Fatalf("output = %q, want success message", out.String())
	}
}

func TestTUIAdapter_NonBareReviewFallsThrough(t *testing.T) {
	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if adapter.HandleCommand("/review staged") {
		t.Fatal("HandleCommand(/review staged) = true, want false")
	}
	if strings.Contains(out.String(), "/review is available in TUI mode only") {
		t.Fatalf("output = %q, should not contain TUI-only review warning", out.String())
	}
}

func TestTUIAdapter_HelpUsesTUISurface(t *testing.T) {
	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/help") {
		t.Fatal("HandleCommand(/help) = false, want true")
	}

	got := out.String()
	if !strings.Contains(got, "/review") {
		t.Fatalf("TUI /help should include /review:\n%s", got)
	}
	if !strings.Contains(got, "/init") {
		t.Fatalf("TUI /help should include /init:\n%s", got)
	}
	if strings.Contains(got, "/project") {
		t.Fatalf("TUI /help should not include classic-only /project:\n%s", got)
	}
}

func TestTUIAdapter_InitCreatesTemplateWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/init") {
		t.Fatal("HandleCommand(/init) = false, want true")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "xelyon.yaml")); err != nil {
		t.Fatalf("xelyon.yaml should be created: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "not available in TUI mode") {
		t.Fatalf("/init should not be blocked in TUI:\n%s", got)
	}
	if !strings.Contains(got, "xelyon.yaml template created") {
		t.Fatalf("output missing created message:\n%s", got)
	}
}

func TestTUIAdapter_InitDoesNotPromptOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	existingPath := filepath.Join(tmpDir, "xelyon.yaml")
	const existing = "existing: true\n"
	if err := os.WriteFile(existingPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/init") {
		t.Fatal("HandleCommand(/init) = false, want true")
	}

	data, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != existing {
		t.Fatalf("xelyon.yaml was overwritten unexpectedly: %q", string(data))
	}
	got := out.String()
	if strings.Contains(got, "Overwrite?") {
		t.Fatalf("TUI /init should not prompt for overwrite:\n%s", got)
	}
	if !strings.Contains(got, "Not overwriting from TUI mode") {
		t.Fatalf("output missing non-overwrite message:\n%s", got)
	}
}

func TestTUIAdapter_ProjectRemainsUnavailableInTUI(t *testing.T) {
	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/project") {
		t.Fatal("HandleCommand(/project) = false, want true")
	}

	got := out.String()
	if !strings.Contains(got, "/project is not available in TUI mode") {
		t.Fatalf("output missing project unavailable message:\n%s", got)
	}
}
