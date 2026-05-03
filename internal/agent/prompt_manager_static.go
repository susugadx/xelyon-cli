package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

type promptStaticBuildInput struct {
	invocationCWD      string
	projectConfigBlock string
}

func (m *PromptManager) buildStaticPromptForRefresh(currentStatic string, input promptStaticBuildInput) string {
	a := m.agent
	if a == nil {
		return strings.TrimRight(currentStatic, "\n")
	}

	staticPrompt := prompt.StripProjectConfigSections(currentStatic)
	if strings.TrimSpace(staticPrompt) == "" {
		return m.buildStaticPrompt(input)
	}

	// 既存 static を起点に整形することで custom prompt (例: SubAgentPrompt) を保持する。
	staticPrompt = m.decorateStaticPrompt(staticPrompt, input)
	return m.wrapStaticPromptForProvider(staticPrompt)
}

func (m *PromptManager) buildStaticPrompt(input promptStaticBuildInput) string {
	a := m.agent
	if a == nil {
		return ""
	}

	staticPrompt := m.baseStaticPrompt()
	staticPrompt = m.decorateStaticPrompt(staticPrompt, input)
	return m.wrapStaticPromptForProvider(staticPrompt)
}

func (m *PromptManager) baseStaticPrompt() string {
	a := m.agent
	if a == nil {
		return ""
	}
	providerName := resolvePromptProviderName(a)
	staticPrompt := prompt.GetSystemPromptForProviderWithConfig(providerName, a.CurrentModel, a.cfg())
	if a.mcpManager != nil && len(a.mcpManager.GetTools()) > 0 {
		staticPrompt += buildMCPToolsPrompt(a.mcpManager)
	}
	return staticPrompt
}

func (m *PromptManager) decorateStaticPrompt(staticPrompt string, input promptStaticBuildInput) string {
	staticPrompt = injectSkillCatalogPrompt(staticPrompt, input.invocationCWD)
	if block := strings.TrimSpace(input.projectConfigBlock); block != "" {
		staticPrompt = prompt.InjectProjectConfigBlock(staticPrompt, block)
	}
	return staticPrompt
}

func (m *PromptManager) wrapStaticPromptForProvider(staticPrompt string) string {
	a := m.agent
	if a == nil {
		return strings.TrimRight(staticPrompt, "\n")
	}
	providerName := resolvePromptProviderName(a)
	if strings.TrimSpace(providerName) == "" {
		return strings.TrimRight(staticPrompt, "\n")
	}
	return prompt.BuildProviderSystemPromptWithConfig(staticPrompt, providerName, a.CurrentModel, a.cfg())
}

func resolvePromptProviderName(a *Agent) string {
	if a == nil {
		return ""
	}
	providerName := strings.TrimSpace(a.ProviderName)
	if providerName == "" && a.CurrentProvider != nil {
		providerName = providerRuntimeNameFromProvider(a.CurrentProvider)
	}
	return providerName
}

func (m *PromptManager) rebuildProjectOnlyStaticPrompt() {
	a := m.agent
	if a == nil {
		return
	}
	layout := api.SplitSystemPromptLayout(a.SystemPrompt)
	refreshedStatic := m.buildStaticPromptForRefresh(layout.Static, promptStaticBuildInput{
		invocationCWD: a.invocationCWD(),
	})
	layout.SetStatic(refreshedStatic)

	withoutProjectInstructions := prompt.StripProjectConfigSections(layout.Compose())
	a.SystemPrompt = stripProjectMapSection(withoutProjectInstructions)
}
