package agent

import (
	"strings"
	"testing"

	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestSyncCurrentDerivedRuntimeState_ModelDrivenEditMode(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{
		ProviderName:    "openrouter",
		CurrentModel:    "anthropic/claude-sonnet-4-6",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "openrouter"},
		Runtime:         runtime,
	}

	agent.registry().SetExcludedTools(newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).excluded())
	agent.syncCurrentDerivedRuntimeState()

	if !strings.Contains(agent.SystemPrompt, "### Legacy edit tools") {
		t.Fatalf("system prompt should rebuild for legacy model-driven mode, got:\n%s", agent.SystemPrompt)
	}
	if strings.Contains(agent.SystemPrompt, "### apply_patch (edit tool)") {
		t.Fatalf("legacy model-driven mode should not keep apply_patch guide, got:\n%s", agent.SystemPrompt)
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

func TestSyncCurrentDerivedRuntimeState_ProviderDrivenEditMode(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{
		ProviderName:    "claude",
		CurrentModel:    "claude-sonnet-4-6",
		CurrentProvider: &mockCacheClearableProvider{name: "claude"},
		Runtime:         runtime,
	}

	agent.registry().SetExcludedTools(newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).excluded())
	agent.syncCurrentDerivedRuntimeState()

	if !strings.Contains(agent.SystemPrompt, "### Claude-specific") {
		t.Fatalf("system prompt should rebuild for current provider, got:\n%s", agent.SystemPrompt)
	}
	if !strings.Contains(agent.SystemPrompt, "### Legacy edit tools") {
		t.Fatalf("provider-driven legacy mode should rebuild legacy guide, got:\n%s", agent.SystemPrompt)
	}

	excluded := agent.registry().GetExcludedTools()
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatalf("provider-driven legacy mode should exclude apply_patch, got %v", excluded)
	}
	for _, name := range []string{"search_code", "read_file", "str_replace", "write_file", "delete_file"} {
		if toolNameInList(excluded, name) {
			t.Fatalf("provider-driven legacy mode should expose %s, got excluded %v", name, excluded)
		}
	}
}

func TestSyncCurrentDerivedRuntimeState_PlanModeUsesPlanSurface(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "openai"},
		PlanModeEnabled: true,
		Runtime:         runtime,
	}

	agent.registry().SetExcludedTools(newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).excluded())
	agent.syncCurrentDerivedRuntimeState()

	excluded := agent.registry().GetExcludedTools()
	if toolNameInList(excluded, "ask_user_question") {
		t.Fatalf("plan mode should expose ask_user_question after derived-state sync, got %v", excluded)
	}
	if toolNameInList(excluded, "read_file") {
		t.Fatalf("plan mode should keep read_file visible as exact-control override, got %v", excluded)
	}
}

func TestSyncCurrentDerivedRuntimeState_PreservesRuntimeSpecificExclusions(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{
		ProviderName:    "openrouter",
		CurrentModel:    "anthropic/claude-sonnet-4-6",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "openrouter"},
		Runtime:         runtime,
	}

	agent.registry().SetExcludedTools(append(
		newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).excluded(),
		"read_file",
		"mcp_github_get_issue",
	))
	agent.syncCurrentDerivedRuntimeState()

	excluded := agent.registry().GetExcludedTools()
	for _, name := range []string{"read_file", "mcp_github_get_issue"} {
		if !toolNameInList(excluded, name) {
			t.Fatalf("derived-state sync should preserve runtime-specific exclusion for %s, got %v", name, excluded)
		}
	}
	if !toolNameInList(excluded, "apply_patch") {
		t.Fatalf("legacy model-driven mode should still exclude apply_patch, got %v", excluded)
	}
}

func TestSyncCurrentDerivedRuntimeState_RebuildKeepsSkillsCatalog(t *testing.T) {
	withTempWorkdir(t)
	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()
	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog {
		return agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{
				{Name: "demo", Description: "demo skill"},
			},
		}
	}

	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "openai"},
		Runtime:         runtime,
	}

	agent.syncCurrentDerivedRuntimeState()

	if !strings.Contains(agent.SystemPrompt, "SKILLS_CATALOG_START") {
		t.Fatalf("skills catalog block should remain after base rebuild:\n%s", agent.SystemPrompt)
	}
	if !strings.Contains(agent.SystemPrompt, "- demo: demo skill") {
		t.Fatalf("skills catalog entry missing after base rebuild:\n%s", agent.SystemPrompt)
	}
}
