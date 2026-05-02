package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
)

type PromptManager struct {
	agent *Agent
}

type systemPromptRebuildMode string

const (
	systemPromptRebuildModeBase        systemPromptRebuildMode = "base"
	systemPromptRebuildModeProjectOnly systemPromptRebuildMode = "project_only"
)

type systemPromptBundleResolveMode string

const (
	systemPromptBundleResolveCached systemPromptBundleResolveMode = "cached"
	systemPromptBundleResolveForced systemPromptBundleResolveMode = "forced"
	systemPromptBundleResolveGiven  systemPromptBundleResolveMode = "given"
)

type systemPromptRebuildRequest struct {
	mode             systemPromptRebuildMode
	input            string
	bundle           *config.ProjectInstructionBundle
	bundleMode       systemPromptBundleResolveMode
	injectProjectMap bool
}

type projectPromptRefreshReason string

const (
	refreshReasonNoAgent              projectPromptRefreshReason = "no_agent"
	refreshReasonDirtyFlag            projectPromptRefreshReason = "dirty_flag"
	refreshReasonProjectMapDisabled   projectPromptRefreshReason = "project_map_disabled"
	refreshReasonRipgrepUnavailable   projectPromptRefreshReason = "ripgrep_unavailable"
	refreshReasonCWDUnavailable       projectPromptRefreshReason = "cwd_unavailable"
	refreshReasonStateKeyChanged      projectPromptRefreshReason = "state_key_changed"
	refreshReasonBaseKeyChanged       projectPromptRefreshReason = "base_key_changed"
	refreshReasonFocusKeyChanged      projectPromptRefreshReason = "focus_key_changed"
	refreshReasonMissingCachedSection projectPromptRefreshReason = "missing_cached_section"
	refreshReasonNoChange             projectPromptRefreshReason = "no_change"
)

type projectPromptRefreshDecision struct {
	NeedRefresh bool
	Reason      projectPromptRefreshReason
	RootPath    string
	StateKey    string
	BaseKey     string
	FocusKey    string
}

type projectPromptRefreshState struct {
	agent    *Agent
	input    string
	sources  projectMapInjectionSources
	decision projectPromptRefreshDecision
}

func newPromptManager(agent *Agent) *PromptManager {
	return &PromptManager{agent: agent}
}

func (a *Agent) promptManager() *PromptManager {
	if a == nil {
		return nil
	}
	if a.promptMgr == nil {
		a.promptMgr = newPromptManager(a)
	}
	return a.promptMgr
}

func (m *PromptManager) RebuildSystemPromptForCurrentProvider() {
	m.RebuildSystemPrompt(systemPromptRebuildRequest{
		mode:             systemPromptRebuildModeBase,
		bundleMode:       systemPromptBundleResolveForced,
		injectProjectMap: true,
	})
}

func (m *PromptManager) InitializeProjectInstructions(opts projectInstructionApplyOptions) {
	a := m.agent
	if a == nil {
		return
	}
	bundle := a.loadProjectInstructionBundleCached(true)
	applyProjectInstructionBundle(a, bundle, opts.showStatus)
	m.rebuildProjectPromptWithResolvedBundle(opts.projectMapInput, bundle, opts.injectProjectMap)
}

func (m *PromptManager) RebuildSystemPrompt(req systemPromptRebuildRequest) {
	a := m.agent
	if a == nil {
		return
	}
	if req.mode == systemPromptRebuildModeBase && a.CurrentProvider == nil {
		return
	}

	if req.mode == systemPromptRebuildModeBase {
		a.SystemPrompt = m.buildBaseSystemPromptForCurrentProvider()
	} else {
		a.SystemPrompt = stripProjectMapSection(prompt.StripProjectConfigSections(a.SystemPrompt))
	}

	bundle := m.resolveBundleForRebuild(req)
	if bundle != nil {
		m.InjectProjectInstructions(bundle, req.input)
	}
	if req.injectProjectMap {
		injectProjectMap(a, req.input)
	}
}

