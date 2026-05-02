package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/prompt"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestRefreshProjectPrompt_ReappliesProviderWrapperWhenMissing(t *testing.T) {
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()
	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog { return agentskills.SkillCatalog{} }

	cfg := newProjectMapDisabledConfig()
	agent := &Agent{
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		CurrentProvider: &MockProvider{name: "openai"},
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		SystemPrompt:    prompt.GetSystemPromptForProviderWithConfig("openai", "gpt-5.4", cfg),
	}
	if strings.Contains(agent.SystemPrompt, "## Provider Notes") {
		t.Fatalf("test setup should start from unwrapped prompt:\n%s", agent.SystemPrompt)
	}

	agent.refreshProjectPrompt("")

	if !strings.Contains(agent.SystemPrompt, "## Provider Notes") {
		t.Fatalf("refresh should preserve or reapply provider wrapper:\n%s", agent.SystemPrompt)
	}
	if !strings.Contains(agent.SystemPrompt, "### OpenAI-specific") {
		t.Fatalf("refresh should keep openai provider notes:\n%s", agent.SystemPrompt)
	}
}

func TestRefreshProjectPrompt_DoesNotDuplicateProviderWrapper(t *testing.T) {
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()
	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog { return agentskills.SkillCatalog{} }

	cfg := newProjectMapDisabledConfig()
	base := prompt.GetSystemPromptForProviderWithConfig("openai", "gpt-5.4", cfg)
	wrapped := prompt.BuildProviderSystemPromptWithConfig(base, "openai", "gpt-5.4", cfg)

	agent := &Agent{
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		CurrentProvider: &MockProvider{name: "openai"},
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		SystemPrompt:    wrapped,
	}

	agent.refreshProjectPrompt("")

	if strings.Count(agent.SystemPrompt, "## Provider Notes") != 1 {
		t.Fatalf("refresh should not duplicate provider notes:\n%s", agent.SystemPrompt)
	}
}
