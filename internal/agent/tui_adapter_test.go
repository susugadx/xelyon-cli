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

func TestTUIAdapter_ProjectFallsThroughToTUILocalRouter(t *testing.T) {
	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	for _, input := range []string{"/project", "/project rules"} {
		if adapter.HandleCommand(input) {
			t.Fatalf("HandleCommand(%q) = true, want false", input)
		}
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q, want no adapter output", out.String())
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
	if !strings.Contains(got, "/project") {
		t.Fatalf("TUI /help should include /project:\n%s", got)
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

func TestTUIAdapter_SaveProjectConfigSyncsProjectFinalChecks(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	globalCfg := newProjectMapDisabledConfig()
	globalCfg.FinalChecks.Commands = []string{"global verify"}
	globalCfg.FinalChecks.Timeout = 900
	if err := config.SaveConfig(globalCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, _ := newTUIAdapterTestAgent(t)
	agent.cfg().FinalChecks = config.FinalChecksConfig{
		Commands: []string{"old project verify"},
		Timeout:  30,
	}
	adapter := NewTUIAdapter(agent, nil)

	projectPath := filepath.Join(t.TempDir(), "xelyon.yaml")
	pc := &config.ProjectConfig{
		Context: "ctx",
		FinalChecks: &config.FinalChecksConfig{
			Commands: []string{"project verify"},
			Timeout:  120,
		},
		FilePath: projectPath,
	}
	if err := adapter.SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error = %v", err)
	}

	got := agent.cfg().FinalChecks
	if len(got.Commands) != 1 || got.Commands[0] != "project verify" {
		t.Fatalf("runtime FinalChecks.Commands = %#v, want project verify", got.Commands)
	}
	if got.Timeout != 120 {
		t.Fatalf("runtime FinalChecks.Timeout = %d, want 120", got.Timeout)
	}
}

func TestTUIAdapter_SaveProjectConfigFallsBackToGlobalFinalChecks(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	globalCfg := newProjectMapDisabledConfig()
	globalCfg.FinalChecks.Commands = []string{"global verify"}
	globalCfg.FinalChecks.Timeout = 900
	if err := config.SaveConfig(globalCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, _ := newTUIAdapterTestAgent(t)
	agent.cfg().FinalChecks = config.FinalChecksConfig{
		Commands: []string{"stale project verify"},
		Timeout:  30,
	}
	adapter := NewTUIAdapter(agent, nil)

	projectPath := filepath.Join(t.TempDir(), "xelyon.yaml")
	pc := &config.ProjectConfig{
		Context:  "ctx",
		FilePath: projectPath,
	}
	if err := adapter.SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error = %v", err)
	}

	got := agent.cfg().FinalChecks
	if len(got.Commands) != 1 || got.Commands[0] != "global verify" {
		t.Fatalf("runtime FinalChecks.Commands = %#v, want global verify", got.Commands)
	}
	if got.Timeout != 900 {
		t.Fatalf("runtime FinalChecks.Timeout = %d, want 900", got.Timeout)
	}
}

func TestTUIAdapter_SaveProjectConfigEmptyOverrideDisablesGlobalFinalChecks(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	globalCfg := newProjectMapDisabledConfig()
	globalCfg.FinalChecks.Commands = []string{"global verify"}
	globalCfg.FinalChecks.Timeout = 900
	if err := config.SaveConfig(globalCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, _ := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	projectPath := filepath.Join(t.TempDir(), "xelyon.yaml")
	pc := &config.ProjectConfig{
		Context:     "ctx",
		FinalChecks: &config.FinalChecksConfig{},
		FilePath:    projectPath,
	}
	if err := adapter.SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error = %v", err)
	}

	got := agent.cfg().FinalChecks
	if len(got.Commands) != 0 {
		t.Fatalf("runtime FinalChecks.Commands = %#v, want empty project override", got.Commands)
	}
}
