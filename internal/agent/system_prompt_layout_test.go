package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestSystemPromptLayout_ComposeNormalizesBoundaryCount(t *testing.T) {
	input := "base prompt" + api.SystemPromptCacheBoundary + "dynamic-a" + api.SystemPromptCacheBoundary + "dynamic-b\n"
	layout := parseSystemPromptLayout(input)

	if layout.Static != "base prompt" {
		t.Fatalf("layout.Static = %q, want base prompt", layout.Static)
	}
	if !strings.Contains(layout.Dynamic, "dynamic-a") || !strings.Contains(layout.Dynamic, "dynamic-b") {
		t.Fatalf("layout.Dynamic should keep all dynamic fragments: %q", layout.Dynamic)
	}

	out := layout.Compose()
	if strings.Count(out, api.SystemPromptCacheBoundary) != 1 {
		t.Fatalf("Compose() should normalize boundary count to 1:\n%s", out)
	}
}

func TestRefreshProjectPrompt_KeepsSkillsInStaticBlockWhenPlanPromptExists(t *testing.T) {
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()
	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog {
		return agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{
				{Name: "demo", Description: "demo description"},
			},
		}
	}

	cfg := newProjectMapDisabledConfig()
	cfg.PromptCache.Enabled = true
	agent := &Agent{
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		CurrentProvider: &MockProvider{name: "openai"},
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		SystemPrompt:    injectSkillCatalogPrompt(prompt.GetSystemPromptForProviderWithConfig("openai", "gpt-5.4", cfg), ""),
	}

	layout := parseSystemPromptLayout(agent.SystemPrompt)
	layout.AppendDynamic(promptplan.BuildPlanningPrompt())
	agent.SystemPrompt = layout.Compose()

	agent.refreshProjectPrompt("")

	if strings.Count(agent.SystemPrompt, api.SystemPromptCacheBoundary) != 1 {
		t.Fatalf("refresh should keep single cache boundary:\n%s", agent.SystemPrompt)
	}

	field := api.BuildSystemFieldWithConfig(agent.SystemPrompt, cfg)
	blocks, ok := field.([]api.SystemBlock)
	if !ok {
		t.Fatalf("BuildSystemFieldWithConfig() type = %T, want []api.SystemBlock", field)
	}
	if len(blocks) != 2 {
		t.Fatalf("system blocks = %d, want 2 (static+dynamic)", len(blocks))
	}

	if !strings.Contains(blocks[0].Text, "SKILLS_CATALOG_START") {
		t.Fatalf("skills catalog should remain in static block:\n%s", blocks[0].Text)
	}
	if strings.Contains(blocks[1].Text, "SKILLS_CATALOG_START") {
		t.Fatalf("skills catalog should not move to dynamic block:\n%s", blocks[1].Text)
	}
	if !strings.Contains(blocks[1].Text, "You are in Plan Mode - producing a text plan") {
		t.Fatalf("planning prompt should remain in dynamic block:\n%s", blocks[1].Text)
	}
}