func (m *PromptManager) resolveBundleForRebuild(req systemPromptRebuildRequest) *config.ProjectInstructionBundle {
	a := m.agent
	if a == nil {
		return nil
	}
	switch req.bundleMode {
	case systemPromptBundleResolveGiven:
		return req.bundle
	case systemPromptBundleResolveForced:
		return a.loadProjectInstructionBundleCached(true)
	case systemPromptBundleResolveCached:
		return a.loadProjectInstructionBundleCached(false)
	default:
		return a.loadProjectInstructionBundleCached(false)
	}
}

func (m *PromptManager) buildBaseSystemPromptForCurrentProvider() string {
	a := m.agent
	if a == nil || a.CurrentProvider == nil {
		return ""
	}

	planningPrompt := promptplan.BuildPlanningPrompt()
	hadPlanPrompt := strings.Contains(a.SystemPrompt, planningPrompt)

	providerName := a.ProviderName
	if providerName == "" {
		providerName = providerRuntimeNameFromProvider(a.CurrentProvider)
	}
	systemPrompt := prompt.GetSystemPromptForProviderWithConfig(providerName, a.CurrentModel, a.cfg())
	if a.mcpManager != nil && len(a.mcpManager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(a.mcpManager)
	}
	systemPrompt = prompt.BuildProviderSystemPromptWithConfig(systemPrompt, providerName, a.CurrentModel, a.cfg())

	if hadPlanPrompt {
		systemPrompt += api.SystemPromptCacheBoundary + planningPrompt
	}

	return systemPrompt
}

func (m *PromptManager) RefreshProjectPrompt(input string) {
	a := m.agent
	if a == nil {
		return
	}

	bundle := a.loadProjectInstructionBundleCached(false)
	m.rebuildProjectPromptWithResolvedBundle(input, bundle, true)
}

func (m *PromptManager) rebuildProjectPromptWithResolvedBundle(input string, bundle *config.ProjectInstructionBundle, injectProjectMap bool) {
	m.RebuildSystemPrompt(systemPromptRebuildRequest{
		mode:             systemPromptRebuildModeProjectOnly,
		input:            input,
		bundle:           bundle,
		bundleMode:       systemPromptBundleResolveGiven,
		injectProjectMap: injectProjectMap,
	})
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

	sources, reason, ok := resolveProjectMapInjectionSources(a, projectMapSourceResolveOptions{
		allowBundleLoad: false,
	})
	if !ok {
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
	if state.evaluateStateKey() {
		return state.decision
	}
	if state.evaluateBaseKey() {
		return state.decision
	}
	if state.evaluateFocusKey() {
		return state.decision
	}
	if state.evaluateCachedSection() {
		return state.decision
	}
	state.decision.Reason = refreshReasonNoChange
	return state.decision
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

func (m *PromptManager) DebugString() string {
	a := m.agent
	if a == nil {
		return "<nil>"
	}
	return fmt.Sprintf("PromptManager(model=%s, dirty=%t, files=%d, symbols=%d)", a.CurrentModel, a.projectMapDirty, a.projectMapFileCount, a.projectMapSymbolCount)
}

// InjectProjectInstructions は現在の SystemPrompt に project instructions を注入する。
func (m *PromptManager) InjectProjectInstructions(bundle *config.ProjectInstructionBundle, input string) {
	a := m.agent
	if a == nil {
		return
	}
	a.SystemPrompt = injectProjectInstructionBundle(a.SystemPrompt, bundle, input)
}

// StripProjectMapSection は SystemPrompt から project map セクションを除去する。
func (m *PromptManager) StripProjectMapSection() {
	a := m.agent
	if a == nil {
		return
	}
	a.SystemPrompt = stripProjectMapSection(a.SystemPrompt)
}

// AppendProjectMapSection は SystemPrompt に project map セクションを追加する。
func (m *PromptManager) AppendProjectMapSection(section string) {
	a := m.agent
	if a == nil {
		return
	}
	a.SystemPrompt = appendProjectMapSection(a.SystemPrompt, section)
}
