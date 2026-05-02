package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
)

type PromptManager struct {
	agent *Agent
}

func newPromptManager(agent *Agent) *PromptManager {
	return &PromptManager{agent: agent}
}

func (a *Agent) promptManager() *PromptManager {
	return newPromptManager(a)
}

func (m *PromptManager) RebuildSystemPromptForCurrentProvider() {
	a := m.agent
	if a == nil || a.CurrentProvider == nil {
		return
	}

	planningPrompt := promptplan.BuildPlanningPrompt()
	prevLayout := parseSystemPromptLayout(a.SystemPrompt)
	hadPlanPrompt := strings.Contains(prevLayout.Static, planningPrompt) || strings.Contains(prevLayout.Dynamic, planningPrompt)

	providerName := a.ProviderName
	if providerName == "" {
		providerName = providerRuntimeNameFromProvider(a.CurrentProvider)
	}
	systemPrompt := prompt.GetSystemPromptForProviderWithConfig(providerName, a.CurrentModel, a.cfg())
	if a.mcpManager != nil && len(a.mcpManager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(a.mcpManager)
	}
	systemPrompt = injectSkillCatalogPrompt(systemPrompt, a.invocationCWD())
	systemPrompt = prompt.BuildProviderSystemPromptWithConfig(systemPrompt, providerName, a.CurrentModel, a.cfg())

	if pc := a.loadProjectConfig(); pc != nil {
		systemPrompt = injectProjectConfig(systemPrompt, pc, "")
	}

	layout := parseSystemPromptLayout("")
	layout.SetStatic(systemPrompt)
	if hadPlanPrompt {
		layout.AppendDynamic(planningPrompt)
	}

	a.SystemPrompt = layout.Compose()
	injectProjectMap(a, "")
}

func (m *PromptManager) RefreshProjectPrompt(input string) {
	a := m.agent
	if a == nil {
		return
	}
	m.refreshProjectPromptWithContext(input, newPromptRefreshContext(a))
}

func (m *PromptManager) refreshProjectPromptWithContext(input string, refreshCtx promptRefreshContext) {
	a := m.agent
	if a == nil {
		return
	}
	pc := refreshCtx.projectConfig
	var newConfigBlock string
	if pc != nil {
		selection := config.SelectProjectPromptSelection(pc, input)
		newConfigBlock = prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts)
	}

	layout := parseSystemPromptLayout(a.SystemPrompt)
	layout.SetDynamic(stripProjectMapSection(layout.Dynamic))

	staticPrompt := prompt.StripProjectConfigSections(layout.Static)
	if newConfigBlock != "" {
		staticPrompt = prompt.InjectProjectConfigBlock(staticPrompt, newConfigBlock)
	}
	staticPrompt = injectSkillCatalogPrompt(staticPrompt, refreshCtx.invocationCWD)
	layout.SetStatic(withProviderPromptWrapper(a, staticPrompt))

	a.SystemPrompt = layout.Compose()
	injectProjectMapWithOverrides(a, input, projectMapInjectionOverrides{
		invocationCWD: refreshCtx.invocationCWD,
		projectConfig: refreshCtx.projectConfig,
	})
}

func withProviderPromptWrapper(a *Agent, systemPrompt string) string {
	if a == nil {
		return systemPrompt
	}
	providerName := strings.TrimSpace(a.ProviderName)
	if providerName == "" && a.CurrentProvider != nil {
		providerName = providerRuntimeNameFromProvider(a.CurrentProvider)
	}
	return prompt.BuildProviderSystemPromptWithConfig(systemPrompt, providerName, a.CurrentModel, a.cfg())
}

func (m *PromptManager) RefreshProjectPromptIfDirty(input string) {
	if m == nil || m.agent == nil {
		return
	}
	refreshCtx := newPromptRefreshContext(m.agent)
	if !m.shouldRefreshProjectPromptWithContext(input, refreshCtx) {
		return
	}
	m.refreshProjectPromptWithContext(input, refreshCtx)
}

func (m *PromptManager) ShouldRefreshProjectPrompt(input string) bool {
	if m == nil || m.agent == nil {
		return false
	}
	return m.shouldRefreshProjectPromptWithContext(input, newPromptRefreshContext(m.agent))
}

func (m *PromptManager) shouldRefreshProjectPromptWithContext(input string, refreshCtx promptRefreshContext) bool {
	a := m.agent
	if a == nil {
		return false
	}
	if a.projectMapDirty {
		return true
	}

	sources, ok := resolveProjectMapInjectionSourcesWithOverrides(a, projectMapInjectionOverrides{
		invocationCWD: refreshCtx.invocationCWD,
		projectConfig: refreshCtx.projectConfig,
	})
	if !ok {
		return false
	}
	if stateKey := currentProjectMapStateKey(a, sources.rootPath); stateKey != "" && stateKey != a.projectMapStateKey {
		return true
	}

	baseKey := buildProjectMapBaseKey(
		a,
		sources.cfg,
		calcProjectMapBudget(a, sources.cfg, a.projectMapFileCount, a.projectMapSymbolCount),
		a.projectMapFileCount,
		a.projectMapSymbolCount,
	)
	if a.projectMapBaseKey != baseKey {
		return true
	}

	focusPaths := extractProjectMapFocusPaths(sources.cwd, sources.rootPath, input, projectMapFocusMaxPaths)
	if a.projectMapFocusKey != buildProjectMapFocusKey(focusPaths) {
		return true
	}

	return a.projectMap == nil || a.projectMapBaseSection == "" || a.projectMapSection == ""
}

func (m *PromptManager) InvalidateProjectMap() {
	a := m.agent
	if a == nil {
		return
	}

	a.projectMap = nil
	a.projectMapRootPath = ""
	a.projectMapIgnoreKey = ""
	a.projectMapStateKey = ""
	a.projectMapWatchDirs = nil
	a.projectMapBaseSection = ""
	a.projectMapFocusSection = ""
	a.projectMapSection = ""
	a.projectMapBaseKey = ""
	a.projectMapFocusKey = ""
	a.projectMapDirty = true
}

func (m *PromptManager) DebugString() string {
	a := m.agent
	if a == nil {
		return "<nil>"
	}
	return fmt.Sprintf("PromptManager(model=%s, dirty=%t, files=%d, symbols=%d)", a.CurrentModel, a.projectMapDirty, a.projectMapFileCount, a.projectMapSymbolCount)
}
