package agent

import (
	"fmt"
	"strings"

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

	systemPrompt := m.buildStaticPrompt(promptStaticBuildInput{
		invocationCWD:      a.invocationCWD(),
		projectConfigBlock: buildProjectConfigPromptBlock(a.loadProjectConfig(), ""),
	})

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
	newConfigBlock := buildProjectConfigPromptBlock(pc, input)

	layout := parseSystemPromptLayout(a.SystemPrompt)
	layout.SetStatic(m.buildStaticPromptForRefresh(layout.Static, promptStaticBuildInput{
		invocationCWD:      refreshCtx.invocationCWD,
		projectConfigBlock: newConfigBlock,
	}))

	a.SystemPrompt = layout.Compose()
	injectProjectMapWithOverrides(a, input, projectMapInjectionOverrides{
		invocationCWD: refreshCtx.invocationCWD,
		projectConfig: refreshCtx.projectConfig,
	})
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

	resetProjectMapRuntimeCounts(a)
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
