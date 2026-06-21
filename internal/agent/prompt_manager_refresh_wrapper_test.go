package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/prompt"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestRefreshProjectPrompt_DoesNotAddProviderWrapperWhenDefaultEmpty(t *testing.T) {
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()
	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog { return agentskills.SkillCatalog{} }

	cfg := newProjectMapDisabledConfig()
	agent := &Agent{
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		CurrentProvider: &MockProvider{name: "gemini"},
		ProviderName:    "gemini",
		CurrentModel:    "gemini-3.1-pro-preview-customtools",
		SystemPrompt:    prompt.GetSystemPromptForProviderWithConfig("gemini", "gemini-3.1-pro-preview-customtools", cfg),
	}
	if strings.Contains(agent.SystemPrompt, "## Provider Notes") {
		t.Fatalf("test setup should start from unwrapped prompt:\n%s", agent.SystemPrompt)
	}

	agent.refreshProjectPrompt("")

	if strings.Contains(agent.SystemPrompt, "## Provider Notes") {
		t.Fatalf("refresh should not add provider notes by default:\n%s", agent.SystemPrompt)
	}
}

func TestRefreshProjectPrompt_StripsStaleProviderWrapperWhenDefaultEmpty(t *testing.T) {
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()
	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog { return agentskills.SkillCatalog{} }

	cfg := newProjectMapDisabledConfig()
	base := prompt.GetSystemPromptForProviderWithConfig("gemini", "gemini-3.1-pro-preview-customtools", cfg)
	staleWrapped := "<!-- PROVIDER_NOTES_START:gemini -->\n" +
		"## Provider Notes\n" +
		"### Gemini-specific\n" +
		"- Emit native tool calls directly; do not serialize tool calls inside markdown code blocks\n" +
		"<!-- PROVIDER_NOTES_END -->\n\n" +
		base

	agent := &Agent{
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		CurrentProvider: &MockProvider{name: "gemini"},
		ProviderName:    "gemini",
		CurrentModel:    "gemini-3.1-pro-preview-customtools",
		SystemPrompt:    staleWrapped,
	}

	agent.refreshProjectPrompt("")

	if strings.Contains(agent.SystemPrompt, "## Provider Notes") {
		t.Fatalf("refresh should strip stale provider notes when default is empty:\n%s", agent.SystemPrompt)
	}
}
