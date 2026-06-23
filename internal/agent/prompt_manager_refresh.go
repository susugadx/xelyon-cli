package agent

func (m *PromptManager) RefreshProjectPrompt(input string) {
	a := m.agent
	if a == nil {
		return
	}

	bundle := a.loadProjectInstructionBundleCachedWithInput(false, input)
	m.rebuildProjectPromptWithResolvedBundle(input, bundle, true)
}

func (m *PromptManager) RefreshProjectPromptIfDirty(input string) {
	if m == nil {
		return
	}
	decision := m.ProjectPromptRefreshDecision(input)
	if !decision.NeedRefresh {
		return
	}
	m.RefreshProjectPrompt(input)
}

func (m *PromptManager) ShouldRefreshProjectPrompt(input string) bool {
	return m.ProjectPromptRefreshDecision(input).NeedRefresh
}

func (m *PromptManager) ProjectPromptRefreshDecision(input string) projectPromptRefreshDecision {
	a := m.agent
	if a == nil {
		return projectPromptRefreshDecision{Reason: refreshReasonNoAgent}
	}
	if a.projectMapDirty {
		return projectPromptRefreshDecision{NeedRefresh: true, Reason: refreshReasonDirtyFlag}
	}
	if decision, ok := evaluateProjectInstructionRefresh(a, input); ok {
		return decision
	}

	sources, reason, ok := resolveProjectMapInjectionSources(a, projectMapSourceResolveOptions{
		allowBundleLoad: false,
	})
	if !ok {
		if isProjectMapSourceUnavailableReason(reason) && a.hasProjectMapState() {
			return projectPromptRefreshDecision{NeedRefresh: true, Reason: reason}
		}
		return projectPromptRefreshDecision{Reason: reason}
	}

	state := projectPromptRefreshState{
		agent:   a,
		input:   input,
		sources: sources,
		decision: projectPromptRefreshDecision{
			RootPath: sources.rootPath,
		},
	}
	return state.evaluate()
}

func (s *projectPromptRefreshState) evaluate() projectPromptRefreshDecision {
	if s == nil {
		return projectPromptRefreshDecision{Reason: refreshReasonNoAgent}
	}
	if s.evaluateStateKey() {
		return s.decision
	}
	if s.evaluateBaseKey() {
		return s.decision
	}
	if s.evaluateFocusKey() {
		return s.decision
	}
	if s.evaluateCachedSection() {
		return s.decision
	}
	s.decision.Reason = refreshReasonNoChange
	return s.decision
}

func (s *projectPromptRefreshState) markNeedRefresh(reason projectPromptRefreshReason) bool {
	s.decision.NeedRefresh = true
	s.decision.Reason = reason
	return true
}

func (s *projectPromptRefreshState) evaluateStateKey() bool {
	if s == nil || s.agent == nil {
		return false
	}
	stateKey := currentProjectMapStateKey(s.agent, s.sources.rootPath)
	if stateKey == "" {
		return false
	}
	s.decision.StateKey = stateKey
	if stateKey != s.agent.projectMapStateKey {
		return s.markNeedRefresh(refreshReasonStateKeyChanged)
	}
	return false
}

func evaluateProjectInstructionRefresh(agent *Agent, input string) (projectPromptRefreshDecision, bool) {
	if agent == nil || !agent.projectInstructionBundleLoaded {
		return projectPromptRefreshDecision{}, false
	}
	cacheKey := newProjectInstructionBundleCache(agent).currentKey(input)
	if cacheKey == "" {
		return projectPromptRefreshDecision{}, false
	}
	if cacheKey == agent.projectInstructionBundleKey {
		return projectPromptRefreshDecision{}, false
	}
	return projectPromptRefreshDecision{
		NeedRefresh: true,
		Reason:      refreshReasonInstructionChanged,
	}, true
}

func (s *projectPromptRefreshState) evaluateBaseKey() bool {
	if s == nil || s.agent == nil {
		return false
	}
	baseKey := buildProjectMapBaseKey(
		s.agent,
		s.sources.cfg,
		calcProjectMapBudget(s.agent, s.sources.cfg, s.agent.projectMapFileCount, s.agent.projectMapSymbolCount),
		s.agent.projectMapFileCount,
		s.agent.projectMapSymbolCount,
	)
	s.decision.BaseKey = baseKey
	if s.agent.projectMapBaseKey != baseKey {
		return s.markNeedRefresh(refreshReasonBaseKeyChanged)
	}
	return false
}

func (s *projectPromptRefreshState) evaluateFocusKey() bool {
	if s == nil || s.agent == nil {
		return false
	}
	focusPaths := extractProjectMapFocusPaths(s.sources.cwd, s.sources.rootPath, s.input, projectMapFocusMaxPaths)
	focusKey := buildProjectMapFocusKey(focusPaths)
	s.decision.FocusKey = focusKey
	if s.agent.projectMapFocusKey != focusKey {
		return s.markNeedRefresh(refreshReasonFocusKeyChanged)
	}
	return false
}

func (s *projectPromptRefreshState) evaluateCachedSection() bool {
	if s == nil || s.agent == nil {
		return false
	}
	if s.agent.projectMap == nil || s.agent.projectMapBaseSection == "" || s.agent.projectMapSection == "" {
		return s.markNeedRefresh(refreshReasonMissingCachedSection)
	}
	return false
}

func (m *PromptManager) InvalidateProjectMap() {
	a := m.agent
	if a == nil {
		return
	}
	a.invalidateProjectInstructionBundleCache()
	a.resetProjectMapState()
}
