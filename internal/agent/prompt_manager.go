package agent

import (
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/config"
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
	refreshReasonInstructionChanged   projectPromptRefreshReason = "instruction_changed"
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

func (m *PromptManager) DebugString() string {
	a := m.agent
	if a == nil {
		return "<nil>"
	}
	return fmt.Sprintf("PromptManager(model=%s, dirty=%t, files=%d, symbols=%d)", a.CurrentModel, a.projectMapDirty, a.projectMapFileCount, a.projectMapSymbolCount)
}
