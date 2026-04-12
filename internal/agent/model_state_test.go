package agent

import (
	"strings"
	"testing"
)

func TestSetCurrentModelAndSync_UpdatesSessionStatsAndSurface(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openrouter"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	agent.setCurrentModelAndSync("anthropic/claude-sonnet-4-6")

	if agent.CurrentModel != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "anthropic/claude-sonnet-4-6")
	}
	if agent.session == nil || agent.session.Model != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "anthropic/claude-sonnet-4-6")
	}
	if agent.Stats == nil || agent.Stats.Model != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("Stats.Model = %q, want %q", agent.Stats.Model, "anthropic/claude-sonnet-4-6")
	}
	if !strings.Contains(agent.SystemPrompt, "### Legacy edit tools") {
		t.Fatalf("system prompt should rebuild for model-driven legacy mode, got:\n%s", agent.SystemPrompt)
	}

	excluded := agent.registry().GetExcludedTools()
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatalf("legacy model-driven mode should exclude apply_patch, got %v", excluded)
	}
	for _, name := range []string{"search_code", "read_file", "str_replace", "write_file", "delete_file"} {
		if toolNameInList(excluded, name) {
			t.Fatalf("legacy model-driven mode should expose %s, got excluded %v", name, excluded)
		}
	}
}
