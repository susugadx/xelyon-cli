package agent

import (
	"context"
	"strings"
	"testing"

	toolsubagent "github.com/susugadx/xelyon-cli/internal/tools/subagent"
)

func TestRunHeadlessWithConfig_SubAgentModeUsesSubPromptAndExcludesSubAgentTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.SubAgentPrompt = toolsubagent.ExplorePrompt
	provider := &headlessToolSetProbeProvider{}

	result := RunHeadlessWithConfig(context.Background(), "probe", "gpt-5.4-nano", provider, cfg)
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if !strings.Contains(provider.systemPrompt, "You are a sub-agent executing a specific exploration task assigned by the orchestrator.") {
		t.Fatalf("systemPrompt should contain sub-agent prompt, got %q", provider.systemPrompt)
	}
	if strings.Contains(provider.systemPrompt, "## Core Identity") {
		t.Fatalf("systemPrompt should not keep the parent prompt, got %q", provider.systemPrompt)
	}
	for _, name := range []string{"spawn_agent", "wait_agent", "ask_user_question"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("sub-agent headless mode should exclude %s", name)
		}
	}
}

func TestRunHeadlessWithConfig_NormalHeadlessKeepsSubAgentTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	provider := &headlessToolSetProbeProvider{}

	result := RunHeadlessWithConfig(context.Background(), "probe", "gpt-5.4", provider, cfg)
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	for _, name := range []string{"spawn_agent", "wait_agent"} {
		if !toolNameInList(provider.toolNames, name) {
			t.Fatalf("normal headless mode should expose %s", name)
		}
	}
	if toolNameInList(provider.toolNames, "ask_user_question") {
		t.Fatal("normal headless mode should still exclude planning tools")
	}
}

func TestRunHeadlessWithConfig_DefaultEditToolVisibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := runHeadlessToolSetProbe(t, "", "gpt-5.4")
	assertHeadlessApplyPatchEditSurface(t, provider, "default headless mode")
	if strings.Contains(provider.systemPrompt, "search_code: code discovery tool") {
		t.Fatal("default headless system prompt should not advertise hidden search_code")
	}
	if !strings.Contains(provider.systemPrompt, "read_file: exact-content reader for edit/apply_patch exact-control override") {
		t.Fatal("default headless system prompt should keep read_file exact-control guidance")
	}
}

func TestRunHeadlessWithConfig_ProviderResolvedLegacyEditToolVisibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tests := []struct {
		name     string
		provider string
		model    string
	}{
		{name: "claude", provider: "claude", model: "claude-sonnet-4-6"},
		{name: "kimi", provider: "kimi", model: "kimi-k2.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := runHeadlessToolSetProbe(t, tt.provider, tt.model)
			assertHeadlessLegacyEditSurface(t, provider, tt.provider+" headless mode")
		})
	}
}

func TestRunHeadlessWithConfig_OpenRouterKnownApplyPatchFamilyVisibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := runHeadlessToolSetProbe(t, "openrouter", "google/gemini-2.5-pro")
	assertHeadlessApplyPatchEditSurface(t, provider, "openrouter google family")
}

func TestRunHeadlessWithConfig_EnvResolvedLegacyEditToolVisibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := runHeadlessToolSetProbe(t, "", "gpt-5.4")
	assertHeadlessLegacyEditSurface(t, provider, "legacy headless mode")
	for _, name := range []string{"search_code", "read_file"} {
		if !toolNameInList(provider.toolNames, name) {
			t.Fatalf("legacy headless mode should expose %s", name)
		}
	}
	if toolNameInList(provider.toolNames, "list_dir") {
		t.Fatal("legacy headless mode should keep list_dir hidden")
	}
}

func runHeadlessToolSetProbe(t *testing.T, providerName string, model string) *headlessToolSetProbeProvider {
	t.Helper()

	provider := &headlessToolSetProbeProvider{name: providerName}
	result := RunHeadlessWithConfig(context.Background(), "probe", model, provider, newProjectMapDisabledConfig())
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	return provider
}

func assertHeadlessApplyPatchEditSurface(t *testing.T, provider *headlessToolSetProbeProvider, label string) {
	t.Helper()

	if !toolNameInList(provider.toolNames, "apply_patch") {
		t.Fatalf("%s should expose apply_patch", label)
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("%s should exclude %s", label, name)
		}
	}
	if !strings.Contains(provider.systemPrompt, "### apply_patch (edit tool)") {
		t.Fatalf("%s system prompt should use apply_patch guidance", label)
	}
}

func assertHeadlessLegacyEditSurface(t *testing.T, provider *headlessToolSetProbeProvider, label string) {
	t.Helper()

	if toolNameInList(provider.toolNames, "apply_patch") {
		t.Fatalf("%s should exclude apply_patch", label)
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if !toolNameInList(provider.toolNames, name) {
			t.Fatalf("%s should expose %s", label, name)
		}
	}
	if !strings.Contains(provider.systemPrompt, "### Legacy edit tools") {
		t.Fatalf("%s system prompt should use legacy edit tool guidance", label)
	}
}
