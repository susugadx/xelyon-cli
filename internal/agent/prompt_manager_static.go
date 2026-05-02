package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
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

	// テスト/軽量実行コンテキストでは provider が未設定な場合があるため、
	// その場合は既存 static を起点に正規化のみ行う。
	if strings.TrimSpace(a.ProviderName) == "" && a.CurrentProvider == nil {
		staticPrompt := prompt.StripProjectConfigSections(currentStatic)
		staticPrompt = injectSkillCatalogPrompt(staticPrompt, input.invocationCWD)
		if block := strings.TrimSpace(input.projectConfigBlock); block != "" {
			staticPrompt = prompt.InjectProjectConfigBlock(staticPrompt, block)
		}
		return strings.TrimRight(staticPrompt, "\n")
	}

	return m.buildStaticPrompt(input)
}

func (m *PromptManager) buildStaticPrompt(input promptStaticBuildInput) string {
	a := m.agent
	if a == nil {
		return ""
	}

	providerName := resolvePromptProviderName(a)
	staticPrompt := prompt.GetSystemPromptForProviderWithConfig(providerName, a.CurrentModel, a.cfg())
	if a.mcpManager != nil && len(a.mcpManager.GetTools()) > 0 {
		staticPrompt += buildMCPToolsPrompt(a.mcpManager)
	}

	staticPrompt = injectSkillCatalogPrompt(staticPrompt, input.invocationCWD)
	if block := strings.TrimSpace(input.projectConfigBlock); block != "" {
		staticPrompt = prompt.InjectProjectConfigBlock(staticPrompt, block)
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

func buildProjectConfigPromptBlock(pc *config.ProjectConfig, input string) string {
	if pc == nil {
		return ""
	}
	selection := config.SelectProjectPromptSelection(pc, input)
	return prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts)
}
