package agent

import (
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

const promptSkillCatalogMaxEntries = 24

var loadSkillCatalogForAgent = func(invocationCWD string) agentskills.SkillCatalog {
	return agentskills.LoadCatalogForInvocationCWD(invocationCWD)
}

func injectSkillCatalogPrompt(systemPrompt string, invocationCWD string) string {
	catalog := loadSkillCatalogForAgent(invocationCWD)
	return agentskills.InjectPromptCatalog(systemPrompt, catalog, promptSkillCatalogMaxEntries)
}

func (a *Agent) loadSkillCatalog() agentskills.SkillCatalog {
	return loadSkillCatalogForAgent(a.invocationCWD())
}

func (a *Agent) SkillCatalog() agentskills.SkillCatalog {
	return a.loadSkillCatalog()
}
