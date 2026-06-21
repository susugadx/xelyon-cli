package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/investigation"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

type toolSurfacePhase string

const (
	toolSurfacePhaseNormal toolSurfacePhase = "normal"
	toolSurfacePhasePlan   toolSurfacePhase = "plan"
)

type toolVisibilityOptions struct {
	allowSubAgents bool
}

// toolVisibilityPolicy は current surface の visible/excluded tool 契約を表す。
// investigationSurface を source of truth とし、そこから prompt/visibility/help を導出する。
type toolVisibilityPolicy struct {
	editToolMode         string
	excludedTools        []string
	investigationSurface investigation.Surface
}

type normalModeRecoveryPromptKind string

const (
	normalModeRecoveryPromptDirectExecution normalModeRecoveryPromptKind = "direct_execution"
	normalModeRecoveryPromptStopPlanning    normalModeRecoveryPromptKind = "stop_planning"
	normalModeRecoveryPromptNoTextPlan      normalModeRecoveryPromptKind = "no_text_plan"
)

func resolveToolVisibilityPolicyWithConfig(providerName string, modelName string, cfg *config.Config, phase toolSurfacePhase, opts toolVisibilityOptions) toolVisibilityPolicy {
	editToolMode := string(prompt.ResolveEditToolModeWithConfig(providerName, modelName, cfg))
	return newToolVisibilityPolicy(editToolMode, phase, opts)
}

func newToolVisibilityPolicy(editToolMode string, phase toolSurfacePhase, opts toolVisibilityOptions) toolVisibilityPolicy {
	surface := investigation.ResolveSurface(editToolMode == EditToolModeLegacy, true)
	policy := toolVisibilityPolicy{
		editToolMode:         editToolMode,
		investigationSurface: surface,
	}

	excluded := baseToolSurfaceExclusions(editToolMode, phase)
	excluded = appendUniqueStrings(excluded, investigationToolSurfaceExclusions(surface)...)
	if !opts.allowSubAgents {
		excluded = appendUniqueStrings(excluded, "spawn_agent", "wait_agent")
	}
	policy.excludedTools = excluded
	return policy
}

func (p toolVisibilityPolicy) excluded() []string {
	return append([]string(nil), p.excludedTools...)
}

func (p toolVisibilityPolicy) normalModeRecoveryPrompt(kind normalModeRecoveryPromptKind) string {
	toolExamples := p.normalModeRecoveryToolExamples()
	switch kind {
	case normalModeRecoveryPromptStopPlanning:
		return "[SYSTEM] STOP planning. Pick the FIRST change and execute it NOW using the appropriate visible tools (" + toolExamples + ", etc). One tool call, no explanation."
	case normalModeRecoveryPromptNoTextPlan:
		return "[SYSTEM] Do NOT output plans as numbered text. Execute the required changes directly using visible tools (" + toolExamples + ", etc)."
	default:
		return "[SYSTEM] You are in NORMAL MODE. Do NOT output JSON directly. Execute the required changes directly using visible tools (" + toolExamples + ", etc)."
	}
}

func (p toolVisibilityPolicy) normalModeRecoveryToolExamples() string {
	if p.investigationSurface.AllowsLowLevelOverrides() {
		return "gather_context, read_file, str_replace"
	}
	if p.investigationSurface.HasVisibleReadFile() {
		return "gather_context, read_file, apply_patch"
	}
	return "gather_context, apply_patch"
}

func baseToolSurfaceExclusions(editToolMode string, phase toolSurfacePhase) []string {
	excluded := make([]string, 0, 6)
	if phase == toolSurfacePhaseNormal {
		excluded = appendUniqueStrings(excluded, prompt.PlanningToolNames...)
	}

	excluded = filterStrings(excluded, "apply_patch", "str_replace", "write_file", "delete_file")
	if editToolMode == EditToolModeLegacy {
		return appendUniqueStrings(excluded, "apply_patch")
	}
	return appendUniqueStrings(excluded, "str_replace", "write_file", "delete_file")
}

func investigationToolSurfaceExclusions(surface investigation.Surface) []string {
	excluded := make([]string, 0, 3)
	for _, toolName := range []string{
		investigation.ToolSearchCode,
		investigation.ToolReadFile,
		investigation.ToolListDir,
	} {
		if !surface.ToolRole(toolName).Visible() {
			excluded = append(excluded, toolName)
		}
	}
	return excluded
}

func (a *Agent) currentToolSurfacePhase() toolSurfacePhase {
	if a != nil && a.PlanModeEnabled {
		return toolSurfacePhasePlan
	}
	return toolSurfacePhaseNormal
}

// mergeSurfaceManagedExcludedTools combines current-surface exclusions with any
// runtime-specific exclusions that callers already applied to the registry.
// Surface sync owns only phase-managed exclusions; it must not re-expose tools
// that runtime code intentionally hid for other reasons.
func mergeSurfaceManagedExcludedTools(runtimeExcluded []string, policy toolVisibilityPolicy) []string {
	return mergeSurfaceManagedExcludedToolsWithRuntimeExclusions(runtimeExcluded, policy, nil, nil)
}

func mergeSurfaceManagedExcludedToolsWithRuntimeExclusions(runtimeExcluded []string, policy toolVisibilityPolicy, previousManaged []string, currentExtra []string) []string {
	managed := appendUniqueStrings(surfaceManagedExcludedToolNames(), previousManaged...)
	extraRuntimeExcluded := filterStrings(
		append([]string(nil), runtimeExcluded...),
		managed...,
	)
	excluded := appendUniqueStrings(extraRuntimeExcluded, policy.excluded()...)
	return appendUniqueStrings(excluded, currentExtra...)
}

func surfaceManagedExcludedToolNames() []string {
	managed := make([]string, 0, 16)
	for _, editToolMode := range []string{EditToolModeApplyPatch, EditToolModeLegacy} {
		managed = appendUniqueStrings(
			managed,
			newToolVisibilityPolicy(editToolMode, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).excluded()...,
		)
		managed = appendUniqueStrings(
			managed,
			newToolVisibilityPolicy(editToolMode, toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true}).excluded()...,
		)
	}
	return managed
}

// syncCurrentSurfaceToolVisibility updates the runtime registry to match the
// current provider/model/plan surface immediately. This keeps `/help` and other
// command-time surface inspection in sync even before the next chat request.
func (a *Agent) syncCurrentSurfaceToolVisibility() {
	if a == nil {
		return
	}
	previousSurface, _ := a.refreshMCPToolSurface()
	a.syncCurrentSurfaceToolVisibilityWithPreviousBudget(previousSurface.omittedExportedNames())
}

func (a *Agent) syncCurrentSurfaceToolVisibilityWithPreviousBudget(previousBudgetExcluded []string) {
	if a == nil {
		return
	}
	policy := a.toolVisibilityPolicy(a.currentToolSurfacePhase(), toolVisibilityOptions{allowSubAgents: true})
	a.registry().SetExcludedTools(mergeSurfaceManagedExcludedToolsWithRuntimeExclusions(
		a.registry().GetExcludedTools(),
		policy,
		previousBudgetExcluded,
		a.currentMCPBudgetExcludedToolNames(),
	))
}

func (a *Agent) toolVisibilityPolicy(phase toolSurfacePhase, opts toolVisibilityOptions) toolVisibilityPolicy {
	if a == nil {
		return newToolVisibilityPolicy(EditToolModeApplyPatch, phase, opts)
	}
	return resolveToolVisibilityPolicyWithConfig(a.ProviderName, a.CurrentModel, a.cfg(), phase, opts)
}
