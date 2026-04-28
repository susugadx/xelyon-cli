package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func TestPrintHelpToWriter_DefaultSurfaceShowsCurrentVisibleTools(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if !strings.Contains(got, "Built-in tools available in current surface (phase: normal, gather_context default + read_file exact-control override):") {
		t.Fatalf("default help should describe the current apply_patch surface, got:\n%s", got)
	}
	for _, fragment := range []string{
		"gather_context    - Primary/default investigation tool",
		"read_file         - Exact file reader for edit/apply_patch exact-control override",
		"apply_patch       - Primary edit tool for precise patch-based file changes",
		"spawn_agent       - Spawn a sub-agent for explore/edit/verify tasks",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("default help missing %q\noutput:\n%s", fragment, got)
		}
	}
	for _, hidden := range []string{"search_code", "list_dir", "ask_user_question", "str_replace"} {
		if strings.Contains(got, hidden) {
			t.Fatalf("default help should not advertise %s\noutput:\n%s", hidden, got)
		}
	}
}

func TestPrintHelpToWriter_LegacySurfaceShowsOverrides(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("claude-sonnet-4-6", &mockProvider{name: "claude"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if !strings.Contains(got, "Built-in tools available in current surface (phase: normal, gather_context default + search_code/read_file low-level overrides):") {
		t.Fatalf("legacy help should describe the legacy override surface, got:\n%s", got)
	}
	for _, fragment := range []string{
		"gather_context    - Primary/default investigation tool",
		"search_code       - Low-level code search override on legacy surfaces when exposed",
		"read_file         - Low-level exact file reader override when exposed",
		"str_replace       - Precise same-file replacements from actual tool output",
		"write_file        - Legacy edit tool to create or overwrite a file",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("legacy help missing %q\noutput:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "apply_patch") {
		t.Fatalf("legacy help should not advertise apply_patch\noutput:\n%s", got)
	}
}

func TestPrintHelpToWriter_PlanModeShowsPlanningSurface(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)
	agent.PlanModeEnabled = true

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if !strings.Contains(got, "Built-in tools available in current surface (phase: plan, gather_context default + read_file exact-control override):") {
		t.Fatalf("plan help should describe the plan surface, got:\n%s", got)
	}
	if !strings.Contains(got, "ask_user_question - Ask the user a clarification question during plan investigation") {
		t.Fatalf("plan help should advertise ask_user_question\noutput:\n%s", got)
	}
}

func TestPrintHelpToWriter_AfterModelSwitchUsesCurrentModeVisibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{
		ProviderName:    "openrouter",
		CurrentModel:    "openai/gpt-5.4",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "openrouter"},
		SystemPrompt:    prompt.BuildProviderSystemPromptWithConfig(prompt.SystemPrompt, "openrouter", "openai/gpt-5.4", cfg),
		Runtime:         runtime,
	}
	agent.syncCurrentSurfaceToolVisibility()

	if !handleModelCommand(agent, []string{"anthropic/claude-sonnet-4-6"}) {
		t.Fatal("handleModelCommand() = false, want true")
	}

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if !strings.Contains(got, "Built-in tools available in current surface (phase: normal, gather_context default + search_code/read_file low-level overrides):") {
		t.Fatalf("help should describe the legacy surface after model switch, got:\n%s", got)
	}
	for _, fragment := range []string{
		"search_code       - Low-level code search override on legacy surfaces when exposed",
		"str_replace       - Precise same-file replacements from actual tool output",
		"write_file        - Legacy edit tool to create or overwrite a file",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("help missing %q after model switch\noutput:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "apply_patch       - ") {
		t.Fatalf("help should not advertise apply_patch after legacy model switch\noutput:\n%s", got)
	}
}

func TestPrintHelpToWriter_AfterProviderSwitchUsesCurrentModeVisibility(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		CurrentProvider: &mockCacheClearableProvider{name: "openai"},
		SystemPrompt:    prompt.BuildProviderSystemPromptWithConfig(prompt.SystemPrompt, "openai", "gpt-5.4", cfg),
		Runtime:         runtime,
	}
	agent.syncCurrentSurfaceToolVisibility()

	if err := agent.SwitchProvider("anthropic"); err != nil {
		t.Fatalf("SwitchProvider() error = %v", err)
	}

	var out bytes.Buffer
	printHelpToWriter(&out, agent)
	got := out.String()

	if !strings.Contains(got, "Built-in tools available in current surface (phase: normal, gather_context default + search_code/read_file low-level overrides):") {
		t.Fatalf("help should describe the legacy surface after provider switch, got:\n%s", got)
	}
	for _, fragment := range []string{
		"search_code       - Low-level code search override on legacy surfaces when exposed",
		"str_replace       - Precise same-file replacements from actual tool output",
		"write_file        - Legacy edit tool to create or overwrite a file",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("help missing %q after provider switch\noutput:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "apply_patch       - ") {
		t.Fatalf("help should not advertise apply_patch after legacy provider switch\noutput:\n%s", got)
	}
}
